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

package httputil

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/mock/gomock"
	"github.com/pressly/chi"
	"github.com/stretchr/testify/require"

	"github.com/uber/makisu/mocks/net/http"
)

const _testURL = "http://localhost:0/test"

func startServer(t *testing.T) (string, func()) {
	require := require.New(t)
	l, err := net.Listen("tcp", ":0")
	require.NoError(err)
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})
	go http.Serve(l, r)
	return l.Addr().String(), func() { l.Close() }
}

func newResponse(status int) *http.Response {
	// We need to set a dummy request in the response so NewStatusError
	// can access the "original" URL.
	dummyReq, err := http.NewRequest("GET", _testURL, nil)
	if err != nil {
		panic(err)
	}

	rec := httptest.NewRecorder()
	rec.WriteHeader(status)
	resp := rec.Result()
	resp.Request = dummyReq

	return resp
}

func TestSendRetry(t *testing.T) {
	require := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transport := mockhttp.NewMockRoundTripper(ctrl)

	for _, status := range []int{503, 502, 200} {
		transport.EXPECT().RoundTrip(gomock.Any()).Return(newResponse(status), nil)
	}

	_, err := Get(
		_testURL,
		SendRetry(),
		SendTransport(transport))
	require.NoError(err)
}

func TestSendRetryOnTransportErrors(t *testing.T) {
	require := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transport := mockhttp.NewMockRoundTripper(ctrl)

	transport.EXPECT().RoundTrip(gomock.Any()).Return(nil, errors.New("some network error")).Times(4)

	_, err := Get(
		_testURL,
		SendRetry(),
		SendTransport(transport))
	require.Error(err)
}

func TestSendRetryWithCodes(t *testing.T) {
	require := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transport := mockhttp.NewMockRoundTripper(ctrl)

	gomock.InOrder(
		transport.EXPECT().RoundTrip(gomock.Any()).Return(newResponse(400), nil),
		transport.EXPECT().RoundTrip(gomock.Any()).Return(newResponse(503), nil),
		transport.EXPECT().RoundTrip(gomock.Any()).Return(newResponse(404), nil),
		transport.EXPECT().RoundTrip(gomock.Any()).Return(newResponse(500), nil), // Non-retryable.
	)

	_, err := Get(
		_testURL,
		SendRetry(
			RetryBackoff(backoff.WithMaxRetries(
				backoff.NewConstantBackOff(200*time.Millisecond),
				10)),
			RetryCodes(400, 404)),
		SendTransport(transport))
	require.Error(err)
	require.Equal(500, err.(StatusError).Status) // Last code returned.
}

func TestStatusChecking(t *testing.T) {
	err := StatusError{Status: http.StatusCreated}
	require.True(t, IsCreated(err))
	err = StatusError{Status: http.StatusNotFound}
	require.True(t, IsNotFound(err))
	err = StatusError{Status: http.StatusConflict}
	require.True(t, IsConflict(err))
	err = StatusError{Status: http.StatusAccepted}
	require.True(t, IsAccepted(err))
	err = StatusError{Status: http.StatusForbidden}
	require.True(t, IsForbidden(err))
}
