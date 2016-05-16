package auth_extension

type authExtn struct {
  token string
}


// Out sets the auth token on the map in Ext()
func (extn authExtn) Out(msg Message) {
  ext := msg.Ext()
  ext["authToken"] = extn.token  // this change is
}

// In just satisfies the interface
func (extn authExtn) In(msg Message) {}


func main() {
  e : = authToken{"abcdef0123456789"}
  url := "http://localhost:8000/faye"

  faye = wray.NewClient(url)
  faye.AddExtension(e)

  go faye.Listen()

  // when sent this message will have the auth token added to it
  faye.Publish("/test", map[string]interface{}{"hello": "world"})
}
