package wray

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
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

// IMessage is an interface to a message
type IMessage interface {
	Data() map[string]interface{}
	Channel() string
}

// Subscription models a subscription, containing the channel it is subscribed
// to and the chan object used to push messages through
type Subscription struct {
	channel string
	msgChan chan Message
}

// MessageWaiter describes an object that will block until a message is available
// to return to the caller, allowing use in for loops similar to chans.
type MessageWaiter interface {
	WaitForMessage() IMessage
}

// the message waiter object that satisfied the matching interface
type messageWaiter struct {
	msgChan chan Message
}

// WaitorMessage blocks until there is a message available to return to the caller
func (w messageWaiter) WaitForMessage() IMessage {
	return <-w.msgChan
}

// FayeClient models a faye client
type FayeClient struct {
	state         int
	url           string
	subscriptions []*Subscription
	transport     Transport
	clientID      string
	schedular     Schedular
	nextRetry     int64
	nextHandshake int64
	mutex         *sync.RWMutex // protects instance vars across goroutines
	connectMutex  *sync.RWMutex // ensures a single connection to the server as per the protocol
}

// NewFayeClient returns a new client for interfacing to a faye server
func NewFayeClient(url string) *FayeClient {
	schedular := ChannelSchedular{}
	return &FayeClient{
		url:          url,
		state:        UNCONNECTED,
		schedular:    schedular,
		mutex:        &sync.RWMutex{},
		connectMutex: &sync.RWMutex{},
	}
}

func (faye *FayeClient) whileConnectingBlockUntilConnected() {
	if faye.state == CONNECTING {
		for {
			if faye.state == CONNECTED {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

func (faye *FayeClient) handshake() {

	// uh oh spaghettios!
	if faye.state == DISCONNECTED {
		panic("Server told us not to reconnect")
	}

	// check if we need to wait before handshaking again
	if faye.nextHandshake > time.Now().Unix() {
		sleepFor := time.Now().Unix() - faye.nextHandshake

		// wait for the duration the server told us
		if sleepFor > 0 {
			fmt.Println("Waiting for", sleepFor, "seconds before next handshake")
			time.Sleep(time.Duration(sleepFor) * time.Second)
		}
	}

	fmt.Println("Handshaking....")

	t, err := SelectTransport(faye, MANDATORY_CONNECTION_TYPES, []string{})
	if err != nil {
		panic("No usable transports available")
	}

	faye.mutex.Lock()
	faye.transport = t
	faye.transport.setUrl(faye.url)
	faye.state = CONNECTING
	faye.mutex.Unlock()

	handshakeParams := map[string]interface{}{"channel": "/meta/handshake",
		"version":                  "1.0",
		"supportedConnectionTypes": []string{"long-polling"}}

	response, err := faye.transport.send(handshakeParams)

	if err != nil {
		fmt.Println("Handshake failed. Retry in 10 seconds")

		faye.mutex.Lock()
		faye.state = UNCONNECTED
		faye.mutex.Unlock()

		time.Sleep(10 * time.Second)
		faye.handshake()

		return
	}

	faye.mutex.Lock()
	oldClientID := faye.clientID
	faye.clientID = response.clientId
	faye.state = CONNECTED
	faye.transport, err = SelectTransport(faye, response.supportedConnectionTypes, []string{})
	faye.mutex.Unlock()

	if err != nil {
		panic("Server does not support any available transports. Supported transports: " + strings.Join(response.supportedConnectionTypes, ","))
	}

	if oldClientID != faye.clientID && len(faye.subscriptions) > 0 {
		fmt.Printf("Client ID changed (%s => %s), %d invlaid subscriptions\n", oldClientID, faye.clientID, len(faye.subscriptions))
		faye.resubscribeAll()
	}
}

// change the state in a thread safe manner
func (faye *FayeClient) changeState(state int) {
	faye.mutex.Lock()
	defer faye.mutex.Unlock()
	faye.state = state
}

// TODO: check the bayeux spec to see if the retry period counts for all requests
// func (faye *FayeClient) makeRequest(data map[string]interface{}) (Response, error) {
// 	faye.connectMutex.Lock()
// 	defer faye.connectMutex.Unlock()
//
// 	// wait to retry if we were told to
// 	if faye.nextRetry > time.Now().Unix() {
// 		sleepFor := faye.nextRetry - time.Now().Unix()
// 		if sleepFor > 0 {
// 			// fmt.Println("Waiting for", sleepFor, "seconds before connecting")
// 			time.Sleep(time.Duration(sleepFor) * time.Second)
// 		}
// 	}
//
// 	return faye.transport.send(subscriptionParams)
// }

// resubscribe all of the subscriptions
func (faye *FayeClient) resubscribeAll() {

	faye.mutex.Lock()
	subs := faye.subscriptions
	faye.subscriptions = []*Subscription{}
	faye.mutex.Unlock()

	fmt.Printf("Attempting to resubscribe %d subscriptions\n", len(subs))
	for _, sub := range subs {

		// fork off all the resubscribe requests
		go func(sub *Subscription) {
			for {
				err := faye.requestSubscription(sub.channel)

				// if it worked add it back to the list
				if err == nil {
					faye.mutex.Lock()
					defer faye.mutex.Unlock()
					faye.subscriptions = append(faye.subscriptions, sub)

					fmt.Println("Resubscribed to", sub.channel)
					return
				}

				time.Sleep(500 * time.Millisecond)
			}
		}(sub)

	}
}

// requests a subscription from the server and returns error if the request failed
func (faye *FayeClient) requestSubscription(channel string) error {
	faye.whileConnectingBlockUntilConnected()
	if faye.state == UNCONNECTED {
		faye.handshake()
	}

	subscriptionParams := map[string]interface{}{"channel": "/meta/subscribe", "clientId": faye.clientID, "subscription": channel, "id": "1"}

	faye.connectMutex.Lock()
	defer faye.connectMutex.Lock()
	res, err := faye.transport.send(subscriptionParams)
	go faye.handleAdvice(res.advice)

	if !res.successful {
		// TODO: put more information in the error message about why it failed
		errmsg := "Response was unsuccessful"
		if err != nil {
			errmsg += err.Error()
		}
		reserr := errors.New(errmsg)
		return reserr
	}

	return nil
}

// handles a response from the server
func (faye *FayeClient) handleResponse(response Response) {
	for _, message := range response.messages {
		for _, subscription := range faye.subscriptions {
			matched, _ := filepath.Match(subscription.channel, message.Channel())
			if matched {
				go func() { subscription.msgChan <- message }()
			}
		}
	}
}

// handles advice from the server
func (faye *FayeClient) handleAdvice(advice Advice) {
	faye.mutex.Lock()
	defer faye.mutex.Unlock()

	if advice.reconnect != "" {
		interval := advice.interval

		switch advice.reconnect {
		case "retry":
			if interval > 0 {
				faye.nextHandshake = int64(time.Duration(time.Now().Unix()) + (time.Duration(interval) * time.Millisecond))
			}
		case "handshake":
			faye.state = UNCONNECTED // force a handshake on the next request
			if interval > 0 {
				faye.nextHandshake = int64(time.Duration(time.Now().Unix()) + (time.Duration(interval) * time.Millisecond))
			}
		case "none":
			faye.state = DISCONNECTED
			panic("Server advised not to reconnect")
		}
	}
}

// connects to the server and waits for a response.  Will block if it is waiting
// for the nextRetry time as advised by the server.  This locks the connectMutex
// so other connections can't go through until the
func (faye *FayeClient) connect() {
	faye.connectMutex.Lock()
	defer faye.connectMutex.Unlock()

	connectParams := map[string]interface{}{"channel": "/meta/connect", "clientId": faye.clientID, "connectionType": faye.transport.connectionType()}
	response, _ := faye.transport.send(connectParams)

	// take the advice given to us by the server
	faye.handleAdvice(response.advice)

	if response.successful {
		go faye.handleResponse(response)
	} else {
		faye.changeState(UNCONNECTED)
	}
}

// Subscribe to a channel
func (faye *FayeClient) Subscribe(channel string) (MessageWaiter, error) {

	err := faye.requestSubscription(channel)
	if err != nil {
		return nil, err
	}

	msgChan := make(chan Message)
	subscription := &Subscription{channel: channel, msgChan: msgChan}
	waiter := &messageWaiter{msgChan}

	// don't add to the subscriptions until we know it succeeded
	faye.mutex.Lock()
	defer faye.mutex.Unlock()
	faye.subscriptions = append(faye.subscriptions, subscription)

	return waiter, nil
}

// WaitSubscribe will send a subscribe request and block until the connection was successful
func (faye *FayeClient) WaitSubscribe(channel string) MessageWaiter {

	for {
		waiter, err := faye.Subscribe(channel)

		if err == nil {
			return waiter
		}
	}
}

// Publish a message to the given channel
func (faye *FayeClient) Publish(channel string, data map[string]interface{}) {
	faye.whileConnectingBlockUntilConnected()
	if faye.state == UNCONNECTED {
		faye.handshake()
	}

	publishParams := map[string]interface{}{"channel": channel, "data": data, "clientId": faye.clientID}
	response, _ := faye.transport.send(publishParams)

	faye.handleAdvice(response.advice)
}

// Listen starts listening for subscription requests from the server.  It is
// blocking but can safely run in it's own goroutine.
func (faye *FayeClient) Listen() {
	for {
		faye.whileConnectingBlockUntilConnected()
		if faye.state == UNCONNECTED {
			faye.handshake()
		}

		for {
			if faye.state != CONNECTED {
				break
			}

			// wait to retry if we were told to
			if faye.nextRetry > time.Now().Unix() {
				sleepFor := faye.nextRetry - time.Now().Unix()
				if sleepFor > 0 {
					// fmt.Println("Waiting for", sleepFor, "seconds before connecting")
					time.Sleep(time.Duration(sleepFor) * time.Second)
				}
			}

			faye.connect()
		}
	}
}

// RegisterTransports allows for the dynamic loading of different transports
// and the most suitable one will be selected
func RegisterTransports(transports []Transport) {
	registeredTransports = transports
}
