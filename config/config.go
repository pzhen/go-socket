package config

const (
	Env                   = "Test" // Test/Prod
	BucketNum             = 10     // 1024
	BucketJobWorkerCount  = 32     // 消息处理协程个数
	BucketBuffWorkerCount = 32     // 待发消息处理协成个数
	BucketBuffChannelSize = 1000
	BucketJobChannelSize  = 1000
)
