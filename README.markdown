# Wray, faye client for Go

Wray is a client for the [Faye](http://faye.jcoglan.com) publish-subscribe messaging service by [James Coglan](https://twitter.com/jcoglan).

## Current status

Beta.

## Getting Started

Wray is only a client for Faye. You will need to setup a server using Ruby or Node.js first. Instructions that can be found on the [Faye project pages](http://faye.jcoglan.com/).  Simple examples are available in the examples folder.

### Subscribing to channels

This is some sample code for subscribing to a channel.  Wildcard channels ar permissible too (eg: `/foo/*`).

```go
package main

import "github.com/pythonandchips/wray"
import "fmt"
import "time"

func main() {
  //register the types of transport you want available. Only long-polling is currently supported
  wray.RegisterTransports([]wray.Transport{ &wray.HTTPTransport{} })

  //create a new client
  client := wray.NewFayeClient("http://localhost:5000/faye")
  go client.Listen()

  //subscribe to the channels you want to listen to
  promise, err := client.Subscribe("/foo")
  if err != nil {
    fmt.Println("Subscription to /foo failed", err)
  }

  go func() {
    for {
      msg := subscription.WaitForMessage()
      fmt.Println("-------------------------------------------")
      fmt.Println(msg.Data())
    }
  }()

  // guarantee a subscription works by blocking until subscription request is received by server
  promise := client.WaitSubscribe("/foo/*")
  msg := subscription.WaitForMessage()
  fmt.Println(msg.Data())
}
```

### Publishing to channels

```go
package main

import "github.com/pythonandchips/wray"
import "fmt"

func main() {
  //register the types of transport you want available. Only long-polling is currently supported
  wray.RegisterTransports([]wray.Transport{ &wray.HTTPTransport{} })

  //create a new client
  client := wray.NewFayeClient("http://localhost:5000/faye")

  params := map[string]interface{}{"hello": "from golang"}

  //send message to server
  if err := client.Publish("/foo", params); err != nil {
    fmt.Println(err)
  }
}
```

### Extensions

Extension must satisfy the `faye.Extenstion` interface:

```go
type Extension interface {
  In(Message)
  Out(Message)
}
```

Then you can write your extension like so:

```go

wray.RegisterTransports([]wray.Transport{ &wray.HTTPTransport{} })

type authExtension struct {
  token string
}

// In does nothing in this extension, but is needed to satisy the interface
func (e *authExtension) In(msg wray.Message) {}

// Out adds the authentication token to the messages ext field
func (e *authExtension) Out(msg wray.Message) {
  ext := msg.Ext()
  ext["token"] = e.token
}

func main() {
  faye := wray.NewFayeClient("http://localhost:8000/faye")
  faye.AddExtension(authExtension{"Y29tZSBhdCBtZSBicm8K"})

  // this message will now reach the server with the authentication token attached
  faye.Publish("/test", map[string]interface{}{"hello": "world"})
}
```

## Future Work

There is still a lot to do to bring Wray in line with Faye functionality. This is a less than exhaustive list of work to be completed:-

- [x] fix the resubscribe after rehandshake logic to work with channels instead of callbacks
- [] web socket support
- [] eventsource support
- [] logging
- [] stop having to register transports
- [] middleware additions
- [x] extensions support
- [] correctly handle disconnect and server down
- [] promises for subscription and publishing
- [] automated integrations test to ensure Wray continues to work with Faye

## Contributing

* Check out the latest master to make sure the feature hasn't been implemented or the bug hasn't been fixed yet
* Check out the issue tracker to make sure someone already hasn't requested it and/or contributed it
* Fork the project
* Start a feature/bugfix branch
* Commit and push until you are happy with your contribution
* Make sure to add tests for it. This is important so I don't break it in a future version unintentionally.

## Copyright

Copyright (c) 2014 Colin Gemmell

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
