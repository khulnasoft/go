package outputcore

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/khulnasoft/go/log/internal/encoders"
	"github.com/khulnasoft/go/log/output"
)

func NewCore(
	output zapcore.WriteSyncer,
	level zapcore.LevelEnabler,
	format output.Format,
	sampling zap.SamplingConfig,
	overrides []Override,
	development bool,
) zapcore.Core {
	newCore := func(level zapcore.LevelEnabler) zapcore.Core {
		return zapcore.NewCore(
			encoders.BuildEncoder(format, development),
			output,
			level,
		)
	}

	core := newOverrideCore(level, overrides, newCore)

	if sampling.Initial > 0 {
		return zapcore.NewSamplerWithOptions(core, time.Second, sampling.Initial, sampling.Thereafter)
	}
	return core
}
