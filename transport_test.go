package wray

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestSelectTransport(t *testing.T) {
	Convey("when all transports are usable", t, func() {
		var transportTypes []string
		var transport Transport
		var fakeHTTPTransport *FakeHTTPTransport
		var fayeClient *FayeClient

		Given(func() { transportTypes = []string{"long-polling"} })
		Given(func() { fakeHTTPTransport = &FakeHTTPTransport{usable: true} })
		Given(func() { fayeClient = &FayeClient{url: "http://localhost"} })
		Given(func() { registeredTransports = []Transport{fakeHTTPTransport} })
		When(func() { transport, _ = selectTransport(fayeClient, transportTypes, []string{}) })
		Then(func() { So(transport, ShouldEqual, fakeHTTPTransport) })
	})

	Convey("when no transports are usable", t, func() {
		var transportTypes []string
		var fakeHTTPTransport *FakeHTTPTransport
		var fayeClient *FayeClient
		var err error

		Given(func() { transportTypes = []string{"long-polling"} })
		Given(func() { fakeHTTPTransport = &FakeHTTPTransport{usable: false} })
		Given(func() { fayeClient = &FayeClient{url: "http://localhost"} })
		Given(func() { registeredTransports = []Transport{fakeHTTPTransport} })
		When(func() { _, err = selectTransport(fayeClient, transportTypes, []string{}) })
		Then(func() { So(err, ShouldNotBeNil) })
	})
}
