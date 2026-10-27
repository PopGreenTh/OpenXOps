package aws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Retry runs fn with exponential backoff on throttling and 5xx errors.
func Retry(ctx context.Context, fn func() error) error {
	var last error
	backoff := 100 * time.Millisecond
	for attempt := 0; attempt < 6; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := fn()
		if err == nil {
			return nil
		}
		last = err
		if !retryable(err) {
			return err
		}
		jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
		timer := time.NewTimer(backoff + jitter)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		backoff *= 2
		if backoff > 5*time.Second {
			backoff = 5 * time.Second
		}
	}
	return fmt.Errorf("after retries: %w", last)
}

func retryable(err error) bool {
	var respErr *smithyhttp.ResponseError
	if errors.As(err, &respErr) && respErr.HTTPStatusCode() >= 500 {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "ThrottlingException", "TooManyRequestsException", "ServiceUnavailable",
			"InternalServerError", "ServiceException":
			return true
		}
	}
	return false
}
