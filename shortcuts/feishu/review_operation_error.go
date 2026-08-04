package feishu

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

type ReviewOperationErrorClass string

const (
	ReviewOperationErrorTerminal          ReviewOperationErrorClass = "terminal"
	ReviewOperationErrorTransient         ReviewOperationErrorClass = "transient"
	ReviewOperationErrorRateLimited       ReviewOperationErrorClass = "rate_limited"
	ReviewOperationErrorStale             ReviewOperationErrorClass = "stale"
	ReviewOperationErrorUnknownSideEffect ReviewOperationErrorClass = "unknown_side_effect"
)

type ReviewOperationError struct {
	Class                    ReviewOperationErrorClass
	Code                     string
	HTTPStatus               int
	RetryAfter               time.Duration
	RemoteSideEffectPossible bool
	Err                      error
}

func (e *ReviewOperationError) Error() string {
	if e == nil {
		return "review operation failed"
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("review operation failed: %s", e.Code)
}

func (e *ReviewOperationError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

var reviewOperationHTTPStatusPattern = regexp.MustCompile(`(?i)HTTP\s+(\d{3})`)

func ClassifyReviewOperationError(err error, operation ReviewOperation, requestStarted bool) *ReviewOperationError {
	if err == nil {
		return nil
	}
	var classified *ReviewOperationError
	if errors.As(err, &classified) {
		copy := *classified
		copy.Err = err
		return &copy
	}
	var httpErr *OpenAPIHTTPError
	if errors.As(err, &httpErr) {
		class := ReviewOperationErrorTerminal
		switch {
		case httpErr.StatusCode == 429:
			class = ReviewOperationErrorRateLimited
		case httpErr.StatusCode >= 500:
			class = ReviewOperationErrorTransient
		}
		possible := requestStarted && operation.RetrySafety != ReviewRetryIdempotent && httpErr.StatusCode >= 500
		if possible {
			class = ReviewOperationErrorUnknownSideEffect
		}
		return &ReviewOperationError{
			Class:                    class,
			Code:                     fmt.Sprintf("http_%d", httpErr.StatusCode),
			HTTPStatus:               httpErr.StatusCode,
			RetryAfter:               httpErr.RetryAfter,
			RemoteSideEffectPossible: possible,
			Err:                      err,
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		possible := requestStarted && operation.RetrySafety != ReviewRetryIdempotent
		class := ReviewOperationErrorTransient
		if possible {
			class = ReviewOperationErrorUnknownSideEffect
		}
		return &ReviewOperationError{Class: class, Code: "request_timeout", RemoteSideEffectPossible: possible, Err: err}
	}
	var networkError net.Error
	if errors.As(err, &networkError) {
		possible := requestStarted && operation.RetrySafety != ReviewRetryIdempotent
		class := ReviewOperationErrorTransient
		if possible {
			class = ReviewOperationErrorUnknownSideEffect
		}
		return &ReviewOperationError{Class: class, Code: "network_error", RemoteSideEffectPossible: possible, Err: err}
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "database is locked") || strings.Contains(message, "sqlite_busy") {
		return &ReviewOperationError{Class: ReviewOperationErrorTransient, Code: "sqlite_busy", Err: err}
	}
	if match := reviewOperationHTTPStatusPattern.FindStringSubmatch(message); len(match) == 2 {
		switch match[1] {
		case "429":
			return &ReviewOperationError{Class: ReviewOperationErrorRateLimited, Code: "http_429", HTTPStatus: 429, Err: err}
		case "502", "503", "504":
			possible := requestStarted && operation.RetrySafety != ReviewRetryIdempotent
			if possible {
				return &ReviewOperationError{Class: ReviewOperationErrorUnknownSideEffect, Code: "http_" + match[1], RemoteSideEffectPossible: true, Err: err}
			}
			return &ReviewOperationError{Class: ReviewOperationErrorTransient, Code: "http_" + match[1], Err: err}
		}
	}
	return &ReviewOperationError{Class: ReviewOperationErrorTerminal, Code: "operation_failed", Err: err}
}

func staleReviewOperationError(code string) *ReviewOperationError {
	return &ReviewOperationError{Class: ReviewOperationErrorStale, Code: strings.TrimSpace(code), Err: fmt.Errorf("review operation is stale: %s", code)}
}
