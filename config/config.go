package config

const (
	Env                   = "Test" // Test/Prod
	BucketNum             = 2
	BucketJobWorkerCount  = 2 // 消息处理协程个数
	BucketBuffWorkerCount = 2 // 待发消息处理协成个数
	BucketBuffChannelSize = 1000
	BucketJobChannelSize  = 1000
)
