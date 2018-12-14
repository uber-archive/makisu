package cache

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
