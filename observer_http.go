package rx_go

import (
	"io/ioutil"
	"net/http"
)

// NewHttpObserver create http Observer from the request and client
func NewHttpObserver(client *http.Client, req *http.Request) (*Observer[[]byte], error) {
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp != nil {
		defer resp.Body.Close()
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	obs := NewObserver[[]byte]()
	obs.Next(bytes)
	return obs, nil
}
