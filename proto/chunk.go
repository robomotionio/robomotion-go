package proto

// Chunk transfer types for large message support.
// These types mirror the protobuf definitions that will be added to plugin.proto.
// Once the proto files are regenerated, these can be removed.

// GetChunkRequest is a request to fetch a chunk of data from the ChunkStore
type GetChunkRequest struct {
	RefId  string `protobuf:"bytes,1,opt,name=ref_id,json=refId,proto3" json:"ref_id,omitempty"`
	Offset int64  `protobuf:"varint,2,opt,name=offset,proto3" json:"offset,omitempty"`
	Length int64  `protobuf:"varint,3,opt,name=length,proto3" json:"length,omitempty"`
}

func (x *GetChunkRequest) GetRefId() string {
	if x != nil {
		return x.RefId
	}
	return ""
}

func (x *GetChunkRequest) GetOffset() int64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

func (x *GetChunkRequest) GetLength() int64 {
	if x != nil {
		return x.Length
	}
	return 0
}

// GetChunkResponse contains chunk data
type GetChunkResponse struct {
	Data      []byte `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
	TotalSize int64  `protobuf:"varint,2,opt,name=total_size,json=totalSize,proto3" json:"total_size,omitempty"`
	IsLast    bool   `protobuf:"varint,3,opt,name=is_last,json=isLast,proto3" json:"is_last,omitempty"`
}

func (x *GetChunkResponse) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *GetChunkResponse) GetTotalSize() int64 {
	if x != nil {
		return x.TotalSize
	}
	return 0
}

func (x *GetChunkResponse) GetIsLast() bool {
	if x != nil {
		return x.IsLast
	}
	return false
}

// StoreChunkRequest is a request to store a chunk of data in the ChunkStore
type StoreChunkRequest struct {
	RefId     string `protobuf:"bytes,1,opt,name=ref_id,json=refId,proto3" json:"ref_id,omitempty"`
	Data      []byte `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
	Offset    int64  `protobuf:"varint,3,opt,name=offset,proto3" json:"offset,omitempty"`
	TotalSize int64  `protobuf:"varint,4,opt,name=total_size,json=totalSize,proto3" json:"total_size,omitempty"`
	IsLast    bool   `protobuf:"varint,5,opt,name=is_last,json=isLast,proto3" json:"is_last,omitempty"`
}

func (x *StoreChunkRequest) GetRefId() string {
	if x != nil {
		return x.RefId
	}
	return ""
}

func (x *StoreChunkRequest) GetData() []byte {
	if x != nil {
		return x.Data
	}
	return nil
}

func (x *StoreChunkRequest) GetOffset() int64 {
	if x != nil {
		return x.Offset
	}
	return 0
}

func (x *StoreChunkRequest) GetTotalSize() int64 {
	if x != nil {
		return x.TotalSize
	}
	return 0
}

func (x *StoreChunkRequest) GetIsLast() bool {
	if x != nil {
		return x.IsLast
	}
	return false
}

// DeleteChunkRequest is a request to delete chunked data from the ChunkStore
type DeleteChunkRequest struct {
	RefId string `protobuf:"bytes,1,opt,name=ref_id,json=refId,proto3" json:"ref_id,omitempty"`
}

func (x *DeleteChunkRequest) GetRefId() string {
	if x != nil {
		return x.RefId
	}
	return ""
}
