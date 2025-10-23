// client.go
package main

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type User struct {
	OpenId string
	UserId uint64
	ZoneId uint32

	AccessToken string
	LoginCode   string

	Logined           bool
	HasGetInfo        bool
	HeartbeatInterval time.Duration
	LastPingTime      time.Time
	Closed            atomic.Bool

	connectionSequence uint64
	connection         *websocket.Conn
	dispatcherLock     sync.Mutex
}
