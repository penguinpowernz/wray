package wray

import (
  "strconv"
)

type Advice struct {
  interval int
  reconnect string
  timeout int
}

type Response struct {
  id                       string
  channel                  string
  successful               bool
  clientId                 string
  supportedConnectionTypes []string
  messages                 []Message
  advice                   Advice
  error                    error
}

type Message struct {
  Channel string
  Id      string
  Data    map[string]interface{}
}

func newResponse(data []interface{}) Response {
  headerData := data[0].(map[string]interface{})
  messagesData := data[1.:]
  messages := parseMessages(messagesData)

  var id string
  if headerData["id"] != nil {
    id = headerData["id"].(string)
  }

  supportedConnectionTypes := []string{}

  if headerData["supportedConnectionTypes"] != nil {
    d := headerData["supportedConnectionTypes"].([]interface{})
    for _, sct := range d {
      supportedConnectionTypes = append(supportedConnectionTypes, sct.(string))
    }
  }

  var clientId string
  if headerData["clientId"] != nil {
    clientId = headerData["clientId"].(string)
  }

  res := Response{
    id:                       id,
    clientId:                 clientId,
    channel:                  headerData["channel"].(string),
    successful:               headerData["successful"].(bool),
    messages:                 messages,
    supportedConnectionTypes: supportedConnectionTypes,
  }

  parseAdvice(headerData, &res)

  return res
}

func parseAdvice(data map[string]interface{}, res *Response) {

  _advice, exists := data["advice"]

  if !exists {
    return
  }

  advice := _advice.(map[string]interface{})
  
  reconnect, exists := advice["reconnect"]
  if exists {
    res.advice.reconnect = reconnect.(string)
  }

  interval, exists := advice["interval"]
  if exists {
    res.advice.interval, _ = strconv.Atoi(interval.(string))
  }

  timeout, exists := advice["timeout"]
  if exists {
    res.advice.timeout, _ = strconv.Atoi(timeout.(string))
  }
}

func parseMessages(data []interface{}) []Message {
  messages := []Message{}

  for _, messageData := range data {

    if messageData == nil {
      continue;
    }

    m := messageData.(map[string]interface{})
    var id string

    if m["id"] != nil {
      id = m["id"].(string)
    }

    message := Message{
      Channel: m["channel"].(string),
      Id:      id,
    }
    
    if m["data"] != nil {
      message.Data = m["data"].(map[string]interface{})
    }

    messages = append(messages, message)
  }

  return messages
}
