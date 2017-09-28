package endpoint

import (
	context "context"

	endpoint "github.com/go-kit/kit/endpoint"
	service "github.com/kujtimiihoxha/kit/test_dir/math/pkg/service"
)

// SumRequest collects the request parameters for the Sum method.
type SumRequest struct {
	A int `json:"a"`
	B int `json:"b"`
}

// SumResponse collects the response parameters for the Sum method.
type SumResponse struct {
	R   int   `json:"r"`
	Err error `json:"err"`
}

// MakeSumEndpoint returns an endpoint that invokes Sum on the service.
func MakeSumEndpoint(s service.MathService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(SumRequest)
		r, err := s.Sum(ctx, req.A, req.B)
		return SumResponse{
			Err: err,
			R:   r,
		}, nil
	}
}

// Failed implements Failer.
func (r SumResponse) Failed() error {
	return r.Err
}

// ProdRequest collects the request parameters for the Prod method.
type ProdRequest struct {
	A int `json:"a"`
	B int `json:"b"`
}

// ProdResponse collects the response parameters for the Prod method.
type ProdResponse struct {
	R   int   `json:"r"`
	Err error `json:"err"`
}

// MakeProdEndpoint returns an endpoint that invokes Prod on the service.
func MakeProdEndpoint(s service.MathService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ProdRequest)
		r, err := s.Prod(ctx, req.A, req.B)
		return ProdResponse{
			Err: err,
			R:   r,
		}, nil
	}
}

// Failed implements Failer.
func (r ProdResponse) Failed() error {
	return r.Err
}

// Failer is an interface that should be implemented by response types.
// Response encoders can check if responses are Failer, and if so they've
// failed, and if so encode them using a separate write path based on the error.
type Failure interface {
	Failed() error
}

// Sum implements Service. Primarily useful in a client.
func (e Endpoints) Sum(ctx context.Context, a int, b int) (r int, err error) {
	request := SumRequest{
		A: a,
		B: b,
	}
	response, err := e.SumEndpoint(ctx, request)
	if err != nil {
		return
	}
	return response.(SumResponse).R, response.(SumResponse).Err
}

// Prod implements Service. Primarily useful in a client.
func (e Endpoints) Prod(ctx context.Context, a int, b int) (r int, err error) {
	request := ProdRequest{
		A: a,
		B: b,
	}
	response, err := e.ProdEndpoint(ctx, request)
	if err != nil {
		return
	}
	return response.(ProdResponse).R, response.(ProdResponse).Err
}

// SubRequest collects the request parameters for the Sub method.
type SubRequest struct {
	A int `json:"a"`
	B int `json:"b"`
}

// SubResponse collects the response parameters for the Sub method.
type SubResponse struct {
	R   int   `json:"r"`
	Err error `json:"err"`
}

// MakeSubEndpoint returns an endpoint that invokes Sub on the service.
func MakeSubEndpoint(s service.MathService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(SubRequest)
		r, err := s.Sub(ctx, req.A, req.B)
		return SubResponse{
			Err: err,
			R:   r,
		}, nil
	}
}

// Failed implements Failer.
func (r SubResponse) Failed() error {
	return r.Err
}

// Sub implements Service. Primarily useful in a client.
func (e Endpoints) Sub(ctx context.Context, a int, b int) (r int, err error) {
	request := SubRequest{
		A: a,
		B: b,
	}
	response, err := e.SubEndpoint(ctx, request)
	if err != nil {
		return
	}
	return response.(SubResponse).R, response.(SubResponse).Err
}
