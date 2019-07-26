package config

const (
	Env                   = "Test" // Test/Prod
	HeartbeatTime         = 60     //心跳间隔
	BucketNum             = 10     // 1024
	BucketJobWorkerCount  = 32     // 消息处理协程个数
	BucketBuffWorkerCount = 32     // 待发消息处理协成个数
	BucketBuffChannelSize = 10000  // 待转发队列容量100000
	BucketJobChannelSize  = 1000   // 每个chan队列容量
)
