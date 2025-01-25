package configurable_test

import (
	"testing"

	"github.com/khulnasoft/go/log"
	"github.com/khulnasoft/go/log/internal/configurable"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestCast(t *testing.T) {
	assert.NotPanics(t, func() {
		log.Init(log.Resource{Name: t.Name()})

		// Cast works
		cl := configurable.Cast(log.Scoped("foo"))

		// Core wrapping works
		_ = cl.WithCore(func(c zapcore.Core) zapcore.Core {
			return zapcore.NewTee(c, zapcore.NewNopCore())
		})
	})
}
