package grpc

import (
	context "context"

	grpc "github.com/go-kit/kit/transport/grpc"
	endpoint "github.com/kujtimiihoxha/kit/test_dir/math/pkg/endpoint"
	pb "github.com/kujtimiihoxha/kit/test_dir/math/pkg/grpc/pb"
	context1 "golang.org/x/net/context"
)

// makeSumHandler creates the handler logic
func makeSumHandler(endpoints endpoint.Endpoints, options []grpc.ServerOption) grpc.Handler {
	return grpc.NewServer(endpoints.SumEndpoint, decodeSumRequest, encodeSumResponse, options...)
}

// decodeSumResponse is a transport/grpc.DecodeRequestFunc that converts a
// gRPC request to a user-domain sum request.
// TODO implement the decoder
func decodeSumRequest(_ context.Context, r interface{}) (interface{}, error) {
	req := r.(*pb.SumRequest)
	return endpoint.SumRequest{A: int(req.A), B: int(req.B)}, nil
}

// encodeSumResponse is a transport/grpc.EncodeResponseFunc that converts
// a user-domain response to a gRPC reply.
// TODO implement the encoder
func encodeSumResponse(_ context.Context, r interface{}) (interface{}, error) {
	resp := r.(endpoint.SumResponse)
	err := ""
	if resp.Err != nil {
		err = resp.Err.Error()
	}
	return &pb.SumReply{R: int32(resp.R), Err: err}, nil
}
func (g *grpcServer) Sum(ctx context1.Context, req *pb.SumRequest) (*pb.SumReply, error) {
	_, rep, err := g.sum.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.SumReply), nil
}

// makeProdHandler creates the handler logic
func makeProdHandler(endpoints endpoint.Endpoints, options []grpc.ServerOption) grpc.Handler {
	return grpc.NewServer(endpoints.ProdEndpoint, decodeProdRequest, encodeProdResponse, options...)
}

// decodeProdResponse is a transport/grpc.DecodeRequestFunc that converts a
// gRPC request to a user-domain sum request.
// TODO implement the decoder
func decodeProdRequest(_ context.Context, r interface{}) (interface{}, error) {
	req := r.(*pb.ProdRequest)
	return endpoint.ProdRequest{A: int(req.A), B: int(req.B)}, nil
}

// encodeProdResponse is a transport/grpc.EncodeResponseFunc that converts
// a user-domain response to a gRPC reply.
// TODO implement the encoder
func encodeProdResponse(_ context.Context, r interface{}) (interface{}, error) {
	resp := r.(endpoint.ProdResponse)
	err := ""
	if resp.Err != nil {
		err = resp.Err.Error()
	}
	return &pb.ProdReply{R: int32(resp.R), Err: err}, nil
}
func (g *grpcServer) Prod(ctx context1.Context, req *pb.ProdRequest) (*pb.ProdReply, error) {
	_, rep, err := g.prod.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.ProdReply), nil
}
