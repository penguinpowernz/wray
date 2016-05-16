package wray

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
)

type HttpTransport struct {
	url string
}

func (self HttpTransport) isUsable(clientUrl string) bool {
	parsedUrl, err := url.Parse(clientUrl)
	if err != nil {
		return false
	}
	if parsedUrl.Scheme == "http" || parsedUrl.Scheme == "https" {
		return true
	}
	return false
}

func (self HttpTransport) connectionType() string {
	return "long-polling"
}

type decoder interface{
  Decode(interface{}) error
}

type encoder interface{
  Encode() ([]byte, error)
}

func (self HttpTransport) send(enc encoder) (decoder, error) {
	dataBytes, _ := enc.Encode()
	buffer := bytes.NewBuffer(dataBytes)
	responseData, err := http.Post(self.url, "application/json", buffer)
	if err != nil {
		return nil, err
	}
	if responseData.StatusCode != 200 {
		return nil, errors.New(responseData.Status)
	}
	readData, _ := ioutil.ReadAll(responseData.Body)
	responseData.Body.Close()
	return json.NewDecoder(readData), nil
}

func (self *HttpTransport) setUrl(url string) {
	self.url = url
}
