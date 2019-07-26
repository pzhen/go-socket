package main

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	_ "github.com/mkevac/debugcharts"
	"go-socket/bucket"
	"go-socket/config"
	"go-socket/hook"
	"go-socket/wsconn"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"time"
)

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

	// 装桶
	conn.Uid = userId
	bucket.GlobalBucketSet.AddBucketSet(conn)

	log.Printf("[Info] user_id %d is connecting...\n", userId)

	// 心跳检测
	go func(conn *wsconn.Connection) {
		var (
			err error
		)
		for {
			if err = conn.WriteMessage([]byte("heartbeat...")); err != nil {
				bucket.GlobalBucketSet.DelBucketSet(conn)
				conn.Close()
				log.Printf("[Info] user_id %d fall away...\n", conn.Uid)
				return
			}
			time.Sleep(60 * time.Second)
		}

	}(conn)

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
	var pushMsgs []*bucket.PushMsg
	pushMsgs = make([]*bucket.PushMsg, 0)

	if err := json.NewDecoder(r.Body).Decode(&pushMsgs); err != nil {
		w.WriteHeader(400)
		json.NewEncoder(w).Encode(&MsgError{0, "message invalid"})
		log.Println("[Error] message invalid", err)
		return
	}

	for _, v := range pushMsgs {
		bucket.GlobalBucketSet.BuffChan <- v
	}

}

func init() {
	log.SetPrefix(config.Env + " - ")
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
	bucket.InitBucketSet()
	http.HandleFunc(wsAddr, hook.HookRecover(wsHandler))
	http.HandleFunc(httpAddr, hook.HookRecover(hook.HookTime(msgHandler)))

	log.Println("[Info] websocket server is running...")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("[Error] %s", err.Error())
	}
}
