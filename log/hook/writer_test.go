package hook_test

import (
	"testing"

	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/require"
	"go.bobheadxi.dev/streamline/jq"
	"go.bobheadxi.dev/streamline/pipe"

	"github.com/khulnasoft/go/log"
	"github.com/khulnasoft/go/log/hook"
	"github.com/khulnasoft/go/log/logtest"
	"github.com/khulnasoft/go/log/output"
)

func TestWriter(t *testing.T) {
	logger, exportLogs := logtest.Captured(t)

	writer, stream := pipe.NewStream()
	hookedLogger := hook.Writer(logger, writer, log.LevelWarn, output.FormatJSON)

	hookedLogger.Debug("debug message")
	hookedLogger.Warn("warn message")
	hookedLogger.Error("error message")

	logger.Error("parent message")

	// done with writing
	writer.CloseWithError(nil)

	// hooked logger output - only warn and above, and messages logged to parent are not
	// included. We only get the messages because there's no easy way to mock the clock.
	hookedOutput, err := stream.WithPipeline(jq.Pipeline(".Body")).Lines()
	require.NoError(t, err)
	autogold.Expect([]string{`"warn message"`, `"error message"`}).Equal(t, hookedOutput)

	// parent logger output - should receive everything
	parentOutput := exportLogs().Messages()
	autogold.Expect([]string{
		"debug message", "warn message", "error message",
		"parent message",
	}).Equal(t, parentOutput)
}
