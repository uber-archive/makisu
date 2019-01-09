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

package keyvalue

import (
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/uber/makisu/mocks/net/http"
)

const _testURL = "http://localhost:0/test"

func TestHTTPStore(t *testing.T) {
	t.Run("get_no_exist", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		transport := mockhttp.NewMockRoundTripper(ctrl)
		transport.EXPECT().RoundTrip(gomock.Any()).
			Return(&http.Response{
				Body:       ioutil.NopCloser(strings.NewReader("")),
				StatusCode: http.StatusNotFound,
			}, nil)

		store := &httpStore{
			address: _testURL,
			headers: nil,
			client:  &http.Client{Transport: transport},
		}
		val, err := store.Get("k")
		require.NoError(t, err)
		require.Equal(t, "", val)
	})

	t.Run("set_then_get", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		transport := mockhttp.NewMockRoundTripper(ctrl)
		transport.EXPECT().RoundTrip(gomock.Any()).
			Return(&http.Response{
				Body:       ioutil.NopCloser(strings.NewReader("")),
				StatusCode: http.StatusOK,
			}, nil)

		transport.EXPECT().RoundTrip(gomock.Any()).
			Return(&http.Response{
				Body:       ioutil.NopCloser(strings.NewReader("v")),
				StatusCode: http.StatusOK,
			}, nil)

		store := &httpStore{
			address: _testURL,
			headers: nil,
			client:  &http.Client{Transport: transport},
		}
		err := store.Put("k", "v")
		require.NoError(t, err)

		val, err := store.Get("k")
		require.NoError(t, err)
		require.Equal(t, "v", val)
	})
}
