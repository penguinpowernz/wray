package wray

import (
	"encoding/json"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func params2msg(params map[string]interface{}) Message {
	b, err := json.Marshal(params)
	if err != nil {
		panic(err)
	}
	msg := message{}
	json.Unmarshal(b, &msg)
	return msgWrapper{&msg}
}

func TestIsUsable(t *testing.T) {
	Convey("usable if the url is http", t, func() {
		var url string
		var httpTransport HTTPTransport
		var isUsable = true
		Given(func() { url = "http://localhost" })
		Given(func() { httpTransport = HTTPTransport{} })
		When(func() { isUsable = httpTransport.isUsable(url) })
		Then(func() { So(isUsable, ShouldEqual, true) })
	})
	Convey("usable if the url is https", t, func() {
		var url string
		var httpTransport HTTPTransport
		var isUsable = true
		Given(func() { url = "https://localhost" })
		Given(func() { httpTransport = HTTPTransport{} })
		When(func() { isUsable = httpTransport.isUsable(url) })
		Then(func() { So(isUsable, ShouldEqual, true) })
	})
	Convey("usable if the url is ws", t, func() {
		var url string
		var httpTransport HTTPTransport
		var isUsable = true
		Given(func() { url = "ws://localhost" })
		Given(func() { httpTransport = HTTPTransport{} })
		When(func() { isUsable = httpTransport.isUsable(url) })
		Then(func() { So(isUsable, ShouldEqual, false) })
	})
	Convey("usable if the url is not blank", t, func() {
		var url string
		var httpTransport HTTPTransport
		var isUsable = true
		Given(func() { url = "" })
		Given(func() { httpTransport = HTTPTransport{} })
		When(func() { isUsable = httpTransport.isUsable(url) })
		Then(func() { So(isUsable, ShouldEqual, false) })
	})
}

func TestConnectionType(t *testing.T) {
	Convey("connection type is long-polling", t, func() {
		var httpTransport HTTPTransport
		Given(func() { httpTransport = HTTPTransport{} })
		Then(func() { So(httpTransport.connectionType(), ShouldEqual, "long-polling") })
	})
}
