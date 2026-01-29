package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	hclog "github.com/hashicorp/go-hclog"
	"github.com/robomotionio/robomotion-go/proto"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	// ChunkSize is the size of each chunk when transferring large data (1MB)
	ChunkSize = 1 * 1024 * 1024

	// ChunkThreshold is the minimum size before data gets chunked (2MB)
	ChunkThreshold = 2 * 1024 * 1024

	// ChunkFetchTimeout is the maximum time to wait for all chunks to be fetched
	ChunkFetchTimeout = 2 * time.Minute
)

var (
	// ErrChunkNotFound is returned when a chunk reference cannot be found
	ErrChunkNotFound = errors.New("chunk not found")

	// ErrChunkFetchTimeout is returned when chunk fetching times out
	ErrChunkFetchTimeout = errors.New("chunk fetch timeout")

	// ErrInvalidChunkRef is returned when a chunk reference is malformed
	ErrInvalidChunkRef = errors.New("invalid chunk reference")
)

// ChunkedRef represents a reference to chunked data stored in the ChunkStore.
// When a field exceeds ChunkThreshold, it is replaced with this reference.
type ChunkedRef struct {
	Chunked   bool   `json:"__chunked__"`
	RefID     string `json:"__chunk_ref__"`
	TotalSize int64  `json:"__total_size__"`
	FieldPath string `json:"__field_path__,omitempty"`
}

// isChunkedRef checks if a JSON value represents a chunked reference
func isChunkedRef(value interface{}) bool {
	m, ok := value.(map[string]interface{})
	if !ok {
		return false
	}
	chunked, _ := m["__chunked__"].(bool)
	return chunked
}

// parseChunkedRef extracts a ChunkedRef from a map
func parseChunkedRef(value interface{}) (*ChunkedRef, error) {
	m, ok := value.(map[string]interface{})
	if !ok {
		return nil, ErrInvalidChunkRef
	}

	chunked, _ := m["__chunked__"].(bool)
	if !chunked {
		return nil, ErrInvalidChunkRef
	}

	refID, _ := m["__chunk_ref__"].(string)
	if refID == "" {
		return nil, ErrInvalidChunkRef
	}

	totalSize, _ := m["__total_size__"].(float64)

	return &ChunkedRef{
		Chunked:   true,
		RefID:     refID,
		TotalSize: int64(totalSize),
		FieldPath: m["__field_path__"].(string),
	}, nil
}

// ContainsChunkedFields checks if a JSON message contains any chunked field references.
// It scans the top-level fields of the JSON object for ChunkedRef markers.
func ContainsChunkedFields(data []byte) bool {
	result := gjson.ParseBytes(data)
	if !result.IsObject() {
		return false
	}

	found := false
	result.ForEach(func(key, value gjson.Result) bool {
		if value.Get("__chunked__").Bool() {
			found = true
			return false // stop iteration
		}
		return true
	})

	return found
}

// FetchChunkedFields resolves all chunked field references in a message by fetching
// the actual data from the ChunkStore via gRPC.
// Note: The SDK does NOT delete chunks after fetching. Chunk lifecycle is managed
// by Deskbot, which tracks consumer counts and handles deletion when all consumers
// have processed the data. This enables fan-out scenarios where multiple nodes
// receive the same chunked message.
func FetchChunkedFields(data []byte, client proto.ChunkRuntimeHelperClient) ([]byte, error) {
	result := gjson.ParseBytes(data)
	if !result.IsObject() {
		return data, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), ChunkFetchTimeout)
	defer cancel()

	var lastErr error
	output := data

	result.ForEach(func(key, value gjson.Result) bool {
		if !value.Get("__chunked__").Bool() {
			return true
		}

		refID := value.Get("__chunk_ref__").String()
		totalSize := value.Get("__total_size__").Int()

		if refID == "" {
			lastErr = ErrInvalidChunkRef
			return false
		}

		// Fetch all chunks for this field
		fieldData, err := fetchAllChunks(ctx, client, refID, totalSize)
		if err != nil {
			hclog.Default().Error("chunk.fetch", "refID", refID, "err", err)
			lastErr = err
			return false
		}

		// Parse the field data as JSON
		var fieldValue interface{}
		if err := json.Unmarshal(fieldData, &fieldValue); err != nil {
			// If it's not valid JSON, treat it as a string
			fieldValue = string(fieldData)
		}

		// Replace the chunked reference with the actual value
		output, err = sjson.SetBytes(output, key.String(), fieldValue)
		if err != nil {
			hclog.Default().Error("chunk.replace", "key", key.String(), "err", err)
			lastErr = err
			return false
		}

		// Note: We do NOT delete the chunk here. Chunk lifecycle is managed by
		// Deskbot, which tracks consumer counts for fan-out scenarios and deletes
		// chunks only when all consumers have processed them.

		return true
	})

	if lastErr != nil {
		return nil, lastErr
	}

	return output, nil
}

// fetchAllChunks retrieves all chunks for a given reference ID and reassembles them.
func fetchAllChunks(ctx context.Context, client proto.ChunkRuntimeHelperClient, refID string, totalSize int64) ([]byte, error) {
	var result bytes.Buffer
	result.Grow(int(totalSize))

	for offset := int64(0); offset < totalSize; {
		select {
		case <-ctx.Done():
			return nil, ErrChunkFetchTimeout
		default:
		}

		resp, err := client.GetChunk(ctx, &proto.GetChunkRequest{
			RefId:  refID,
			Offset: offset,
			Length: ChunkSize,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch chunk at offset %d: %w", offset, err)
		}

		if len(resp.Data) == 0 {
			break
		}

		result.Write(resp.Data)
		offset += int64(len(resp.Data))

		if resp.IsLast {
			break
		}
	}

	return result.Bytes(), nil
}

// StoreAndCreateRefs scans a message for large fields and stores them in the ChunkStore,
// replacing them with ChunkedRef markers. Returns the modified message with refs.
func StoreAndCreateRefs(data []byte, client proto.ChunkRuntimeHelperClient) ([]byte, error) {
	var msg map[string]interface{}
	if err := json.Unmarshal(data, &msg); err != nil {
		// Not a JSON object, check if the whole thing needs chunking
		if len(data) > ChunkThreshold {
			return storeWholeMessage(data, client)
		}
		return data, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), ChunkFetchTimeout)
	defer cancel()

	output := data

	for key, value := range msg {
		fieldBytes, err := json.Marshal(value)
		if err != nil {
			continue
		}

		if len(fieldBytes) <= ChunkThreshold {
			continue
		}

		// Store this field as chunks
		refID := uuid.New().String()
		if err := storeChunkedData(ctx, client, refID, fieldBytes); err != nil {
			hclog.Default().Error("chunk.store", "key", key, "err", err)
			return nil, fmt.Errorf("store chunked field %s: %w", key, err)
		}

		// Replace the field with a chunked reference
		ref := ChunkedRef{
			Chunked:   true,
			RefID:     refID,
			TotalSize: int64(len(fieldBytes)),
			FieldPath: key,
		}

		output, err = sjson.SetBytes(output, key, ref)
		if err != nil {
			hclog.Default().Error("chunk.setref", "key", key, "err", err)
			return nil, fmt.Errorf("set chunked ref for %s: %w", key, err)
		}
	}

	return output, nil
}

// storeWholeMessage stores an entire message (non-JSON or single value) as chunks
func storeWholeMessage(data []byte, client proto.ChunkRuntimeHelperClient) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), ChunkFetchTimeout)
	defer cancel()

	refID := uuid.New().String()
	if err := storeChunkedData(ctx, client, refID, data); err != nil {
		return nil, fmt.Errorf("store whole message: %w", err)
	}

	ref := ChunkedRef{
		Chunked:   true,
		RefID:     refID,
		TotalSize: int64(len(data)),
		FieldPath: "",
	}

	return json.Marshal(ref)
}

// storeChunkedData stores data in the ChunkStore by sending it in chunks via StoreChunk RPC.
func storeChunkedData(ctx context.Context, client proto.ChunkRuntimeHelperClient, refID string, data []byte) error {
	totalSize := int64(len(data))

	for offset := int64(0); offset < totalSize; offset += ChunkSize {
		select {
		case <-ctx.Done():
			return ErrChunkFetchTimeout
		default:
		}

		end := offset + ChunkSize
		if end > totalSize {
			end = totalSize
		}
		isLast := end >= totalSize

		_, err := client.StoreChunk(ctx, &proto.StoreChunkRequest{
			RefId:     refID,
			Data:      data[offset:end],
			Offset:    offset,
			TotalSize: totalSize,
			IsLast:    isLast,
		})
		if err != nil {
			return fmt.Errorf("store chunk at offset %d: %w", offset, err)
		}
	}

	return nil
}
