package std_test

import (
	"testing"

	"github.com/khulnasoft/go/log"
	"github.com/khulnasoft/go/log/logtest"
	"github.com/khulnasoft/go/log/std"
	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	root, export := logtest.Captured(t)

	l := std.NewLogger(root, log.LevelInfo)
	l.Println("foobar")

	l.SetPrefix("prefix: ")
	l.Println("baz")

	logs := export()
	assert.Len(t, logs, 2)

	assert.Equal(t, logs[0].Level, log.LevelInfo)
	assert.Equal(t, logs[0].Message, "foobar")

	assert.Equal(t, logs[1].Level, log.LevelInfo)
	assert.Equal(t, logs[1].Message, "prefix: baz")
}
