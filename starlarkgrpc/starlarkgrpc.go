// Pacakge starlarkgrpc provides a grpc client implementation based on server
// reflection.
package starlarkgrpc

import (
	"bytes"
	"context"
	"crypto/sha256"

	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	//"google.golang.org/protobuf/reflect/protodesc"
)

func NewModule() *starlarkstruct.Module {
	//p := NewProto(resolver)
	return &starlarkstruct.Module{
		Name: "grpc",
		Members: starlark.StringDict{
			"dial": starlark.NewBuiltin("grpc.dial", Dial),
		},
	}
}

type ClientConn struct {
	cc *grpc.ClientConn
}

// Dial accepts an address and optional
func Dial(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {

	var target string
	if err := starlark.UnpackPositionalArgs("grpc.dial", args, kwargs, 1, &target); err != nil {
		return nil, err
	}

	ctx, ok := thread.Local("context").(context.Context)
	if !ok {
		ctx = context.Background()
	}

	// TODO: opts?
	cc, err := grpc.DialContext(ctx, target)
	if err != nil {
		return nil, err
	}

	c := rpb.NewServerReflectionClient(cc)

	// TODO: watch the stream. When it is recreated refresh the service
	// methods and recreate the mux if needed.
	stream, err := c.ServerReflectionInfo(ctx, grpc.WaitForReady(true))
	if err != nil {
		return err
	}

	//return nil, fmt.Errorf("todo")
}

func (s *state) createHandler(
	cc *grpc.ClientConn,
	stream rpb.ServerReflection_ServerReflectionInfoClient,
) error {
	// TODO: async fetch and mux creation.

	if err := stream.Send(&rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_ListServices{},
	}); err != nil {
		return err
	}

	r, err := stream.Recv()
	if err != nil {
		return err
	}
	// TODO: check r.GetErrorResponse()?

	// File descriptors hash for detecting updates. TODO: sort fds?
	h := sha256.New()

	fds := make(map[string]*descriptorpb.FileDescriptorProto)
	for _, svc := range r.GetListServicesResponse().GetService() {
		if err := stream.Send(&rpb.ServerReflectionRequest{
			MessageRequest: &rpb.ServerReflectionRequest_FileContainingSymbol{
				FileContainingSymbol: svc.GetName(),
			},
		}); err != nil {
			return err
		}

		fdr, err := stream.Recv()
		if err != nil {
			return err
		}

		fdbb := fdr.GetFileDescriptorResponse().GetFileDescriptorProto()

		for _, fdb := range fdbb {
			fd := &descriptorpb.FileDescriptorProto{}
			if err := proto.Unmarshal(fdb, fd); err != nil {
				return err
			}
			fds[fd.GetName()] = fd

			if _, err := h.Write(fdb); err != nil {
				return err
			}
		}
	}

	fdHash := h.Sum(nil)

	// Check if previous connection exists.
	if cl, ok := s.conns[cc]; ok {
		if bytes.Equal(cl.fdHash, fdHash) {
			return nil // nothing to do
		}

		// Drop and recreate below.
		s.removeHandler(cc)
	}

	rslvr, err := newResolver(stream)
	if err != nil {
		return err
	}

	var methods []methodDesc
	for _, fd := range fds {
		file, err := protodesc.NewFile(fd, rslvr)
		if err != nil {
			return err
		}

		ms, err := s.processFile(cc, file)
		if err != nil {
			// TODO: partial dregister?
			return err
		}
		methods = append(methods, ms...)
	}

	// Update methods list.
	s.conns[cc] = connList{
		descs:  methods,
		fdHash: fdHash,
	}
	for _, method := range methods {
		s.methods[method.name] = append(
			s.methods[method.name], methodConn{method, cc},
		)
	}
	return nil
}
