package debug

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	gops "github.com/mitchellh/go-ps"
	"github.com/robomotionio/robomotion-go/proto"
	"google.golang.org/grpc"
)

func getRPCAddr() string {
	tabs := GetNetStatPorts(SS_LISTENING, "robomotion-runner")

	switch len(tabs) {
	case 0:
		return ""
	case 1:
		return tabs[0].LocalAddress
	default:
		return selectTab(tabs)
	}
}

func selectTab(tabs []*SockTabEntry) string {
	count := len(tabs)

	robots := ""
	for i, tab := range tabs {
		addr := tab.LocalAddress
		name := getRobotName(addr)
		robots += fmt.Sprintf("%d) %s\n", i+1, name)
	}

	flags := log.Flags()
	defer log.SetFlags(flags)

	log.SetFlags(0)
	log.Printf("\nFound %d robots running on the machine:\n", count)
	log.Printf("%s", robots)

	selected := 0
	log.Printf("Please select a robot to attach (1-%d): ", count)
	for {
		fmt.Scanf("%d", &selected)
		if selected > 0 && selected <= count {
			return tabs[selected-1].LocalAddress
		}
	}
}

func getRobotName(addr string) string {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalln(err)
	}
	defer conn.Close()

	runnerCli := proto.NewRunnerClient(conn)
	resp, err := runnerCli.RobotName(context.Background(), &proto.Null{})
	if err != nil {
		log.Println(err)
		return ""
	}

	return resp.RobotName
}

type SocketState string
type List struct {
	arr interface{}
}

type SockTabEntry struct {
	Process      gops.Process
	LocalAddress string
}

const (
	SS_UNKNOWN     SocketState = "UNKNOWN"
	SS_CLOSED      SocketState = ""
	SS_LISTENING   SocketState = "LISTENING"
	SS_SYN_SENT    SocketState = "SYN_SENT"
	SS_SYN_RECV    SocketState = "SYN_RECV"
	SS_ESTABLISHED SocketState = "ESTABLISHED"
	SS_FIN_WAIT1   SocketState = "FIN_WAIT1"
	SS_FIN_WAIT2   SocketState = "FIN_WAIT2"
	SS_CLOSE_WAIT  SocketState = "CLOSE_WAIT"
	SS_CLOSING     SocketState = "CLOSING"
	SS_LAST_ACK    SocketState = "LAST_ACK"
	SS_TIME_WAIT   SocketState = "TIME_WAIT"
	SS_DELETE_TCB  SocketState = "DELETE_TCB"
)

func NewList(d interface{}) *List {
	arrType := reflect.TypeOf(d)
	kind := arrType.Kind()
	if kind != reflect.Array && kind != reflect.Slice {
		log.Fatalf("Expected an array/slice. Have %+v.", kind.String())
	}

	return &List{arr: d}
}

func (l *List) Filter(predicate func(interface{}) bool) *List {
	arr := reflect.ValueOf(l.arr)
	arrType := arr.Type()
	resultSliceType := reflect.SliceOf(arrType.Elem())
	filtered := reflect.MakeSlice(resultSliceType, 0, 0)

	for i := 0; i < arr.Len(); i++ {
		elem := arr.Index(i)
		if predicate(elem.Interface()) {
			filtered = reflect.Append(filtered, elem)
		}
	}

	return NewList(filtered.Interface())
}

func (l *List) First() interface{} {
	arr := reflect.ValueOf(l.arr)
	if arr.Len() > 0 {
		return arr.Index(0).Interface()
	}
	return nil
}

func GetNetStatPorts(state SocketState, processName string) []*SockTabEntry {
	tabs := []*SockTabEntry{}

	cmd := exec.Command("netstat", "-a", "-n", "-o")
	out, err := cmd.Output()
	if err != nil {
		log.Println(err)
		return nil
	}

	procs, err := gops.Processes()
	if err != nil {
		log.Println(err)
		return nil
	}

	processList := NewList(procs)
	processList = processList.Filter(func(p interface{}) bool {
		proc := p.(gops.Process)
		a := strings.ToLower(proc.Executable())
		b := strings.ToLower(processName)
		return strings.Contains(a, b)
	})

	rr := regexp.MustCompile(`\r\n`)
	rows := rr.Split(string(out), -1)

	for _, row := range rows {
		if row == "" {
			continue
		}

		tr := regexp.MustCompile(`\s+`)
		tokens := tr.Split(row, -1)
		if len(tokens) <= 5 {
			continue
		}

		pid, err := strconv.Atoi(tokens[5])
		if err != nil {
			continue
		}

		proc := processList.Filter(func(p interface{}) bool {
			proc := p.(gops.Process)
			return proc.Pid() == pid
		}).First()

		if tokens[1] == "TCP" && tokens[4] == string(state) && proc != nil {
			localAddress := tokens[2]
			tabs = append(tabs, &SockTabEntry{
				Process:      proc.(gops.Process),
				LocalAddress: localAddress,
			})
		}
	}

	return tabs
}
