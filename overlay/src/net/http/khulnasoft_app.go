//go:build khulnsoft
// +build khulnsoft

package http

import (
	"context"
	_ "unsafe"
)

// khulnsoftBeginRoundTrip is called by net/http when a RoundTrip begins.
func khulnsoftBeginRoundTrip(req *Request) (context.Context, error)

// khulnsoftFinishRoundTrip is called by net/http when a RoundTrip completes.
func khulnsoftFinishRoundTrip(req *Request, resp *Response, err error)
