package step

import (
	"testing"

	"github.com/uber/makisu/lib/context"

	"github.com/stretchr/testify/require"
)

func TestRunStepExecutionFail(t *testing.T) {
	require := require.New(t)
	context, cleanup := context.BuildContextFixture()
	defer cleanup()

	step := NewRunStep("", "echo hello", false)
	err := step.Execute(context, false)
	require.Error(err)
}
