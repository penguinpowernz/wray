package wray

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

// HTTPTransport models a faye protocol transport over HTTP long polling
type HTTPTransport struct {
	url string
}

func (t HTTPTransport) isUsable(clientURL string) bool {
	parsedURL, err := url.Parse(clientURL)
	if err != nil {
		return false
	}
	if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
		return true
	}
	return false
}

func (t HTTPTransport) connectionType() string {
	return "long-polling"
}

func (t HTTPTransport) send(msg Message) (decoder, error) {
	buffer := bytes.NewBuffer([]byte{})
	json.NewEncoder(buffer).Encode(msg)
	responseData, err := http.Post(t.url, "application/json", buffer)
	if err != nil {
		return nil, err
	}
	if responseData.StatusCode != 200 {
		return nil, errors.New(responseData.Status)
	}
	defer responseData.Body.Close()
	return json.NewDecoder(responseData.Body), nil
}

func (t *HTTPTransport) setURL(url string) {
	t.url = url
}
