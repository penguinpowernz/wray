package wray

import (
	"errors"
)

type decoder interface {
	Decode(interface{}) error
}

// Transport models a faye protocol transport
type Transport interface {
	isUsable(string) bool
	connectionType() string
	send(Message) (decoder, error)
	setURL(string)
}

func selectTransport(client *FayeClient, transportTypes []string, disabled []string) (Transport, error) {
	for _, transport := range registeredTransports {
		if contains(transport.connectionType(), transportTypes) && transport.isUsable(client.url) {
			return transport, nil
		}
	}
	return nil, errors.New("No usable transports available")
}
