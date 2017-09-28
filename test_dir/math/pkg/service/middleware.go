package service

import (
	context "context"

	log "github.com/go-kit/kit/log"
)

// Middleware describes a service middleware.
type Middleware func(MathService) MathService

type loggingMiddleware struct {
	logger log.Logger
	next   MathService
}

// LoggingMiddleware takes a logger as a dependency
// and returns a MathService Middleware.
func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next MathService) MathService {
		return &loggingMiddleware{logger, next}
	}

}

func (l loggingMiddleware) Sum(ctx context.Context, a int, b int) (r int, err error) {
	defer func() {
		l.logger.Log("method", "Sum", "a", a, "b", b, "r", r, "err", err)
	}()
	return l.next.Sum(ctx, a, b)
}
func (l loggingMiddleware) Prod(ctx context.Context, a int, b int) (r int, err error) {
	defer func() {
		l.logger.Log("method", "Prod", "a", a, "b", b, "r", r, "err", err)
	}()
	return l.next.Prod(ctx, a, b)
}

func (l loggingMiddleware) Sub(ctx context.Context, a int, b int) (r int, err error) {
	defer func() {
		l.logger.Log("method", "Sub", "a", a, "b", b, "r", r, "err", err)
	}()
	return l.next.Sub(ctx, a, b)
}
