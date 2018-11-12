package image

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMarshalUnmarshalImageConfig(t *testing.T) {
	require := require.New(t)

	config := NewDefaultImageConfig()
	config.Comment = "This is a test comment"
	content, err := config.MarshalJSON()
	require.NoError(err)

	newConfig, err := NewImageConfigFromJSON(content)
	newConfig.rawJSON = nil
	require.NoError(err)
	require.Equal(config, *newConfig)
	require.Equal(config.ID(), newConfig.ID())

	config.RootFS = nil
	content, err = config.MarshalJSON()
	require.NoError(err)
	_, err = NewImageConfigFromJSON(content)
	require.Error(err)
}

func TestCopyImageConfig(t *testing.T) {
	require := require.New(t)

	config := NewDefaultImageConfig()
	config.Comment = "This is a test comment"

	newConfig, err := NewImageConfigFromCopy(&config)
	newConfig.rawJSON = nil
	require.NoError(err)
	require.Equal(config, *newConfig)

	newConfig.History = append(newConfig.History, History{
		Created:   time.Now(),
		CreatedBy: "makisu ...",
		Author:    "makisu",
	})

	require.NotEqual(config, *newConfig)
}
