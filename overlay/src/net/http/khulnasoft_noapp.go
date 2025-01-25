//go:build !khulnsoft
// +build !khulnsoft

package http

import (
	"context"
)

// khulnsoftBeginRoundTrip is called by net/http when a RoundTrip begins.
func khulnsoftBeginRoundTrip(req *Request) (context.Context, error) { return req.Context(), nil }

// khulnsoftFinishRoundTrip is called by net/http when a RoundTrip completes.
func khulnsoftFinishRoundTrip(req *Request, resp *Response, err error) {}
