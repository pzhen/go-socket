package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"golang.org/x/net/websocket"
	"log"
	"net/http"
	"time"
)

type Error struct {
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

type Client struct {
	UserId    string
	Timestamp int64
	conn      *websocket.Conn
	wsServer  *WsServer
}

type Message struct {
	UserId  string `json:"user_id"`
	Message string `json:"message"`
}

type WsServer struct {
	Clients map[string][]*Client
	AddCli  chan *Client
	DelCli  chan *Client
	Message chan *Message
}

func NewWsServer() *WsServer {
	return &WsServer{
		make(map[string][]*Client),
		make(chan *Client),
		make(chan *Client),
		make(chan *Message, 1000),
	}
}

func (wsServer *WsServer) MessageHandler(w http.ResponseWriter, r *http.Request) {
	m := &Message{}
	err := json.NewDecoder(r.Body).Decode(m)
	defer r.Body.Close()
	if nil != err {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(&Error{"params_error", "params invalid"})
		return
	}

	log.Printf("message reveived, user_id: %s, message: %s", m.UserId, m.Message)
	wsServer.Message <- &Message{m.UserId, m.Message}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(m)
}

func (c *Client) heartbeat() error {
	millis := time.Now().UnixNano() / 1000000
	heartbeat := struct {
		Heartbeat int64 `json:"heartbeat"`
	}{millis}
	bytes, _ := json.Marshal(heartbeat)
	_, err := c.conn.Write(bytes)
	return err
}

func (c *Client) Listen() {
	for range time.Tick(5 * time.Second) {
		err := c.heartbeat()
		if nil != err {
			log.Printf("client heartbeat error, user_id: %v, timestamp: %d, err: %s", c.UserId, c.Timestamp, err)
			c.wsServer.DelCli <- c
			return
		}
	}
}

func (wsServer *WsServer) WsClientHandler(conn *websocket.Conn) {
	userId := conn.Request().URL.Query().Get("user_id")
	defer conn.Close()
	if len(userId) > 0 {
		millis := time.Now().UnixNano() / 1000000
		c := &Client{userId, millis, conn, wsServer}
		wsServer.AddCli <- c
		c.Listen()
	}
}

func (wsServer *WsServer) SendMessage(userId, message string) {
	clients := wsServer.Clients[userId]
	if len(clients) > 0 {
		for _, c := range clients {
			c.conn.Write([]byte(message))
		}
		log.Printf("message success sent to client, user_id: %s", userId)
	} else {
		log.Printf("client not found, user_id: %s", userId)
	}
}

func (wsServer *WsServer) addClient(c *Client) {
	clients := wsServer.Clients[c.UserId]
	wsServer.Clients[c.UserId] = append(clients, c)
	log.Printf("a client added, userId: %s, timestamp: %d", c.UserId, c.Timestamp)
}

func (wsServer *WsServer) delClient(c *Client) {
	clients := wsServer.Clients[c.UserId]
	if len(clients) > 0 {
		for i, client := range clients {
			if client.Timestamp == c.Timestamp {
				wsServer.Clients[c.UserId] = append(clients[:i], clients[i+1:]...)
				break
			}
		}
	}
	if 0 == len(clients) {
		delete(wsServer.Clients, c.UserId)
	}
	log.Printf("a client deleted, user_id: %s, timestamp: %d", c.UserId, c.Timestamp)
}

func (wsServer *WsServer) Start() {
	for {
		select {
		case msg := <-wsServer.Message:
			wsServer.SendMessage(msg.UserId, msg.Message)
		case c := <-wsServer.AddCli:
			wsServer.addClient(c)
		case c := <-wsServer.DelCli:
			wsServer.delClient(c)
		}
	}
}

var port = flag.Int("serverPort", 29999, "server port")

func main() {
	flag.Parse()
	wsServer := NewWsServer()

	go wsServer.Start()
	log.Printf("wsServer running...")
	log.Printf("wsServer server port %d", *port)

	r := mux.NewRouter()
	r.Handle("/ws", websocket.Handler(wsServer.WsClientHandler))
	r.HandleFunc("/message", wsServer.MessageHandler).Methods(http.MethodPost)
	r.Headers("Content-Type", "application/json; charset=UTF-8")

	log.Printf("httpServer running...")
	log.Printf("httpServer receive api '/message' ")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), r))
}
