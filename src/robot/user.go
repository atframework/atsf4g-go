// client.go
package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	libatapp "github.com/atframework/libatapp-go"
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
	rpcChan            map[uint64]chan string
	csLog              *libatapp.LogBufferedRotatingWriter
}

var CurrentUser *User

func CreateUser(openId string) {
	bufferWriter, _ := libatapp.NewlogBufferedRotatingWriter(
		"../log", openId, 1*1024*1024, 3, time.Second*3, false, false)
	runtime.SetFinalizer(bufferWriter, func(writer *libatapp.LogBufferedRotatingWriter) {
		writer.Close()
	})

	conn, _, err := websocket.DefaultDialer.Dial(socketUrl, nil)
	if err != nil {
		log.Fatal("Error connecting to Websocket Server:", err)
	}

	CurrentUser = &User{
		OpenId:             openId,
		UserId:             0,
		ZoneId:             1,
		AccessToken:        fmt.Sprintf("access-token-for-%s", openId),
		connectionSequence: 99,
		connection:         conn,
		rpcChan:            make(map[uint64]chan string),
		csLog:              bufferWriter,
	}

	go receiveHandler(CurrentUser)
	log.Println("Create User:", openId)
}

func GetCurrentUser() *User {
	return CurrentUser
}

func (u *User) IsLogin() bool {
	if u == nil {
		return false
	}
	if u.Closed.Load() {
		return false
	}
	if !u.Logined {
		return false
	}
	return true
}

func (u *User) CmdHelpInfo() string {
	if u == nil {
		return "Need Login,CMD: login opendid"
	}
	return ""
}

func (u *User) CheckPingTask() {
	if !u.Logined {
		return
	}
	if u.LastPingTime.Add(u.HeartbeatInterval).Before(time.Now()) {
		err := PingRpc(u)
		if err != nil {
			log.Println("ping error stop check")
			return
		}
	}
	time.AfterFunc(5*time.Second, u.CheckPingTask)
}

func (u *User) Logout() {
	if !u.Logined {
		return
	}
	u.Logined = false
	u.Closed.Store(true)

	if u.connection != nil {
		// Close our websocket connection
		err := u.connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("Error during closing websocket:", err)
			return
		}
	}
}
