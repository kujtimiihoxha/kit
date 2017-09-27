package grpc

import (
	context "context"
	errors "errors"

	endpoint "github.com/go-kit/kit/endpoint"
	grpc1 "github.com/go-kit/kit/transport/grpc"
	endpoint1 "github.com/kujtimiihoxha/kit/test_dir/math/pkg/endpoint"
	pb "github.com/kujtimiihoxha/kit/test_dir/math/pkg/grpc/pb"
	service "github.com/kujtimiihoxha/kit/test_dir/math/pkg/service"
	grpc "google.golang.org/grpc"
)

// New returns an AddService backed by a gRPC server at the other end
//  of the conn. The caller is responsible for constructing the conn, and
// eventually closing the underlying transport. We bake-in certain middlewares,
// implementing the client library pattern.
func New(conn *grpc.ClientConn, options map[string][]grpc1.ClientOption) (service.MathService, error) {
	var sumEndpoint endpoint.Endpoint
	{
		sumEndpoint = grpc1.NewClient(conn, "pb.Math", "Sum", encodeSumRequest, decodeSumResponse, pb.SumReply{}, options["Sum"]...).Endpoint()
	}

	var prodEndpoint endpoint.Endpoint
	{
		prodEndpoint = grpc1.NewClient(conn, "pb.Math", "Prod", encodeProdRequest, decodeProdResponse, pb.ProdReply{}, options["Prod"]...).Endpoint()
	}

	return endpoint1.Endpoints{
		ProdEndpoint: prodEndpoint,
		SumEndpoint:  sumEndpoint,
	}, nil
}

// encodeSumRequest is a transport/grpc.EncodeRequestFunc that converts a
//  user-domain sum request to a gRPC request.
func encodeSumRequest(_ context.Context, request interface{}) (interface{}, error) {
	r := request.(endpoint1.SumRequest)
	return &pb.SumRequest{A: int32(r.A), B: int32(r.B)}, nil
}

// decodeSumResponse is a transport/grpc.DecodeResponseFunc that converts
// a gRPC concat reply to a user-domain concat response.
func decodeSumResponse(_ context.Context, reply interface{}) (interface{}, error) {
	r := reply.(*pb.SumReply)
	var err error
	if r.Err != "" {
		err = errors.New(r.Err)
	}
	return endpoint1.SumResponse{R: int(r.R), Err: err}, nil
}

// encodeProdRequest is a transport/grpc.EncodeRequestFunc that converts a
//  user-domain sum request to a gRPC request.
func encodeProdRequest(_ context.Context, request interface{}) (interface{}, error) {
	r := request.(endpoint1.ProdRequest)
	return &pb.ProdRequest{A: int32(r.A), B: int32(r.B)}, nil
}

// decodeProdResponse is a transport/grpc.DecodeResponseFunc that converts
// a gRPC concat reply to a user-domain concat response.
func decodeProdResponse(_ context.Context, reply interface{}) (interface{}, error) {
	r := reply.(*pb.ProdReply)
	var err error
	if r.Err != "" {
		err = errors.New(r.Err)
	}
	return endpoint1.ProdResponse{R: int(r.R), Err: err}, nil
}
