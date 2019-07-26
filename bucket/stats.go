package bucket

import "sync/atomic"

type Stats struct {
	ConnectionTotal      int64 `json:"connection_total"`        //在线链接总数
	MessageBufferTotal   int64 `json:"message_buffer_total"`    //待推送消息总数
	MessageSendFailTotal int64 `json:"message_send_fail_total"` //发送失败总数
}

var (
	GlobalStats *Stats
)

func InitStats() {
	GlobalStats = &Stats{}
}

func ConnectionTotal_INCR() {
	atomic.AddInt64(&GlobalStats.ConnectionTotal, 1)
}

func ConnectionTotal_DESC() {
	atomic.AddInt64(&GlobalStats.ConnectionTotal, -1)
}

func MessageBufferTotal_INCR() {
	atomic.AddInt64(&GlobalStats.MessageBufferTotal, 1)
}

func MessageBufferTotal_DESC() {
	atomic.AddInt64(&GlobalStats.MessageBufferTotal, -1)
}

func MessageSendFailTotal_INCR() {
	atomic.AddInt64(&GlobalStats.MessageSendFailTotal, 1)
}