package service

import "context"

// MathService describes the service.
type MathService interface {
	// Add your methods here
	Sum(ctx context.Context, a, b int) (r int, err error)
	Prod(ctx context.Context, a, b int) (r int, err error)
}

type basicMathService struct{}

func (ba *basicMathService) Sum(ctx context.Context, a int, b int) (r int, err error) {
	// TODO implement the business logic of Sum
	return a + b, err
}
func (ba *basicMathService) Prod(ctx context.Context, a int, b int) (r int, err error) {
	// TODO implement the business logic of Prod
	return a * b, err
}

// NewBasicMathService returns a naive, stateless implementation of MathService.
func NewBasicMathService() MathService {
	return &basicMathService{}
}

// New returns a MathService with all of the expected middleware wired in.
func New(middleware []Middleware) MathService {
	var svc MathService = NewBasicMathService()
	for _, m := range middleware {
		svc = m(svc)
	}
	return svc
}
