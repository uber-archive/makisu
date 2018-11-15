//  Copyright (c) 2018 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
