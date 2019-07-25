// websocket 消息推送
// 	配合 nginx 对 websocket 进行转发,可以实现分布式部署
//
// 使用方法:
// 	1.开启服务 go run main.go
// 	2.业务程序调用 http api
// 接口来发送数据 (curl -d '{"user_id": "10", "message": "test_user_10"}' http://localhost:29999/message)
package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	_ "github.com/mkevac/debugcharts"
	"go-socket/hook"
	"go-socket/wsconn"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"sync"
	"time"
)

// dev/prod
const (
	Env = "Test"
)

// 链接的映射池
var clients sync.Map

type Message struct {
	UserId  string `json:"user_id"`
	Message string `json:"message"`
}

type MsgError struct {
	Code   int    `json:"code"`
	Reason string `json:"reason"`
}

// 协议升级配置
var upGrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// socket处理函数
func wsHandler(w http.ResponseWriter, r *http.Request) {
	var (
		wsConn *websocket.Conn
		err    error
		conn   *wsconn.Connection
		userId int64
		data   []byte
	)
	if wsConn, err = upGrader.Upgrade(w, r, nil); err != nil {
		log.Printf("[Error] %s \n", err.Error())
		return
	}

	if conn, err = wsconn.NewConnection(wsConn); err != nil {
		log.Printf("[Error] %s \n", err.Error())
		goto ERR
	}

	if err = conn.WriteMessage([]byte("welcome to connection ...")); err != nil {
		log.Printf("[Error] %s \n", err.Error())
		goto ERR
	}

	r.ParseForm()
	if len(r.Form["user_id"]) <= 0 {
		log.Println("[Error] user_id invalid ")
		goto ERR
	}

	if userId, err = strconv.ParseInt(r.Form["user_id"][0], 10, 64); err != nil {
		log.Printf("[Error] %s \n", err.Error())
		goto ERR
	}

	clients.LoadOrStore(userId, conn)
	log.Printf("[Info] user_id %d is connecting...\n", userId)

	// 心跳检测
	go func(uid int64) {
		var (
			err error
		)
		for {
			if err = conn.WriteMessage([]byte("heartbeat...")); err != nil {
				clients.Delete(uid)
				conn.Close()
				log.Printf("[Info] user_id %d fall away...\n", uid)
				return
			}
			time.Sleep(60 * time.Second)
		}

	}(userId)

	for {
		if data, err = conn.ReadMessage(); err != nil {
			log.Printf("[Error] %s \n", err.Error())
			goto ERR
		}

		if err = conn.WriteMessage(data); err != nil {
			log.Printf("[Error] %s \n", err.Error())
			goto ERR
		}
	}

ERR:
	conn.Close()

}

// 发送消息接口
func msgHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var (
		conn   *wsconn.Connection
		err    error
		msg    *Message
		userId int64
		value  interface{}
		ok     bool
	)

	msg = &Message{}
	if err := json.NewDecoder(r.Body).Decode(msg); err != nil {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(&MsgError{0, "message invalid"})
		log.Println("[Error] message invalid")
		return
	}

	if userId, err = strconv.ParseInt(msg.UserId, 10, 64); err != nil || userId <= 0 {
		log.Println("[Error] message of user_id invalid")
		return
	}

	if value, ok = clients.Load(userId); ok == false {
		log.Printf("[Error] user_id %s may be fall away \n", strconv.FormatInt(userId, 10))
		return
	}

	conn = value.(*wsconn.Connection)
	if err = conn.WriteMessage([]byte(msg.Message)); err != nil {
		log.Printf("[Error] send message '%s' fail \n", msg.Message)
		return
	}

	log.Printf("[Info] send message '%s' ok \n", msg.Message)
}

func init() {
	log.SetPrefix(Env + " - ")
	log.SetFlags(log.Ldate | log.Ltime | log.Llongfile)
}

func main() {
	var (
		// 服务监听地址
		addr = "127.0.0.1:29999"
		// websocket 地址
		wsAddr = "/ws"
		// 消息推送地址
		httpAddr = "/message"
	)

	http.HandleFunc(wsAddr, hook.HookRecover(wsHandler))
	http.HandleFunc(httpAddr, hook.HookRecover(hook.HookTime(msgHandler)))

	log.Println("[Info] websocket server is running...")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("[Error] %s", err.Error())
	}
}
