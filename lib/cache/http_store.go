package cache

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type httpStore struct {
	address string
	headers map[string]string
	client  *http.Client
}

// NewHTTPStore returns a new instance of KVStore backed by a server that
// implements the following API:
// GET <address>/key => http.StatusOK with value in body
// PUT <address>/key => 200 <= code < 300
// The "headers" entries are of the form <header>:<value>.
func NewHTTPStore(address string, headers ...string) (KVStore, error) {
	headerMap := map[string]string{}
	for _, tuple := range headers {
		split := strings.SplitN(tuple, ":", 2)
		if len(split) != 2 {
			return nil, fmt.Errorf("Malformed http header: %s, format is <header>:<value>", tuple)
		}
		headerMap[strings.TrimSpace(split[0])] = strings.TrimSpace(split[1])
	}
	store := &httpStore{
		address: address,
		headers: headerMap,
		client:  http.DefaultClient,
	}
	return store, nil
}

func (store *httpStore) Get(key string) (string, error) {
	key = base64.URLEncoding.EncodeToString([]byte(key))
	url := fmt.Sprintf("%s/%s", store.address, key)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	store.addHeaders(req)

	resp, err := store.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status code from cache server: %d", resp.StatusCode)
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read cache server response: %s", err)
	}
	return string(content), nil
}

func (store *httpStore) Put(key, value string) error {
	key = base64.URLEncoding.EncodeToString([]byte(key))
	url := fmt.Sprintf("%s/%s", store.address, key)
	body := strings.NewReader(value)
	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return fmt.Errorf("failed to create cache request: %s", err)
	}
	store.addHeaders(req)

	resp, err := store.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 || resp.StatusCode < 200 {
		return fmt.Errorf("bad status code from cache server: %d", resp.StatusCode)
	}
	return nil
}

func (store *httpStore) Cleanup() error { return nil }

func (store *httpStore) addHeaders(req *http.Request) {
	for k, v := range store.headers {
		req.Header.Set(k, v)
	}
}
