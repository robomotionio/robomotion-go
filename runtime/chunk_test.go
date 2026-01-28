package runtime

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/robomotionio/robomotion-go/proto"
	"google.golang.org/grpc"
)

// mockChunkClient implements proto.ChunkRuntimeHelperClient for testing
type mockChunkClient struct {
	proto.RuntimeHelperClient
	chunks map[string][]byte
}

func newMockChunkClient() *mockChunkClient {
	return &mockChunkClient{
		chunks: make(map[string][]byte),
	}
}

func (m *mockChunkClient) GetChunk(ctx context.Context, req *proto.GetChunkRequest, opts ...grpc.CallOption) (*proto.GetChunkResponse, error) {
	data, ok := m.chunks[req.RefId]
	if !ok {
		return nil, ErrChunkNotFound
	}

	end := req.Offset + req.Length
	if end > int64(len(data)) {
		end = int64(len(data))
	}

	isLast := end >= int64(len(data))

	return &proto.GetChunkResponse{
		Data:      data[req.Offset:end],
		TotalSize: int64(len(data)),
		IsLast:    isLast,
	}, nil
}

func (m *mockChunkClient) StoreChunk(ctx context.Context, req *proto.StoreChunkRequest, opts ...grpc.CallOption) (*proto.Empty, error) {
	existing, ok := m.chunks[req.RefId]
	if !ok {
		existing = make([]byte, req.TotalSize)
		m.chunks[req.RefId] = existing
	}

	copy(existing[req.Offset:], req.Data)
	return &proto.Empty{}, nil
}

func (m *mockChunkClient) DeleteChunk(ctx context.Context, req *proto.DeleteChunkRequest, opts ...grpc.CallOption) (*proto.Empty, error) {
	delete(m.chunks, req.RefId)
	return &proto.Empty{}, nil
}

func TestContainsChunkedFields(t *testing.T) {
	tests := []struct {
		name     string
		data     string
		expected bool
	}{
		{
			name:     "no chunked fields",
			data:     `{"name": "test", "value": 123}`,
			expected: false,
		},
		{
			name:     "has chunked field",
			data:     `{"name": "test", "data": {"__chunked__": true, "__chunk_ref__": "abc123", "__total_size__": 1000}}`,
			expected: true,
		},
		{
			name:     "chunked false",
			data:     `{"data": {"__chunked__": false}}`,
			expected: false,
		},
		{
			name:     "empty object",
			data:     `{}`,
			expected: false,
		},
		{
			name:     "not an object",
			data:     `[1, 2, 3]`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsChunkedFields([]byte(tt.data))
			if result != tt.expected {
				t.Errorf("ContainsChunkedFields() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestChunkedRef(t *testing.T) {
	ref := ChunkedRef{
		Chunked:   true,
		RefID:     "test-ref-123",
		TotalSize: 5000,
		FieldPath: "data",
	}

	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("Failed to marshal ChunkedRef: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal ChunkedRef: %v", err)
	}

	if parsed["__chunked__"] != true {
		t.Error("__chunked__ should be true")
	}
	if parsed["__chunk_ref__"] != "test-ref-123" {
		t.Error("__chunk_ref__ should be 'test-ref-123'")
	}
	if parsed["__total_size__"].(float64) != 5000 {
		t.Error("__total_size__ should be 5000")
	}
}

func TestFetchChunkedFields(t *testing.T) {
	client := newMockChunkClient()

	// Store some test data
	originalData := `["item1", "item2", "item3"]`
	client.chunks["test-ref-1"] = []byte(originalData)

	// Create a message with a chunked field reference
	msg := map[string]interface{}{
		"name": "test",
		"data": map[string]interface{}{
			"__chunked__":    true,
			"__chunk_ref__":  "test-ref-1",
			"__total_size__": float64(len(originalData)),
			"__field_path__": "data",
		},
	}

	msgBytes, _ := json.Marshal(msg)

	// Fetch the chunked fields
	result, err := FetchChunkedFields(msgBytes, client)
	if err != nil {
		t.Fatalf("FetchChunkedFields failed: %v", err)
	}

	// Parse the result
	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Check that the chunked field was resolved
	if resultMap["name"] != "test" {
		t.Error("name field should be preserved")
	}

	dataArray, ok := resultMap["data"].([]interface{})
	if !ok {
		t.Fatalf("data field should be an array, got %T", resultMap["data"])
	}

	if len(dataArray) != 3 {
		t.Errorf("data array should have 3 elements, got %d", len(dataArray))
	}

	// Check that the chunk was deleted
	if _, ok := client.chunks["test-ref-1"]; ok {
		t.Error("chunk should be deleted after fetch")
	}
}

func TestStoreAndCreateRefs(t *testing.T) {
	client := newMockChunkClient()

	// Create a large message that exceeds threshold
	largeData := make([]byte, ChunkThreshold+1000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	msg := map[string]interface{}{
		"name":      "test",
		"largeData": largeData,
	}

	msgBytes, _ := json.Marshal(msg)

	// Store and create refs
	result, err := StoreAndCreateRefs(msgBytes, client)
	if err != nil {
		t.Fatalf("StoreAndCreateRefs failed: %v", err)
	}

	// Parse the result
	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	// Check that name is preserved
	if resultMap["name"] != "test" {
		t.Error("name field should be preserved")
	}

	// Check that largeData is replaced with a chunked ref
	largeDataRef, ok := resultMap["largeData"].(map[string]interface{})
	if !ok {
		t.Fatalf("largeData should be a chunked ref, got %T", resultMap["largeData"])
	}

	if largeDataRef["__chunked__"] != true {
		t.Error("largeData should have __chunked__ = true")
	}

	refID, ok := largeDataRef["__chunk_ref__"].(string)
	if !ok || refID == "" {
		t.Error("largeData should have a valid __chunk_ref__")
	}

	// Check that the chunk was stored
	if _, ok := client.chunks[refID]; !ok {
		t.Error("chunk should be stored in the client")
	}
}

func TestSmallMessageNotChunked(t *testing.T) {
	client := newMockChunkClient()

	// Create a small message
	msg := map[string]interface{}{
		"name":  "test",
		"value": 123,
	}

	msgBytes, _ := json.Marshal(msg)

	// Store and create refs - should return unchanged
	result, err := StoreAndCreateRefs(msgBytes, client)
	if err != nil {
		t.Fatalf("StoreAndCreateRefs failed: %v", err)
	}

	// Result should be the same as input
	var resultMap map[string]interface{}
	if err := json.Unmarshal(result, &resultMap); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if resultMap["name"] != "test" {
		t.Error("name should be 'test'")
	}
	if resultMap["value"].(float64) != 123 {
		t.Error("value should be 123")
	}

	// No chunks should be stored
	if len(client.chunks) != 0 {
		t.Error("no chunks should be stored for small messages")
	}
}

func TestRoundTrip_LargeData(t *testing.T) {
	client := newMockChunkClient()

	// Create a message with large data
	largeData := make([]interface{}, 100000) // Array that will exceed threshold when serialized
	for i := range largeData {
		largeData[i] = map[string]interface{}{
			"index": i,
			"value": "some data that makes this larger",
		}
	}

	msg := map[string]interface{}{
		"name": "test",
		"data": largeData,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Skip if the message isn't large enough
	if len(msgBytes) <= ChunkThreshold {
		t.Skip("Test data not large enough to trigger chunking")
	}

	// Store and create refs
	chunkedMsg, err := StoreAndCreateRefs(msgBytes, client)
	if err != nil {
		t.Fatalf("StoreAndCreateRefs failed: %v", err)
	}

	// Verify chunking occurred
	if !ContainsChunkedFields(chunkedMsg) {
		t.Fatal("Message should contain chunked fields after StoreAndCreateRefs")
	}

	// Fetch the chunked fields to restore original
	restored, err := FetchChunkedFields(chunkedMsg, client)
	if err != nil {
		t.Fatalf("FetchChunkedFields failed: %v", err)
	}

	// Verify restoration
	var restoredMap map[string]interface{}
	if err := json.Unmarshal(restored, &restoredMap); err != nil {
		t.Fatalf("Failed to parse restored: %v", err)
	}

	if restoredMap["name"] != "test" {
		t.Error("name should be 'test' after round-trip")
	}

	restoredData, ok := restoredMap["data"].([]interface{})
	if !ok {
		t.Fatalf("data should be an array, got %T", restoredMap["data"])
	}

	if len(restoredData) != len(largeData) {
		t.Errorf("Expected %d items, got %d", len(largeData), len(restoredData))
	}
}

func TestMultipleChunkedFields(t *testing.T) {
	client := newMockChunkClient()

	// Create data that exceeds threshold
	largeField1 := make([]byte, ChunkThreshold+100)
	largeField2 := make([]byte, ChunkThreshold+200)

	for i := range largeField1 {
		largeField1[i] = byte(i % 256)
	}
	for i := range largeField2 {
		largeField2[i] = byte((i + 100) % 256)
	}

	msg := map[string]interface{}{
		"field1": largeField1,
		"field2": largeField2,
		"small":  "this is small",
	}

	msgBytes, _ := json.Marshal(msg)

	// Store and create refs
	chunkedMsg, err := StoreAndCreateRefs(msgBytes, client)
	if err != nil {
		t.Fatalf("StoreAndCreateRefs failed: %v", err)
	}

	// Parse to check structure
	var chunkedMap map[string]interface{}
	json.Unmarshal(chunkedMsg, &chunkedMap)

	// Both large fields should be chunked
	field1Ref, ok := chunkedMap["field1"].(map[string]interface{})
	if !ok {
		t.Fatal("field1 should be a chunked ref")
	}
	if field1Ref["__chunked__"] != true {
		t.Error("field1 should be chunked")
	}

	field2Ref, ok := chunkedMap["field2"].(map[string]interface{})
	if !ok {
		t.Fatal("field2 should be a chunked ref")
	}
	if field2Ref["__chunked__"] != true {
		t.Error("field2 should be chunked")
	}

	// Small field should remain inline
	if chunkedMap["small"] != "this is small" {
		t.Error("small field should remain inline")
	}
}

func TestChunkFetchError(t *testing.T) {
	client := newMockChunkClient()

	// Create a message with a chunk ref that doesn't exist
	msg := map[string]interface{}{
		"name": "test",
		"data": map[string]interface{}{
			"__chunked__":    true,
			"__chunk_ref__":  "nonexistent-ref",
			"__total_size__": float64(1000),
			"__field_path__": "data",
		},
	}

	msgBytes, _ := json.Marshal(msg)

	// FetchChunkedFields should fail
	_, err := FetchChunkedFields(msgBytes, client)
	if err == nil {
		t.Error("Expected error when fetching nonexistent chunk")
	}
}

func TestIsChunkedTransferCapable_NoRobotInfo(t *testing.T) {
	// Without a proper runtime helper set up, this should return false
	result := IsChunkedTransferCapable()
	if result {
		t.Error("Should return false when robot info is not available")
	}
}
