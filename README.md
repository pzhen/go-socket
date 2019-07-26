# go-socket

golang 实现的横向可扩展消息推送程序

# 简介

1.采用golang实现消息并发推送到客户端,
2.客户端接入更简单,无需告知服务器是否中断通信,服务器为每个client发送心跳检测,自动识别.
3.结合 Nginx 转发,可实现分布式部署


![struct](https://github.com/pzhen/go-socket/blob/master/struct.jpg)
          

# 运行

```
go run main.go
```

# 潜在问题
    
* 计算机本身的发包瓶颈,以及解码编码资源的损耗问题

    消息前置合并, 减少编码CPU损耗, 减少系统网络调用 

