// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: janction/videoRendering/v1/tx.proto

package videoRenderingv1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Msg_CreateVideoRenderingTask_FullMethodName = "/janction.videoRendering.v1.Msg/CreateVideoRenderingTask"
)

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type MsgClient interface {
	// CreateGame create a game.
	CreateVideoRenderingTask(ctx context.Context, in *MsgCreateVideoRenderingTask, opts ...grpc.CallOption) (*MsgCreateVideoRenderingTaskResponse, error)
}

type msgClient struct {
	cc grpc.ClientConnInterface
}

func NewMsgClient(cc grpc.ClientConnInterface) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) CreateVideoRenderingTask(ctx context.Context, in *MsgCreateVideoRenderingTask, opts ...grpc.CallOption) (*MsgCreateVideoRenderingTaskResponse, error) {
	out := new(MsgCreateVideoRenderingTaskResponse)
	err := c.cc.Invoke(ctx, Msg_CreateVideoRenderingTask_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
// All implementations must embed UnimplementedMsgServer
// for forward compatibility
type MsgServer interface {
	// CreateGame create a game.
	CreateVideoRenderingTask(context.Context, *MsgCreateVideoRenderingTask) (*MsgCreateVideoRenderingTaskResponse, error)
	mustEmbedUnimplementedMsgServer()
}

// UnimplementedMsgServer must be embedded to have forward compatible implementations.
type UnimplementedMsgServer struct {
}

func (UnimplementedMsgServer) CreateVideoRenderingTask(context.Context, *MsgCreateVideoRenderingTask) (*MsgCreateVideoRenderingTaskResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateVideoRenderingTask not implemented")
}
func (UnimplementedMsgServer) mustEmbedUnimplementedMsgServer() {}

// UnsafeMsgServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MsgServer will
// result in compilation errors.
type UnsafeMsgServer interface {
	mustEmbedUnimplementedMsgServer()
}

func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	s.RegisterService(&Msg_ServiceDesc, srv)
}

func _Msg_CreateVideoRenderingTask_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCreateVideoRenderingTask)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CreateVideoRenderingTask(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_CreateVideoRenderingTask_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CreateVideoRenderingTask(ctx, req.(*MsgCreateVideoRenderingTask))
	}
	return interceptor(ctx, in, info, handler)
}

// Msg_ServiceDesc is the grpc.ServiceDesc for Msg service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Msg_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "janction.videoRendering.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateVideoRenderingTask",
			Handler:    _Msg_CreateVideoRenderingTask_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "janction/videoRendering/v1/tx.proto",
}
