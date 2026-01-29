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

	// Chunk should NOT be deleted - Deskbot manages lifecycle for fan-out support
	if _, ok := client.chunks["test-ref-1"]; !ok {
		t.Error("chunk should NOT be deleted by SDK - Deskbot manages chunk lifecycle")
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

// deleteTrackingMockClient tracks if DeleteChunk is ever called
type deleteTrackingMockClient struct {
	*mockChunkClient
	deleteCalled bool
}

func (m *deleteTrackingMockClient) DeleteChunk(ctx context.Context, req *proto.DeleteChunkRequest, opts ...grpc.CallOption) (*proto.Empty, error) {
	m.deleteCalled = true
	return m.mockChunkClient.DeleteChunk(ctx, req, opts...)
}

// TestNoDeleteChunkCalled verifies SDK never calls DeleteChunk
func TestNoDeleteChunkCalled(t *testing.T) {
	client := &deleteTrackingMockClient{
		mockChunkClient: newMockChunkClient(),
		deleteCalled:    false,
	}

	largeData := `["item1", "item2", "item3"]`
	client.chunks["test-ref"] = []byte(largeData)

	msg := `{"data": {"__chunked__": true, "__chunk_ref__": "test-ref", "__total_size__": 25, "__field_path__": "data"}}`

	_, err := FetchChunkedFields([]byte(msg), client)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if client.deleteCalled {
		t.Error("DeleteChunk was called - SDK should NOT delete chunks!")
	}
}

// TestFanOut_MultipleConsumers verifies chunks persist for all consumers
func TestFanOut_MultipleConsumers(t *testing.T) {
	client := newMockChunkClient()
	originalData := `["item1", "item2", "item3"]`

	refID := "fanout-test-ref"
	client.chunks[refID] = []byte(originalData)

	// Create message with chunked ref
	msg := `{"data": {"__chunked__": true, "__chunk_ref__": "fanout-test-ref", "__total_size__": 25, "__field_path__": "data"}}`

	// Consumer 1 fetches
	result1, err := FetchChunkedFields([]byte(msg), client)
	if err != nil {
		t.Fatalf("Consumer 1 fetch failed: %v", err)
	}

	// Chunk should STILL exist for consumer 2 (no deletion!)
	if _, ok := client.chunks[refID]; !ok {
		t.Error("Chunk was deleted after first consumer - fan-out broken!")
	}

	// Consumer 2 fetches same chunk
	result2, err := FetchChunkedFields([]byte(msg), client)
	if err != nil {
		t.Fatalf("Consumer 2 fetch failed: %v", err)
	}

	// Both should get identical data
	var resultMap1, resultMap2 map[string]interface{}
	json.Unmarshal(result1, &resultMap1)
	json.Unmarshal(result2, &resultMap2)

	data1, _ := json.Marshal(resultMap1["data"])
	data2, _ := json.Marshal(resultMap2["data"])

	if string(data1) != string(data2) {
		t.Error("Consumer 1 and 2 got different data")
	}
}

// TestFanOut_ThreeWaySplit verifies chunks work with 3+ consumers
func TestFanOut_ThreeWaySplit(t *testing.T) {
	client := newMockChunkClient()
	originalData := `["large","array","data"]`

	refID := "three-way-ref"
	client.chunks[refID] = []byte(originalData)

	msg := `{"items": {"__chunked__": true, "__chunk_ref__": "three-way-ref", "__total_size__": 23, "__field_path__": "items"}}`

	// Simulate 3 consumers fetching in sequence
	for i := 1; i <= 3; i++ {
		_, err := FetchChunkedFields([]byte(msg), client)
		if err != nil {
			t.Fatalf("Consumer %d fetch failed: %v", i, err)
		}

		// Chunk must persist for remaining consumers
		if _, ok := client.chunks[refID]; !ok {
			t.Errorf("Chunk deleted after consumer %d, but consumers %d-%d still need it",
				i, i+1, 3)
		}
	}
}

// TestFanOut_MultipleChunkedFields verifies fan-out with multiple large fields
func TestFanOut_MultipleChunkedFields(t *testing.T) {
	client := newMockChunkClient()

	field1Data := `["field1","data"]`
	field2Data := `["field2","data"]`

	client.chunks["ref-field1"] = []byte(field1Data)
	client.chunks["ref-field2"] = []byte(field2Data)

	msg := `{
		"field1": {"__chunked__": true, "__chunk_ref__": "ref-field1", "__total_size__": 17, "__field_path__": "field1"},
		"field2": {"__chunked__": true, "__chunk_ref__": "ref-field2", "__total_size__": 17, "__field_path__": "field2"},
		"small": "inline"
	}`

	// Consumer 1
	_, err := FetchChunkedFields([]byte(msg), client)
	if err != nil {
		t.Fatalf("Consumer 1 failed: %v", err)
	}

	// Both chunks should still exist
	if _, ok := client.chunks["ref-field1"]; !ok {
		t.Error("ref-field1 deleted prematurely")
	}
	if _, ok := client.chunks["ref-field2"]; !ok {
		t.Error("ref-field2 deleted prematurely")
	}

	// Consumer 2
	_, err = FetchChunkedFields([]byte(msg), client)
	if err != nil {
		t.Fatalf("Consumer 2 failed: %v", err)
	}

	// Chunks should still exist after consumer 2 as well
	if _, ok := client.chunks["ref-field1"]; !ok {
		t.Error("ref-field1 deleted after consumer 2")
	}
	if _, ok := client.chunks["ref-field2"]; !ok {
		t.Error("ref-field2 deleted after consumer 2")
	}
}
