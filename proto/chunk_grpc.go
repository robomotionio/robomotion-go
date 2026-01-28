package proto

import (
	"context"

	"google.golang.org/grpc"
)

// Chunk transfer method names
const (
	RuntimeHelper_GetChunk_FullMethodName    = "/proto.RuntimeHelper/GetChunk"
	RuntimeHelper_StoreChunk_FullMethodName  = "/proto.RuntimeHelper/StoreChunk"
	RuntimeHelper_DeleteChunk_FullMethodName = "/proto.RuntimeHelper/DeleteChunk"
)

// ChunkRuntimeHelperClient extends RuntimeHelperClient with chunk transfer methods.
// This interface is implemented by wrapping the generated runtimeHelperClient.
type ChunkRuntimeHelperClient interface {
	RuntimeHelperClient
	GetChunk(ctx context.Context, in *GetChunkRequest, opts ...grpc.CallOption) (*GetChunkResponse, error)
	StoreChunk(ctx context.Context, in *StoreChunkRequest, opts ...grpc.CallOption) (*Empty, error)
	DeleteChunk(ctx context.Context, in *DeleteChunkRequest, opts ...grpc.CallOption) (*Empty, error)
}

// chunkRuntimeHelperClient wraps the generated client with chunk methods
type chunkRuntimeHelperClient struct {
	RuntimeHelperClient
	cc grpc.ClientConnInterface
}

// NewChunkRuntimeHelperClient creates a new client that supports chunk transfer methods.
// It wraps the standard RuntimeHelperClient with additional chunk operations.
func NewChunkRuntimeHelperClient(cc grpc.ClientConnInterface) ChunkRuntimeHelperClient {
	return &chunkRuntimeHelperClient{
		RuntimeHelperClient: NewRuntimeHelperClient(cc),
		cc:                  cc,
	}
}

func (c *chunkRuntimeHelperClient) GetChunk(ctx context.Context, in *GetChunkRequest, opts ...grpc.CallOption) (*GetChunkResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetChunkResponse)
	err := c.cc.Invoke(ctx, RuntimeHelper_GetChunk_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *chunkRuntimeHelperClient) StoreChunk(ctx context.Context, in *StoreChunkRequest, opts ...grpc.CallOption) (*Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Empty)
	err := c.cc.Invoke(ctx, RuntimeHelper_StoreChunk_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *chunkRuntimeHelperClient) DeleteChunk(ctx context.Context, in *DeleteChunkRequest, opts ...grpc.CallOption) (*Empty, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Empty)
	err := c.cc.Invoke(ctx, RuntimeHelper_DeleteChunk_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}
