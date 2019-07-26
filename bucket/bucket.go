package bucket

import (
	"go-socket/wsconn"
	"log"
	"sync"
)

type Bucket struct {
	rwMutex  sync.RWMutex
	bucketId int // 桶编号
	connMap  map[int64]*wsconn.Connection
}

func InitBucket(bucketId int) (bucket *Bucket) {
	bucket = &Bucket{
		bucketId: bucketId,
		connMap:  make(map[int64]*wsconn.Connection),
	}
	return
}

func (bucket *Bucket) AddBucket(connection *wsconn.Connection) {
	bucket.rwMutex.Lock()
	defer bucket.rwMutex.Unlock()
	bucket.connMap[connection.Uid] = connection
}

func (bucket *Bucket) DelBucket(connection *wsconn.Connection) {
	bucket.rwMutex.Lock()
	defer bucket.rwMutex.Unlock()

	delete(bucket.connMap, connection.Uid)
}

func (bucket *Bucket) PushMessage(pushMsg *PushMsg) {
	// 锁Bucket
	bucket.rwMutex.RLock()
	defer bucket.rwMutex.RUnlock()

	// 全量非阻塞推送
	for _, wsConn := range bucket.connMap {
		wsConn.WriteMessage([]byte(pushMsg.Msg))
		log.Printf("[Info] Send bucket_id %d user_id %d message %s \n", bucket.bucketId, wsConn.Uid,pushMsg.Msg)
	}
}
