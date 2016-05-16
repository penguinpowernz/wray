package wray

import (
  // "strconv"
  "encoding/json"
)

// Models a message sent over Faye
type Message interface {
  Channel() string
  ID()      string
  Data()    map[string]interface{}
  Ext()     map[string]interface{}
  ConnectionType() string
  Decode(interface{}) error
}

// Models a response received from the Faye server
type Response() interface {
  Successful() bool
  Channel()    string
  Error()      string
  Advice()     Advice
  ClientID()   string
  SupportedConnectionTypes() []string
}

// Advice given by the Bayeux server about how to reconnect, etc
type Advice interface {
  Interval() int
  Reconnect() string
  Timeout() int
}

// Models a Bayeux message that can be for sending or receiving
// TODO: omitempty
type message struct {
  ID                       string `json:"id"`
  Channel                  string `json:"channel"`
  Successful               bool `json:"successful"`
  ClientID                 string `json:"clientId"`
  SupportedConnectionTypes []string `json:"supportedConnectionTypes"`
  Data                     map[string]interface{}
  Advice                   advice `json:"advice"`
  Error                    error `json:"error"`
  decoder                  decoder
  Ext                      map[string]interface{} `json:"ext"`
  ConnectionType           string `json:"connectionType"`
  Subscription             string `json:"subscription"`
  Version                  string `json:"version"`
}

type msgWrapper struct {
  msg *message
}


func (w msgWrapper) Data() map[string]interface{} { return w.msg.Data }
func (w msgWrapper) ID() string { return w.msg.ID }
func (w msgWrapper) Channel() string { return w.msg.Channel }

// Ext returns a map of extension data.  As it is a map (and thus a pointer), changes
// to the returned object will modify the content of this field in the message
func (w msgWrapper) Ext() map[string]interface{} { return w.msg.Ext }
func (w msgWrapper) OK() bool { return w.msg.Successful }
func (w msgWrapper) Error() string { return w.msg.Error }
func (w msgWrapper) HasError() bool { return w.msg.Error == "" }
func (w msgWrapper) SetError(msg string) { w.msg.Error = msg }
func (w msgWrapper) ConnectionType() string { return w.msg.ConnectionType }
func (w msgWrapper) Subscription() string { return w.msg.Subscription }
func (w msgWrapper) Advice() Advice { return w.msg.Advice }
func (w msgWrapper) Version() string { return w.msg.Version }
func (w msgWrapper) SupportedConnectionTypes() []string { return w.msg.SupportedConnectionTypes }
func (w msgWrapper) ConnectionType() string { return w.msg.ConnectionType }
func (w msgWrapper) ClientID() string { return w.msg.ClientID }

// Decodes the message data into the given pointer
func (w msgWrapper) Decode(obj interface{}) {
  bytes, err := json.Marshal(w.Data())
  if err != nil {
    return err
  }

  return json.NewDecoder(bytes).Decode(obj)
}

func decodeResponse(dec decoder) (Response, []Message, error) {
  var msgs = []Message{}
  if err := dec.Decode(&jsonData); err != nil {
    return nil, msgs, err
  }

  return messages[0], messages[0:], nil
}

type advice struct {
  Reconnect string `json:"reconnect"`
  Interval float64 `json:"interval"`
  Timeout float64 `json:"timeout"`
}

