package service

import (
	"context"
	"fmt"

	http3 "github.com/go-kit/kit/transport/http"
	"github.com/kujtimiihoxha/kit/test_dir/math/client/http"
)

// StringsService describes the service.
type StringsService interface {
	// Add your methods here
	Task(ctx context.Context, operator string, a, b int) (rs string, err error)
}

type basicStringsService struct{}

func (ba *basicStringsService) Task(ctx context.Context, operator string, a int, b int) (rs string, err error) {
	svc, err := http.New("http://math_svc:8081", map[string][]http3.ClientOption{})
	if err != nil {
		return "", err
	}
	if operator == "*" {
		r, err := svc.Prod(context.Background(), a, b)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("The product of %d and %d is %d", a, b, r), nil
	}
	if operator == "+" {
		r, err := svc.Sum(context.Background(), a, b)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("The sum of %d and %d is %d", a, b, r), nil
	}

	if operator == "-" {
		r, err := svc.Sub(context.Background(), a, b)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("The sub of %d and %d is %d", a, b, r), nil
	}
	return "Not Found", err
}

// NewBasicStringsService returns a naive, stateless implementation of StringsService.
func NewBasicStringsService() StringsService {
	return &basicStringsService{}
}

// New returns a StringsService with all of the expected middleware wired in.
func New(middleware []Middleware) StringsService {
	var svc StringsService = NewBasicStringsService()
	for _, m := range middleware {
		svc = m(svc)
	}
	return svc
}
