package service

import (
	context "context"

	log "github.com/go-kit/kit/log"
)

// Middleware describes a service middleware.
type Middleware func(StringsService) StringsService

type loggingMiddleware struct {
	logger log.Logger
	next   StringsService
}

// LoggingMiddleware takes a logger as a dependency
// and returns a StringsService Middleware.
func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next StringsService) StringsService {
		return &loggingMiddleware{logger, next}
	}

}

func (l loggingMiddleware) Task(ctx context.Context, operator string, a int, b int) (rs string, err error) {
	defer func() {
		l.logger.Log("method", "Task", "operator", operator, "a", a, "b", b, "rs", rs, "err", err)
	}()
	return l.next.Task(ctx, operator, a, b)
}
