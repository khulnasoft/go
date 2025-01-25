package lspext

import "github.com/khulnasoft/go/go-lsp/lspext"

// See https://github.com/khulnasoft/language-server-protocol/pull/14

// CacheGetParams is the input for 'cache/get'. The response is any or null.
type CacheGetParams = lspext.CacheGetParams

// CacheSetParams is the input for the notification 'cache/set'.
type CacheSetParams = lspext.CacheSetParams
