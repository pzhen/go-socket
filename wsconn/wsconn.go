package wsconn

import (
	"errors"
	"github.com/gorilla/websocket"
	"log"
	"strconv"
	"sync"
)

type Connection struct {
	Uid       int64
	wsConn    *websocket.Conn
	inChan    chan []byte
	outChan   chan []byte
	closeChan chan byte
	mutex     sync.Mutex
	isClosed  bool
}

func (conn *Connection) ReadMessage() (data []byte, err error) {
	select {
	case data = <-conn.inChan:
	case <-conn.closeChan:
		err = errors.New("user " + strconv.Itoa(int(conn.Uid)) + " connection is closed")
	}
	return
}

func (conn *Connection) WriteMessage(data []byte) (err error) {
	select {
	case conn.outChan <- data:
	case <-conn.closeChan:
		err = errors.New("user " + strconv.Itoa(int(conn.Uid)) + " connection is closed")
	}
	return
}

func (conn *Connection) Close() {
	// 线程安全的Close
	conn.wsConn.Close()
	conn.mutex.Lock()
	if !conn.isClosed {
		close(conn.closeChan)
		conn.isClosed = true
	}
	conn.mutex.Unlock()
}

func (conn *Connection) readLoop() {
	var (
		data []byte
		err  error
	)
	for {
		if _, data, err = conn.wsConn.ReadMessage(); err != nil {
			log.Printf("[Error] readLoop user_id %d %s \n", conn.Uid, err)
			goto ERR
		}

		select {
		case conn.inChan <- data:
		case <-conn.closeChan:
			goto ERR
		}
	}

ERR:
	conn.Close()
}

func (conn *Connection) writeLoop() {
	var (
		data []byte
		err  error
	)
	for {
		select {
		case data = <-conn.outChan:
		case <-conn.closeChan:
			goto ERR
		}

		if err = conn.wsConn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[Error] writeLoop user_id %d %s \n", conn.Uid, err)
			goto ERR
		}
	}
ERR:
	conn.Close()
}

func NewConnection(wsConn *websocket.Conn) (conn *Connection, err error) {
	conn = &Connection{
		wsConn:    wsConn,
		inChan:    make(chan []byte, 1000),
		outChan:   make(chan []byte, 1000),
		closeChan: make(chan byte, 1),
	}
	go conn.readLoop()
	go conn.writeLoop()
	return
}
