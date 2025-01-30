package assert

import (
	"context"
	"time"
)

func Assert(cond bool, msg string) {
	if !cond {
		panic(msg)
	}
}

func AssertDeadline(ctx context.Context) {
	if ctx == nil {
		panic("context is nil")
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		panic("deadline not set")
	}
	if deadline.Before(time.Now()) {
		panic("deadline has already passed")
	}
}

func AssertNotEmpty(s string) {
	if s == "" {
		panic("expected non-empty string")
	}
}

func AssertNotNil(a any) {
	if a == nil {
		panic("expect non-nil value")
	}
}
