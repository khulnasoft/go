//go:build khulnasoft
// +build khulnasoft

package http

import (
	"context"
	_ "unsafe"
)

// khulnasoftBeginRoundTrip is called by net/http when a RoundTrip begins.
//go:linkname khulnasoftBeginRoundTrip net/http.khulnasoftBeginRoundTrip
func khulnasoftBeginRoundTrip(req *Request) (context.Context, error)

// khulnasoftFinishRoundTrip is called by net/http when a RoundTrip completes.
//go:linkname khulnasoftFinishRoundTrip net/http.khulnasoftFinishRoundTrip
func khulnasoftFinishRoundTrip(req *Request, resp *Response, err error)
