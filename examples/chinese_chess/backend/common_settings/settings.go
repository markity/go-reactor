package commsettings

// 下面是心跳配置, 最多耗时1s就能检测到对方是否断线
// 客户端和服务端丢需要检测, 双方都进行判定

// 单位秒, 发送心跳的频率ms
const HeartbeatInterval = 200

// 最大丢丢失心跳包的个数
const MaxLoseHeartbeat = 5

// 服务端的配置
const ServerListenIP = "0.0.0.0"
const ServerListenPort = 8080
