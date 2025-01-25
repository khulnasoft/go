package resource

import (
	"context"
	"testing"

	"github.com/khulnasoft/go/log"
	"github.com/stretchr/testify/assert"
)

func TestBuildOpenTelemetryResource(t *testing.T) {
	_, err := BuildOpenTelemetryResource(context.Background(), log.Resource{})
	assert.NoError(t, err)
}
