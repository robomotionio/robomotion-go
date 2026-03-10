package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	hclog "github.com/hashicorp/go-hclog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/robomotionio/robomotion-go/proto"
)

const (
	defaultSessionTimeout = 30 * time.Minute
	daemonPollInterval    = 100 * time.Millisecond
	daemonPollMaxWait     = 3 * time.Second
	maxSessionMsgSize     = 64 * 1024 * 1024 // 64 MB, matches debug/attach.go
)

// sessionMetadata is written to <session-id>.json alongside the socket/port file.
type sessionMetadata struct {
	SessionID    string   `json:"session_id"`
	PID          int      `json:"pid"`
	Namespace    string   `json:"namespace,omitempty"`
	CreatedAt    string   `json:"created_at"`
	LastActivity string   `json:"last_activity"`
	Nodes        []string `json:"nodes,omitempty"` // active node guids
}

// sessionTimeoutInterceptor resets the inactivity timer on each gRPC call.
func sessionTimeoutInterceptor(timer *time.Timer, timeout time.Duration, mu *sync.Mutex) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		mu.Lock()
		timer.Reset(timeout)
		mu.Unlock()
		return handler(ctx, req)
	}
}

// RunSessionDaemon starts a plain gRPC server reusing the existing Node service.
// Called internally via --session-daemon <id> [flags].
func RunSessionDaemon(sessionID string, timeout time.Duration, vaultID, itemID string) {
	// Set up CLI runtime helper as the global client
	cliHelper := NewCLIRuntimeHelper()

	if vaultID != "" && itemID != "" {
		vaultClient, err := NewCLIVaultClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, `{"error":"vault auth: %v"}`+"\n", err)
			os.Exit(1)
		}
		creds, err := vaultClient.FetchVaultItem(vaultID, itemID)
		if err != nil {
			fmt.Fprintf(os.Stderr, `{"error":"vault fetch: %v"}`+"\n", err)
			os.Exit(1)
		}
		cliHelper.SetCredentials(creds)
	}

	client = cliHelper

	// Pre-close initReady so GRPCServer.OnCreate doesn't block waiting for Init
	initReady = make(chan struct{})
	close(initReady)

	// Register node factories
	RegisterFactories()

	// Create platform-specific listener
	lis, err := sessionListen(sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, `{"error":"listen: %v"}`+"\n", err)
		os.Exit(1)
	}

	// Inactivity timeout
	var timerMu sync.Mutex
	timer := time.NewTimer(timeout)

	sessionMode = true

	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxSessionMsgSize),
		grpc.MaxSendMsgSize(maxSessionMsgSize),
		grpc.UnaryInterceptor(sessionTimeoutInterceptor(timer, timeout, &timerMu)),
	)
	proto.RegisterNodeServer(grpcServer, &GRPCServer{Impl: &Node{}})

	// Write metadata
	writeSessionMetadata(sessionID, lis)

	// Timeout goroutine — graceful stop on inactivity
	go func() {
		<-timer.C
		hclog.Default().Info("session.timeout", "session_id", sessionID)
		closeAllSessionNodes()
		grpcServer.GracefulStop()
		sessionCleanup(sessionID)
	}()

	// Also listen for done signal (triggered when nc reaches 0 after OnClose)
	go func() {
		<-done
		timer.Stop()
		grpcServer.GracefulStop()
		sessionCleanup(sessionID)
	}()

	// Serve blocks until GracefulStop
	if err := grpcServer.Serve(lis); err != nil {
		hclog.Default().Info("session.serve", "err", err)
	}
}

// RunSessionClient connects to an existing session daemon and sends a command.
func RunSessionClient(sessionID, commandName string, flags map[string]string) {
	addr := sessionDialAddr(sessionID)
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxSessionMsgSize),
			grpc.MaxCallSendMsgSize(maxSessionMsgSize),
		),
	)
	if err != nil {
		cliError("session dial failed: %v", err)
		return
	}
	defer conn.Close()

	nodeClient := proto.NewNodeClient(conn)
	ctx := context.Background()

	// Use unique guid for each call so Custom-scope variables are fresh per invocation
	guid := commandName + "-" + generateSessionID()

	// Extract vault flags before building message context
	vaultID := flags["vault-id"]
	itemID := flags["item-id"]
	delete(flags, "vault-id")
	delete(flags, "item-id")

	// Resolve node type for config and message building
	RegisterFactories()
	commands := buildCommandMap()
	cmd, ok := commands[commandName]
	if !ok {
		cliError("unknown command %q", commandName)
		return
	}

	// Read metadata to check if this node was already created
	meta, _ := readSessionMetadata(sessionID)
	nodeExists := false
	if meta != nil {
		for _, g := range meta.Nodes {
			if g == guid {
				nodeExists = true
				break
			}
		}
	}

	// Build CLI context: split flags into message context vs config patches
	msgData, configPatches := buildCLIContext(cmd.nodeType, flags)

	if !nodeExists {
		// Build config JSON (same as cli.go)
		nodeConfig := map[string]interface{}{
			"guid": guid,
			"name": commandName,
		}
		injectVariableConfig(cmd.nodeType, nodeConfig)
		if vaultID != "" && itemID != "" {
			injectCredentialConfig(cmd.nodeType, nodeConfig, vaultID, itemID)
		}
		for k, v := range configPatches {
			nodeConfig[k] = v
		}

		configJSON, _ := json.Marshal(nodeConfig)
		_, err := nodeClient.OnCreate(ctx, &proto.OnCreateRequest{
			Name:   cmd.nodeID,
			Config: configJSON,
		})
		if err != nil {
			cliError("session OnCreate failed: %v", err)
			return
		}

		// Update metadata with new node guid
		if meta != nil {
			meta.Nodes = append(meta.Nodes, guid)
			meta.LastActivity = time.Now().UTC().Format(time.RFC3339)
			saveSessionMetadata(sessionID, meta)
		}
	}

	// Build message data for Message-scope variables
	msgJSON, _ := json.Marshal(msgData)

	// Compress message data (GRPCServer.OnMessage expects gzip-compressed data)
	compressed, compErr := Compress(msgJSON)
	if compErr != nil {
		cliError("compress failed: %v", compErr)
		return
	}

	// Call OnMessage
	resp, err := nodeClient.OnMessage(ctx, &proto.OnMessageRequest{
		Guid:      guid,
		InMessage: compressed,
	})
	if err != nil {
		cliError("session OnMessage failed: %v", err)
		return
	}

	// Print response with session_id included
	var result map[string]interface{}
	if resp.OutMessage != nil {
		if err := json.Unmarshal(resp.OutMessage, &result); err != nil {
			// Not valid JSON — wrap it
			result = map[string]interface{}{"result": string(resp.OutMessage)}
		}
	} else {
		result = map[string]interface{}{"status": "completed"}
	}
	result["session_id"] = sessionID
	out, _ := json.Marshal(result)
	fmt.Println(string(out))
}

// CloseSession sends OnClose for all nodes and the daemon exits.
func CloseSession(sessionID string) {
	meta, err := readSessionMetadata(sessionID)
	if err != nil {
		cliError("cannot read session %s: %v", sessionID, err)
		return
	}

	addr := sessionDialAddr(sessionID)
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxSessionMsgSize),
			grpc.MaxCallSendMsgSize(maxSessionMsgSize),
		),
	)
	if err != nil {
		cliError("session dial failed: %v", err)
		return
	}
	defer conn.Close()

	nodeClient := proto.NewNodeClient(conn)
	ctx := context.Background()

	// Close all active nodes
	for _, guid := range meta.Nodes {
		_, err := nodeClient.OnClose(ctx, &proto.OnCloseRequest{Guid: guid})
		if err != nil {
			fmt.Fprintf(os.Stderr, `{"warning":"OnClose %s: %v"}`+"\n", guid, err)
		}
	}

	result, _ := json.Marshal(map[string]string{"status": "closed"})
	fmt.Println(string(result))
}

// StartDaemonProcess forks the current binary as a session daemon, waits for
// the socket/port file to appear, then runs the first command as a client.
func StartDaemonProcess(sessionID string, timeout time.Duration, vaultID, itemID string, origArgs []string) {
	// Build daemon args: <binary> --session-daemon <id> [--session-timeout=X] [--vault-id=X] [--item-id=X]
	exe, err := os.Executable()
	if err != nil {
		cliError("cannot find executable: %v", err)
		return
	}

	args := []string{exe, "--session-daemon", sessionID}
	if timeout != defaultSessionTimeout {
		args = append(args, fmt.Sprintf("--session-timeout=%s", timeout))
	}
	if vaultID != "" {
		args = append(args, fmt.Sprintf("--vault-id=%s", vaultID))
	}
	if itemID != "" {
		args = append(args, fmt.Sprintf("--item-id=%s", itemID))
	}

	// Detached process
	attr := &os.ProcAttr{
		Dir:   ".",
		Env:   os.Environ(),
		Files: []*os.File{os.Stdin, os.Stderr, os.Stderr}, // stdout→stderr for daemon logs
	}

	proc, err := os.StartProcess(exe, args, attr)
	if err != nil {
		cliError("failed to start session daemon: %v", err)
		return
	}
	proc.Release()

	// Poll for socket/port file to appear
	deadline := time.Now().Add(daemonPollMaxWait)
	for time.Now().Before(deadline) {
		if sessionReady(sessionID) {
			return
		}
		time.Sleep(daemonPollInterval)
	}

	cliError("session daemon did not start within %s", daemonPollMaxWait)
}

// generateSessionID returns 8 random hex characters.
func generateSessionID() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)
	}
	return hex.EncodeToString(b)
}

// writeSessionMetadata writes the metadata JSON file for a session.
func writeSessionMetadata(sessionID string, lis net.Listener) {
	dir := sessionDir()
	os.MkdirAll(dir, 0700)

	config := ReadConfigFile()
	namespace := config.Get("namespace").String()

	meta := sessionMetadata{
		SessionID:    sessionID,
		PID:          os.Getpid(),
		Namespace:    namespace,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
		LastActivity: time.Now().UTC().Format(time.RFC3339),
	}

	data, _ := json.MarshalIndent(meta, "", "  ")
	metaPath := filepath.Join(dir, sessionID+".json")
	os.WriteFile(metaPath, data, 0600)
}

// readSessionMetadata reads the metadata JSON file for a session.
func readSessionMetadata(sessionID string) (*sessionMetadata, error) {
	metaPath := filepath.Join(sessionDir(), sessionID+".json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}
	var meta sessionMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// saveSessionMetadata writes updated metadata back to disk.
func saveSessionMetadata(sessionID string, meta *sessionMetadata) {
	data, _ := json.MarshalIndent(meta, "", "  ")
	metaPath := filepath.Join(sessionDir(), sessionID+".json")
	os.WriteFile(metaPath, data, 0600)
}

// closeAllSessionNodes calls OnClose on all active handlers in the pool.
func closeAllSessionNodes() {
	guids := listNodeHandlerGUIDs()
	for _, guid := range guids {
		node := GetNodeHandler(guid)
		if node != nil {
			node.Handler.OnClose()
			atomic.AddInt32(&nc, -1)
		}
	}
}

// sessionReady checks if the session socket/port file exists.
func sessionReady(sessionID string) bool {
	addr := sessionDialAddr(sessionID)
	if addr == "" {
		return false
	}
	return true
}

// parseSessionTimeout parses timeout string, returns default if empty/invalid.
func parseSessionTimeout(s string) time.Duration {
	if s == "" {
		return defaultSessionTimeout
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultSessionTimeout
	}
	return d
}
