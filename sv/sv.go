package main

import (
  "fmt"
  "net/http"
  "io"
  "strings"
  "golang.org/x/net/websocket"
  "time"
)

/* 
 * Server 
 */

const (
  /* bigger batch size seems better because it avoids artificial congestion when 
     there is too many clients, knowing the batching time window will 
     dispatch the messages soon enough anyway.
     In other words, the bigger the batch size, the better. */
  MAX_MESSAGE_BATCH_COUNT = 100000
)

type Server struct {
  // FIXME
  // Put clients in database?
  Clients           map[*websocket.Conn]*Client
  Usernames         map[string]*Client
  History           *History
  MessageBatch      [MAX_MESSAGE_BATCH_COUNT]*ClientSentString
  // FIXME
  // Use slice instead
  MessageBatchCount int
  SpawnClient       chan *SpawnClient
  KillClient        chan *Client
  ClaimUsername     chan ClientSentString
  GetHistory        chan *Client
  SendMessage       chan ClientSentString
  SendBatch         chan bool
  KillServer        chan bool
}

func NewServer() *Server {
  sv                  := &Server{}
  sv.Clients           = make(map[*websocket.Conn]*Client)
  sv.Usernames         = make(map[string]*Client)
  sv.History           = NewHistory()
  sv.MessageBatchCount = 0
  sv.SpawnClient       = make(chan *SpawnClient)
  sv.KillClient        = make(chan *Client)
  sv.ClaimUsername     = make(chan ClientSentString)
  sv.GetHistory        = make(chan *Client)
  sv.SendMessage       = make(chan ClientSentString)
  sv.SendBatch         = make(chan bool)
  sv.KillServer        = make(chan bool)
  return sv
}

type Client struct {
  Ws          *websocket.Conn
  Username    *string
  /* supposely anti DDOS, cause server is blocking on 'History.MakeJSON' */
  GotHistory  bool
  SendJSON    chan string
  KillIt      chan bool
  /* "websocket get closed when flow quit handler" problem */
  CloseWs     chan bool
}

// FIXME
// Implement Codec instead of using 'variant'
/* This is a variant. 
   'Type' tells what fields can be read in the union. */
type ClientCmd struct {
  Type      string `json:"type"`
  Username  string `json:"username"`
  Message   string `json:"message"`
  Ping      string `json:"ping"`
}

type ClientSentString struct {
  Cl        *Client
  String    string
}

/* this class is here because of the 
   "websocket get closed when flow quit handler" 
   problem */
type SpawnClient struct {
  Ws      *websocket.Conn
  CloseWs chan bool
}

const (
  CLCMD_CLAIM_USERNAME = "claimUsername"
  CLCMD_GET_HISTORY    = "getHistory"
  CLCMD_SEND_MESSAGE   = "sendMessage"
  CLCMD_PING           = "ping"
)

func (this *Server) ServerController() {
  // FIXME
  // Implement server shutdown
  go this.BatchDispatcher()
  for {
    select {
      case spawnClient := <-this.SpawnClient:
        cl := &Client{
          spawnClient.Ws,
          nil,
          false,
          make(chan string),
          make(chan bool),
          spawnClient.CloseWs,
        }
        this.Clients[cl.Ws]  = cl
        go this.ClientController(cl)
      case cl := <-this.KillClient:
        delete(this.Clients, cl.Ws)
        cl.CloseWs <- true /* "websocket get closed when flow quit handler" problem */
      case clientSentString := <-this.ClaimUsername:
        cl        := clientSentString.Cl
        username  := clientSentString.String
        free      := this.IsUsernameFree(username)
        if free {
          this.SetUsername(cl, username)
        }
        cl.SendJSON <- fmt.Sprintf(
          `{"type": "%s", "success": %t}`, 
          CLCMD_CLAIM_USERNAME, free)
      case cl := <-this.GetHistory:
        cl.SendJSON <- fmt.Sprintf(
          `{"type": "%s", "history": %s}`, 
          CLCMD_GET_HISTORY, this.History.MakeJSON())
      case clientSentString := <-this.SendMessage:
        cl          := clientSentString.Cl
        message     := clientSentString.String
        this.History.AddHistoryMessage(*cl.Username, message)
        this.AddMessageToBatch(&clientSentString)
      case <-this.SendBatch:
        this.DoSendBatch()
    }
  }
}

func (this *Server) BatchDispatcher() {
  // FIXME
  // Implement server shutdown
  for {
    time.Sleep(250 * 1024 * 1024 * time.Nanosecond)
    this.SendBatch <- true
  }
}

/* 'ClientController' is mostly here because of the idea that 
   we can parallelize the websocket 'send' calls.
   Problem is, I haven't really tested the assumption that it's *significantly* faster.
   I'm planning to write a bot for testing performance anyway. */
func (this *Server) ClientController(cl *Client) {
  fmt.Printf("Client got connected. Total client count: %d\n", len(this.Clients))
  /* 'websocket.JSON.Receive' is blocking so we can't 
     call it in the default-case of the following select. */
  go this.ClientListen(cl)
  for {
    select {
      case json := <-cl.SendJSON:
        //fmt.Printf("Client sending json %s\n", json)
        websocket.JSON.Send(cl.Ws, json)
      /* 'ClientListen' is responsible 
         for touching the 'KillIt' channel; 
         'ClientListen' would have stopped the 'ClientController' 
         forever loop in the default-case instead 
         if a separate goroutine wasn't required.
         See: 'ClientListen' */
      case <-cl.KillIt:
        this.KillClient <- cl
        return
    }
  }
}

/* 'websocket.JSON.Receive' is blocking so we can't 
   call it in the default-case of client controller select. 
   So we've made a separate goroutine.
   'ClientListen' is responsible for stopping the 
   'ClientController' forever loop by touching the 'KillIt' 
   channel the same way it would have broke the forever 
   loop in default-case */
func (this *Server) ClientListen(cl *Client) {
  done := false
  for !done {
    var cmd ClientCmd
    err := websocket.JSON.Receive(cl.Ws, &cmd)
    if err == nil {
      //fmt.Printf("Client cmd recieved: '%s'.\n", cmd.Type)
      this.ExecClientCmd(cl, &cmd)
    } else {
      if err == io.EOF {
        fmt.Print("Client got disconnected.\n")
      } else {
        fmt.Printf("Client error: %s.\n", err)
      }
      done = true
    }
  }
  cl.KillIt <- true
}

func (this *Server) ExecClientCmd(cl *Client, cmd *ClientCmd) {
  if cl.Username == nil {
    switch cmd.Type {
      case CLCMD_CLAIM_USERNAME:
        this.ClaimUsername <- ClientSentString{cl, cmd.Username}
      case CLCMD_GET_HISTORY, CLCMD_SEND_MESSAGE, CLCMD_PING:
        fmt.Printf("Client without a name is trying to use chatroom.\n")
    }
  } else {
    switch cmd.Type {
      case CLCMD_CLAIM_USERNAME:
        fmt.Printf("Client with a username is trying to claim another username.\n")
      case CLCMD_GET_HISTORY:
        /* supposely anti DDOS, cause server is blocking on 'History.MakeJSON' */
        if !cl.GotHistory {
          cl.GotHistory = true
          this.GetHistory <- cl
        } else {
          fmt.Printf("Client which got history is trying to get history again.\n")
        }
      case CLCMD_SEND_MESSAGE:
        this.SendMessage <- ClientSentString{cl, cmd.Message}
      case CLCMD_PING:
        cl.SendJSON <- fmt.Sprintf(`{"type": "%s", "ping": "%s"}`, 
          CLCMD_PING, cmd.Ping)
    }
  }
}

func (this *Server) IsUsernameFree(username string) bool {
  _, there := this.Usernames[username]
  return !there
}

func (this *Server) SetUsername(cl *Client, username string) {
  this.Usernames[username] = cl
  cl.Username = &username
}

func (this *Server) DoSendBatch() {
  if this.MessageBatchCount > 0 {
    json := this.MakeJSONFromBatch()
    for _, cl2  := range this.Clients { 
      cl2.SendJSON <- fmt.Sprintf(
        `{"type": "%s", "messages": %s}`,
        CLCMD_SEND_MESSAGE, json)
    }
    this.MessageBatchCount = 0
  }
}

func (this *Server) AddMessageToBatch(clientSentString *ClientSentString) {
  this.MessageBatch[this.MessageBatchCount] = clientSentString
  this.MessageBatchCount++
  if this.MessageBatchCount >= MAX_MESSAGE_BATCH_COUNT {
    this.DoSendBatch()
  }
}

/* I think that's okay that this function is similar to 'History.MakeJSON'... */
func (this *Server) MakeJSONFromBatch() string {
  result := "["
  /* this.MessageBatchCount is > 0 because we dont want to sent an empty array anyway.. */
  messages := make([]string, this.MessageBatchCount)
  for i := 0; i < this.MessageBatchCount; i++ {
    clientSentString := this.MessageBatch[i]
    messages[i] = fmt.Sprintf(`{"author": "%s", "message": "%s"}`,
      *clientSentString.Cl.Username, clientSentString.String)
  }
  result += strings.Join(messages, ",")
  result += "]"
  return result
}

/* 
 * History 
 */

const (
  MAX_HISTORY_MESSAGES = 100
)

type History struct {
  /* cannot be array of 'ClientSentString's because 
     clients could be disconnected and pointers invalid */
  historyMessages [MAX_HISTORY_MESSAGES]*HistoryMessage
  next            int
}

type HistoryMessage struct {
  Author   string
  Message  string
}

func NewHistory() *History {
  history := &History{}
  history.next = 0
  return history
}

func (this *History) AddHistoryMessage(author, message string) {
  historyMessage := &HistoryMessage{author, message}
  this.historyMessages[this.next] = historyMessage
  this.next = (this.next+1)%MAX_HISTORY_MESSAGES
}

func (this *History) MakeJSON() string {
  result := "["
  /* if we ever put something in the history, first 'historyMessage' wont be nil */
  if this.historyMessages[0] != nil {
    var first, count int
    /* if we've never set the last historyMessage of the array, we haven't looped yet */
    if this.historyMessages[MAX_HISTORY_MESSAGES-1] == nil {
      first = 0
      count = this.next
    } else {
      first = this.next
      count = MAX_HISTORY_MESSAGES
    }
    messages := make([]string, count)
    for i := 0; i < count; i++ {
      historyMessage := this.historyMessages[(first+i)%MAX_HISTORY_MESSAGES]
      messages[i] = fmt.Sprintf(`{"author": "%s", "message": "%s"}`,
        historyMessage.Author, historyMessage.Message)
    }
    result += strings.Join(messages, ",")
  }
  result += "]"
  return result
}

/* 
 * main 
 */

func main() {
  fmt.Print("Server starting up.\n")
  sv := NewServer()
  go sv.ServerController()
  http.Handle("/ws/", websocket.Handler(func(ws *websocket.Conn) {
    spawnClient := &SpawnClient{ws, make(chan bool)}
    sv.SpawnClient <- spawnClient
    // SUPER FIXME
    // Find a way to remove that s***
    <-spawnClient.CloseWs
  }))
  http.ListenAndServe(":3000", nil)
}