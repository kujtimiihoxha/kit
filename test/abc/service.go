package abc

import "context"

type Request struct {
	Name string `json:"name"`
}

type Response struct {
	Hello string `json:"hello"`
}

// @service()
type Service interface {
	// @http(methods='post', route='/hi')
	// @http(methods='post', route='/hello')
	Hello(context.Context, Request) (*Response, error)
}

type baseService struct{}

func (b baseService) Hello(ctx context.Context, req Request) (*Response, error) {
	return &Response{
		Hello: "Hello " + req.Name,
	}, nil
}

func New() Service {
	return &baseService{}
}
