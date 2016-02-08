package wray

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	UNCONNECTED  = 1
	CONNECTING   = 2
	CONNECTED    = 3
	DISCONNECTED = 4

	HANDSHAKE = "handshake"
	RETRY     = "retry"
	NONE      = "none"

	CONNECTION_TIMEOUT = 60.0
	DEFAULT_RETRY      = 5.0
	MAX_REQUEST_SIZE   = 2048
)

var (
	MANDATORY_CONNECTION_TYPES = []string{"long-polling"}
	registeredTransports       = []Transport{}
)

type FayeClient struct {
	state         int
	url           string
	subscriptions []Subscription
	transport     Transport
	clientId      string
	schedular     Schedular
	nextRetry     int64
	nextHandshake int64
}

type Subscription struct {
	channel  string
	callback func(Message)
}

type SubscriptionPromise struct {
	subscription Subscription
	subError     error
}

func (self SubscriptionPromise) Error() error {
	return self.subError
}

func (self SubscriptionPromise) Successful() bool {
	return self.subError == nil
}

func NewFayeClient(url string) *FayeClient {
	schedular := ChannelSchedular{}
	client := &FayeClient{url: url, state: UNCONNECTED, schedular: schedular}
	return client
}

func (self *FayeClient) whileConnectingBlockUntilConnected() {
	if self.state == CONNECTING {
		for {
			if self.state == CONNECTED {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func (self *FayeClient) handshake() {

	if self.state == DISCONNECTED {
		panic("Server told us not to reconnect")
	}

	// check if we need to wait before handshaking again
	if self.nextHandshake > time.Now().Unix() {
		sleepFor := time.Now().Unix() - self.nextHandshake
		
		// wait for the duration the server told us
		if sleepFor > 0 {
			fmt.Println("Waiting for",sleepFor, "seconds before next handshake")
			time.Sleep(time.Duration(sleepFor) * time.Second)
		}
	}

  fmt.Println("Handshaking....")
	
	t, err := SelectTransport(self, MANDATORY_CONNECTION_TYPES, []string{})
	if err != nil {
		panic("No usable transports available")
	}
	self.transport = t
	self.transport.setUrl(self.url)
	self.state = CONNECTING
	handshakeParams := map[string]interface{}{"channel": "/meta/handshake",
		"version":                  "1.0",
		"supportedConnectionTypes": []string{"long-polling"}}
	response, err := self.transport.send(handshakeParams)
	if err != nil {
		fmt.Println("Handshake failed. Retry in 10 seconds")
		self.state = UNCONNECTED
		self.schedular.wait(10*time.Second, func() {
			fmt.Println("Retying handshake")
			self.handshake()
		})
		return
	}
	self.clientId = response.clientId
	self.state = CONNECTED
	self.transport, err = SelectTransport(self, response.supportedConnectionTypes, []string{})
	if err != nil {
		panic("Server does not support any available transports. Supported transports: " + strings.Join(response.supportedConnectionTypes, ","))
	}
}

func (self *FayeClient) Subscribe(channel string, force bool, callback func(Message)) (promise SubscriptionPromise, err error) {
	self.whileConnectingBlockUntilConnected()
	if self.state == UNCONNECTED {
		self.handshake()
	}
	subscriptionParams := map[string]interface{}{"channel": "/meta/subscribe", "clientId": self.clientId, "subscription": channel, "id": "1"}
	subscription := Subscription{channel: channel, callback: callback}

	res, err := self.transport.send(subscriptionParams)
	
	self.handleAdvice(res.advice)

	promise = SubscriptionPromise{subscription, nil}

	if err != nil {
		promise.subError = err
		return
	}

	if !res.successful {
		// TODO: put more information in the error message about why it failed
		err = errors.New("Response was unsuccessful")
		promise.subError = err
		return
	}

	// don't add to the subscriptions until we know it succeeded
	self.subscriptions = append(self.subscriptions, subscription)

	return
}

// Send a subscribe request, but if it fails keep retrying until it succeeds, then return a promise.
// This will block until the subscription is successful.
func (self *FayeClient) WaitSubscribe(channel string, callback func(Message)) SubscriptionPromise {

	for {
		promise, err := self.Subscribe(channel, false, callback)

		if promise.Successful() {
			return promise
		}
	}
}

// Send a subscribe request and if it fails, keep trying.  On success it will fire the callback with a promise object.
// This will block until the subscription is successful.
func (self *FayeClient) SubscribeThen(channel string, callback func(Message), then func(SubscriptionPromise)) {
	for {
		promise, err := self.Subscribe(channel, false, callback)

		if promise.Successful() {
			then(promise)
		}
	}
}

func (self *FayeClient) handleResponse(response Response) {
	for _, message := range response.messages {
		for _, subscription := range self.subscriptions {
			matched, _ := filepath.Match(subscription.channel, message.Channel)
			if matched {
				go subscription.callback(message)
			}
		}
	}
}

func (self *FayeClient) connect() {
	connectParams := map[string]interface{}{"channel": "/meta/connect", "clientId": self.clientId, "connectionType": self.transport.connectionType()}

  fmt.Println("Connecting... waiting for response...")
  response, _ := self.transport.send(connectParams)

  fmt.Println("got a response")
  fmt.Printf("%+v\n", response)

  // take the advice given to us by the server
	self.handleAdvice(response.advice)

	if response.successful {
		go self.handleResponse(response)
	} else {
		self.state = UNCONNECTED
	}
}

func (self *FayeClient) handleAdvice(advice Advice) {

  if advice.reconnect != "" {
  	interval := advice.interval

	  switch(advice.reconnect) {
		case "retry":
			if interval > 0 {
				self.nextHandshake = int64(time.Duration(time.Now().Unix()) + (time.Duration(interval) * time.Millisecond))
			}
		case "handshake":
			self.state = UNCONNECTED // force a handshake on the next request
		  if interval > 0 {
		  	self.nextHandshake = int64(time.Duration(time.Now().Unix()) + (time.Duration(interval) * time.Millisecond))
		  }
		case "none":
			self.state = DISCONNECTED
			panic("Server advised not to reconnect")
	  }
	}
}

func (self *FayeClient) Listen() {
	for {
		self.whileConnectingBlockUntilConnected()
		if self.state == UNCONNECTED {
			self.handshake()
		}

		for {
			if self.state != CONNECTED {
				break
			}

			// wait to retry if we were told to
			if self.nextRetry > time.Now().Unix() {
				sleepFor := self.nextRetry - time.Now().Unix()
				if sleepFor > 0 {
					fmt.Println("Waiting for",sleepFor, "seconds before connecting")
					time.Sleep(time.Duration(sleepFor) * time.Second)
				}
			}

			self.connect()
		}
	}
}

func (self *FayeClient) Publish(channel string, data map[string]interface{}) {
	self.whileConnectingBlockUntilConnected()
	if self.state == UNCONNECTED {
		self.handshake()
	}
	publishParams := map[string]interface{}{"channel": channel, "data": data, "clientId": self.clientId}
	response, _ := self.transport.send(publishParams)

	self.handleAdvice(response.advise)
}

func RegisterTransports(transports []Transport) {
	registeredTransports = transports
}
