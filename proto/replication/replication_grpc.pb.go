// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package replication

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion7

// ReplicationClient is the client API for Replication service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ReplicationClient interface {
	Replicated(ctx context.Context, in *ReplicatedReq, opts ...grpc.CallOption) (*ReplicatedRes, error)
	AppendEntries(ctx context.Context, in *AppendEntriesReq, opts ...grpc.CallOption) (*AppendEntriesRes, error)
}

type replicationClient struct {
	cc grpc.ClientConnInterface
}

func NewReplicationClient(cc grpc.ClientConnInterface) ReplicationClient {
	return &replicationClient{cc}
}

func (c *replicationClient) Replicated(ctx context.Context, in *ReplicatedReq, opts ...grpc.CallOption) (*ReplicatedRes, error) {
	out := new(ReplicatedRes)
	err := c.cc.Invoke(ctx, "/Replication/Replicated", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *replicationClient) AppendEntries(ctx context.Context, in *AppendEntriesReq, opts ...grpc.CallOption) (*AppendEntriesRes, error) {
	out := new(AppendEntriesRes)
	err := c.cc.Invoke(ctx, "/Replication/AppendEntries", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ReplicationServer is the server API for Replication service.
// All implementations must embed UnimplementedReplicationServer
// for forward compatibility
type ReplicationServer interface {
	Replicated(context.Context, *ReplicatedReq) (*ReplicatedRes, error)
	AppendEntries(context.Context, *AppendEntriesReq) (*AppendEntriesRes, error)
	mustEmbedUnimplementedReplicationServer()
}

// UnimplementedReplicationServer must be embedded to have forward compatible implementations.
type UnimplementedReplicationServer struct {
}

func (UnimplementedReplicationServer) Replicated(context.Context, *ReplicatedReq) (*ReplicatedRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Replicated not implemented")
}
func (UnimplementedReplicationServer) AppendEntries(context.Context, *AppendEntriesReq) (*AppendEntriesRes, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AppendEntries not implemented")
}
func (UnimplementedReplicationServer) mustEmbedUnimplementedReplicationServer() {}

// UnsafeReplicationServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ReplicationServer will
// result in compilation errors.
type UnsafeReplicationServer interface {
	mustEmbedUnimplementedReplicationServer()
}

func RegisterReplicationServer(s *grpc.Server, srv ReplicationServer) {
	s.RegisterService(&_Replication_serviceDesc, srv)
}

func _Replication_Replicated_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReplicatedReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReplicationServer).Replicated(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Replication/Replicated",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReplicationServer).Replicated(ctx, req.(*ReplicatedReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _Replication_AppendEntries_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AppendEntriesReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReplicationServer).AppendEntries(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/Replication/AppendEntries",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReplicationServer).AppendEntries(ctx, req.(*AppendEntriesReq))
	}
	return interceptor(ctx, in, info, handler)
}

var _Replication_serviceDesc = grpc.ServiceDesc{
	ServiceName: "Replication",
	HandlerType: (*ReplicationServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Replicated",
			Handler:    _Replication_Replicated_Handler,
		},
		{
			MethodName: "AppendEntries",
			Handler:    _Replication_AppendEntries_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "replication.proto",
}
