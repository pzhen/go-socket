// websocket 消息推送
// 使用方法:
// 1.开启服务 go run main.go
// 2.业务程序调用 http api
// 接口来发送数据 (curl -d '{"user_id": "10", "message": "test_user_10"}' http://localhost:29999/message)
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// 链接的映射池
var clients sync.Map

type Message struct {
	UserId  string `json:"user_id"`
	Message string `json:"message"`
}

type MsgError struct {
	Code   int `json:"code"`
	Reason string `json:"reason"`
}

// 客户端链接
type Connection struct {
	clientId  int64
	wsConn    *websocket.Conn
	inChan    chan []byte
	outChan   chan []byte
	closeChan chan byte
	mutex     sync.Mutex
	isClosed  bool
}

// 读取
func (client *Connection) ReadMessage() (data []byte, err error) {
	select {
	case data = <-client.inChan:
	case <-client.closeChan:
		err = errors.New("connection is closed")
	}
	return
}

// 发送
func (client *Connection) WriteMessage(data []byte) (err error) {
	select {
	case client.outChan <- data:
	case <-client.closeChan:
		err = errors.New("connection is closed")
	}
	return
}

// 关闭
func (client *Connection) Close() {
	// 线程安全的Close
	client.wsConn.Close()
	client.mutex.Lock()
	if !client.isClosed {
		close(client.closeChan)
		client.isClosed = true
	}
	client.mutex.Unlock()
}

// 对原始读取封装,读到数据放到inchan
func (client *Connection) readLoop() {
	var (
		data []byte
		err  error
	)
	for {
		if _, data, err = client.wsConn.ReadMessage(); err != nil {
			goto ERR
		}

		select {
		case client.inChan <- data:
		case <-client.closeChan:
			goto ERR
		}
	}

ERR:
	client.Close()
}

func (client *Connection) writeLoop() {
	var (
		data []byte
		err  error
	)
	for {
		select {
		case data = <- client.outChan:
		case <- client.closeChan:
			goto ERR
		}

		if err = client.wsConn.WriteMessage(websocket.TextMessage, data); err != nil {
			goto ERR
		}
	}
ERR:
	client.Close()
}

func NewConnection(wsConn *websocket.Conn) (conn *Connection, err error)  {
	conn = &Connection{
		wsConn: wsConn,
		inChan: make(chan []byte, 1000),
		outChan: make(chan []byte, 1000),
		closeChan: make(chan byte, 1),
	}
	go conn.readLoop()
	go conn.writeLoop()
	return
}



// 协议升级配置
var upGrader = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// socket处理函数
func wsHandler(w http.ResponseWriter, r *http.Request) {
	var(
		wsConn *websocket.Conn
		err error
		conn *Connection
		userId int64
		data []byte
	)
	if wsConn, err = upGrader.Upgrade(w, r, nil);err != nil {
		return
	}

	if conn, err = NewConnection(wsConn); err != nil {
		goto ERR
	}

	if err = conn.WriteMessage([]byte("welcome to connection ...")); err != nil {
		goto ERR
	}

	r.ParseForm()
	if len(r.Form["user_id"]) <= 0 {
		goto ERR
	}

	if userId, err = strconv.ParseInt(r.Form["user_id"][0],10,64); err != nil {
		goto ERR
	}

	clients.LoadOrStore(userId, conn)
	log.Printf("user_id %d is connecting...\n", userId)

	// 心跳检测
	go func(uid int64) {
		var (
			err error
		)
		for {
			if err = conn.WriteMessage([]byte("heartbeat...")); err != nil {
				clients.Delete(uid)
				conn.Close()
				fmt.Printf("user_id %d is go away ...\n", uid)
				return
			}
			time.Sleep(1 * time.Second)
		}

	}(userId)

	for {
		if data, err = conn.ReadMessage(); err != nil {
			goto ERR
		}

		if err = conn.WriteMessage(data); err != nil {
			goto ERR
		}
	}

	ERR:
		conn.Close()

}

// 发送消息接口
func msgHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var(
		conn *Connection
		err error
		msg *Message
		userId int64
		value interface{}
		ok bool
	)

	msg = &Message{}
	if err := json.NewDecoder(r.Body).Decode(msg);err != nil {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(&MsgError{0, "message invalid"})
		fmt.Println("message is invalid...")
		return
	}

	if userId, err = strconv.ParseInt(msg.UserId,10,64); err != nil || userId <= 0 {
		fmt.Println("user_id is invalid...")
		return
	}

	if value, ok = clients.Load(userId); ok == false {
		fmt.Println("connection queue not having user_id " + strconv.FormatInt(userId,10))
		return
	}

	conn = value.(*Connection)
	if err = conn.WriteMessage([]byte(msg.Message));err != nil {
		fmt.Println("send message '"+msg.Message+"' fail...")
		return
	}
}

func main()  {
	var (
		// 服务监听地址
		addr = "0.0.0.0:29999"
		// websocket 地址
		wsAddr = "/ws"
		// 消息推送地址
		httpAddr = "/message"
	)

	log.Printf("server addr '%s' ...\n",addr)
	log.Printf("receive api '%s' ...\n",httpAddr)
	log.Printf("webservice api '%s' ...\n",wsAddr)

	http.HandleFunc(wsAddr,wsHandler)
	http.HandleFunc(httpAddr,msgHandler)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe", err.Error())
	}
}
