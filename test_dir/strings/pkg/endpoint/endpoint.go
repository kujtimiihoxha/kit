package endpoint

import (
	context "context"

	endpoint "github.com/go-kit/kit/endpoint"
	service "github.com/kujtimiihoxha/kit/test_dir/strings/pkg/service"
)

// TaskRequest collects the request parameters for the Task method.
type TaskRequest struct {
	Operator string `json:"operator"`
	A        int    `json:"a"`
	B        int    `json:"b"`
}

// TaskResponse collects the response parameters for the Task method.
type TaskResponse struct {
	Rs  string `json:"rs"`
	Err error  `json:"err"`
}

// MakeTaskEndpoint returns an endpoint that invokes Task on the service.
func MakeTaskEndpoint(s service.StringsService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(TaskRequest)
		rs, err := s.Task(ctx, req.Operator, req.A, req.B)
		return TaskResponse{
			Err: err,
			Rs:  rs,
		}, nil
	}
}

// Failed implements Failer.
func (r TaskResponse) Failed() error {
	return r.Err
}

// Failer is an interface that should be implemented by response types.
// Response encoders can check if responses are Failer, and if so they've
// failed, and if so encode them using a separate write path based on the error.
type Failure interface {
	Failed() error
}

// Task implements Service. Primarily useful in a client.
func (e Endpoints) Task(ctx context.Context, operator string, a int, b int) (rs string, err error) {
	request := TaskRequest{
		A:        a,
		B:        b,
		Operator: operator,
	}
	response, err := e.TaskEndpoint(ctx, request)
	if err != nil {
		return
	}
	return response.(TaskResponse).Rs, response.(TaskResponse).Err
}
