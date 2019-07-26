package bucket

import (
	"go-socket/config"
	"go-socket/wsconn"
)

type PushMsg struct {
	Msg string `json:"msg"`
}

var (
	GlobalBucketSet *BucketSet
)

// 桶的集合
type BucketSet struct {
	buckets  []*Bucket
	jobChan  []chan *PushMsg // 每个桶对应一个chan,消息队列
	BuffChan chan *PushMsg   // 待发消息队列
}

// 初始化桶集合
func InitBucketSet() {
	globalBucketSet := &BucketSet{
		buckets:  make([]*Bucket, config.BucketNum),
		jobChan:  make([]chan *PushMsg, config.BucketNum),
		BuffChan: make(chan *PushMsg, config.BucketBuffChannelSize),
	}

	for bucketId := range globalBucketSet.buckets {
		globalBucketSet.buckets[bucketId] = InitBucket(bucketId)
		globalBucketSet.jobChan[bucketId] = make(chan *PushMsg, config.BucketJobChannelSize)
		for i := 0; i < config.BucketJobWorkerCount; i++ {
			go globalBucketSet.JobWorker(i, bucketId)
		}
	}

	for i := 0; i < config.BucketBuffWorkerCount; i++ {
		go globalBucketSet.BuffWorker(i)
	}

	GlobalBucketSet = globalBucketSet
}

// 并发的从chan中获取消息并发送消息
func (bucketSet *BucketSet) JobWorker(workerId int, bucketId int) {
	var (
		bucket = bucketSet.buckets[bucketId]
		msg    *PushMsg
	)

	for {
		select {
		case msg = <-bucketSet.jobChan[bucketId]:
			bucket.PushMessage(msg)
		}
	}
}

// 将消息放入每个桶对应的chan队列中
func (bucketSet *BucketSet) BuffWorker(workerId int) {
	var bucketId int
	for {
		select {
		case pushJob := <-bucketSet.BuffChan:
			// 为每个桶都生成一份消息
			for bucketId = range bucketSet.buckets {
				bucketSet.jobChan[bucketId] <- pushJob
			}
		}
	}
}

// 连接->桶集合->对应桶的connMap
func (bucketSet *BucketSet) AddBucketSet(connection *wsconn.Connection) {
	bucketSet.GetBucket(connection).AddBucket(connection)
}

// 从对应桶的connMap中删除连接
func (bucketSet *BucketSet) DelBucketSet(connection *wsconn.Connection) {
	bucketSet.GetBucket(connection).DelBucket(connection)
}

// 获取某个桶
func (bucketSet *BucketSet) GetBucket(connection *wsconn.Connection) (bucket *Bucket) {
	return bucketSet.buckets[connection.Uid%config.BucketNum]
}
