//go:build !khulnasoft
// +build !khulnasoft

package http

import (
	"context"
)

// khulnasoftBeginRoundTrip is called by net/http when a RoundTrip begins.
func khulnasoftBeginRoundTrip(req *Request) (context.Context, error) { return req.Context(), nil }

// khulnasoftFinishRoundTrip is called by net/http when a RoundTrip completes.
func khulnasoftFinishRoundTrip(req *Request, resp *Response, err error) {
	// Ensure the function signature matches the one in khulnasoft_app.go
	_ = resp
	_ = err
}
