package examples

import (
  "github.com/autogrowsystems/wray"
  "fmt"
)

func RunPublisherExample() {
  wray.RegisterTransports([]wray.Transport{ &wray.HttpTransport{} })
  client := wray.NewFayeClient("http://localhost:5000/faye")

  params := map[string]interface{}{"hello": "from golang"}
  fmt.Println("sending")
  client.Publish("/foo", params)
}


