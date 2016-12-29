package main

import (
  "fmt"
  "io"
  "golang.org/x/net/websocket"
  "github.com/nu7hatch/gouuid"
  "strconv"
  "encoding/json"
  "time"
)

/* 
 * Bot 
 */

// FIXME: dedupe
const (
  TOTAL_BOTS_COUNT = 10000
	SERVER_PORT = "3000"
)

// FIXME: dedupe
const (
  CLCMD_CLAIM_USERNAME = "claimUsername"
  CLCMD_GET_HISTORY    = "getHistory"
  CLCMD_SEND_MESSAGE   = "sendMessage"
  CLCMD_PING           = "ping"
)

type Bot struct {
	Ws       *websocket.Conn
  Username string
}

/* This is a variant */
type ServerCmd struct {
  Type      string     `json:"type"`
  Success   bool       `json:"success"`
  Messages  []Message  `json:"messages"`
  Author    string     `json:"author"`
  Ping      string     `json:"ping"`
}

type Message struct {
  Author    string `json:"author"`
  Message   string `json:"message"`
}

func NewBot() *Bot {
	bot             := &Bot{}
	ws, err         := websocket.Dial("ws://127.0.0.1:" + SERVER_PORT + "/ws/", "", "http://127.0.0.1/")
  if err == nil {
    fmt.Print("Bot successfully connected to server.\n")
    bot.Ws        = ws
    uuid, _       := uuid.NewV4()
    bot.Username  = "Bot-" + uuid.String()
    json          := fmt.Sprintf(`{"type": "%s", "username": "%s"}`, CLCMD_CLAIM_USERNAME, bot.Username)
    websocket.Message.Send(bot.Ws, []byte(json))
  } else {
    fmt.Print("Something went wrong, bot can't connect.\n")
    bot           = nil
  }
	return bot
}

func (bot *Bot) BotListen() {
  done := false
  for !done {
    var in []byte
    err := websocket.Message.Receive(bot.Ws, &in)
    if err == nil {
      unquoted, _ := strconv.Unquote(string(in))
      var cmd ServerCmd
      err = json.Unmarshal([]byte(unquoted), &cmd)
      if err == nil {
        //fmt.Printf("Server cmd recieved: '%s'.\n", cmd.Type)
        switch cmd.Type {
          case CLCMD_CLAIM_USERNAME:
            if cmd.Success {
              fmt.Printf("'ClaimUsername' SUCCESS for bot %s.\n", bot.Username)
            } else {
              fmt.Printf("'ClaimUsername' ERROR for bot %s.\n", bot.Username)
            }
          case CLCMD_SEND_MESSAGE:
            //for i := range cmd.Messages {
              //message := cmd.Messages[i]
              //fmt.Printf("%s: %s\n", message.Author, message.Message)
            //}
          case CLCMD_PING:
            ping, _ := strconv.ParseInt(cmd.Ping, 10, 64)
            fmt.Printf("ping is %fs\n", float64(time.Now().UnixNano() - ping)/(1000*1000*1000))
        }
      } else {
        fmt.Printf("Unmarshal error: %s.\n", err)
      }
    } else {
      if err == io.EOF {
        fmt.Print("Bot got disconnected.\n")
      } else {
        fmt.Printf("Bot error: %s.\n", err)
      }
      done = true
    }
  }
}

func (bot *Bot) BotController() {
  for {
    time.Sleep(1 * time.Second)
    json := fmt.Sprintf(`{"type": "%s", "ping": "%d"}`, 
      CLCMD_PING, time.Now().UnixNano())
    websocket.Message.Send(bot.Ws, []byte(json))
    json = fmt.Sprintf(`{"type": "%s", "message": "%s"}`, 
      CLCMD_SEND_MESSAGE, "Hello, I'm " + bot.Username + ", you wanna come at my place?")
    websocket.Message.Send(bot.Ws, []byte(json))
  }
}

func (bot *Bot) Run() {
  go bot.BotListen()
  go bot.BotController()
}

/* 
 * main 
 */

func main() {
  fmt.Print("Bots starting up.\n")
  for i := 0; i < TOTAL_BOTS_COUNT; i++ {
    time.Sleep(200 * 1024 * 1024 * time.Nanosecond)
    NewBot().Run()
  }
  time.Sleep(3600 * time.Second)
}