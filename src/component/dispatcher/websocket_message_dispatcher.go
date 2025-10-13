package atframework_component_dispatcher

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	libatapp "github.com/atframework/libatapp-go"

	private_protocol_config "github.com/atframework/atsf4g-go/component-protocol-private/config/protocol/config"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

type WebSocketSession struct {
	SessionId uint64

	Connection *websocket.Conn

	Authorized       bool
	sentCloseMessage bool

	runningContext context.Context
	runningCancel  context.CancelFunc

	sendQueue    chan *public_protocol_extension.CSMsg
	errorCounter int

	PrivateData interface{}
}

type (
	WebSocketCallbackOnNewSession    = func(ctx *RpcContext, session *WebSocketSession) error
	WebSocketCallbackOnRemoveSession = func(ctx *RpcContext, session *WebSocketSession)
	WebSocketCallbackOnNewMessage    = func(session *WebSocketSession, message *public_protocol_extension.CSMsg) error
)

type WebSocketMessageDispatcher struct {
	DispatcherBase

	serverConfig *private_protocol_config.WebserverCfg
	wsConfig     *private_protocol_config.WebsocketServerCfg
	upgrader     *websocket.Upgrader
	upgraderLock sync.RWMutex

	sessions           map[uint64]*WebSocketSession
	sessionIdAllocator atomic.Uint64
	sessionLock        sync.Mutex

	webServerHandle   *http.ServeMux
	webServerInstance *http.Server
	webServerAddress  string

	stopContext context.Context
	stopCancel  context.CancelFunc

	onNewSession    atomic.Value
	onRemoveSession atomic.Value
	onNewMessage    atomic.Value
}

func CreateCSMessageWebsocketDispatcher(owner libatapp.AppImpl) *WebSocketMessageDispatcher {
	// 使用时间戳作为初始值, 避免与重启前的值冲突
	ret := &WebSocketMessageDispatcher{
		DispatcherBase: CreateDispatcherBase(owner),

		sessions:           make(map[uint64]*WebSocketSession),
		sessionIdAllocator: atomic.Uint64{},
	}

	ret.sessionIdAllocator.Store(uint64(time.Since(time.Unix(int64(private_protocol_pbdesc.EnSystemLimit_EN_SL_TIMESTAMP_FOR_ID_ALLOCATOR_OFFSET), 0)).Nanoseconds()))

	return ret
}

func (d *WebSocketMessageDispatcher) Name() string { return "WebSocketMessageDispatcher" }

func (d *WebSocketMessageDispatcher) Init(initCtx context.Context) error {
	err := d.DispatcherBase.Init(initCtx)
	if err != nil {
		return err
	}

	return d.setupListen()
}

func (d *WebSocketMessageDispatcher) setupUpgrader() {
	d.upgraderLock.Lock()
	defer d.upgraderLock.Unlock()

	d.upgrader = &websocket.Upgrader{
		HandshakeTimeout: d.wsConfig.HandshakeTimeout.AsDuration(),
		ReadBufferSize:   int(d.wsConfig.ReadBufferSize),
		WriteBufferSize:  int(d.wsConfig.WriteBufferSize),
		Subprotocols:     d.wsConfig.SubProtocols,
		CheckOrigin: func(r *http.Request) bool {
			// Configure origin checking for production
			return true
		},
		EnableCompression: d.wsConfig.EnableCompression,
	}
}

func (d *WebSocketMessageDispatcher) setupListen() error {
	if d.GetApp().IsClosing() || d.GetApp().IsClosed() {
		return fmt.Errorf("application is closing or closed")
	}

	d.setupUpgrader()

	if d.webServerHandle == nil {
		d.webServerHandle = http.NewServeMux()
		d.webServerHandle.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, d.wsConfig.Path) {
				http.NotFound(w, r)
				return
			}

			d.handleConnection(w, r)
		})
	}

	if d.webServerInstance != nil && d.webServerAddress != d.serverConfig.Host+":"+fmt.Sprintf("%d", d.serverConfig.Port) {
		d.webServerInstance.Close()
		d.webServerInstance = nil
	}

	if d.webServerInstance == nil {
		d.webServerAddress = d.serverConfig.Host + ":" + fmt.Sprintf("%d", d.serverConfig.Port)
		d.webServerInstance = &http.Server{
			Addr:         d.webServerAddress,
			Handler:      d.webServerHandle,
			ReadTimeout:  d.serverConfig.ReadTimeout.AsDuration(),
			WriteTimeout: d.serverConfig.WriteTimeout.AsDuration(),
			IdleTimeout:  d.serverConfig.IdleTimeout.AsDuration(),
		}
	}

	go d.runServer()

	return nil
}

func (d *WebSocketMessageDispatcher) runServer() error {
	if d.webServerInstance == nil {
		return fmt.Errorf("web server instance not initialized")
	}

	var err error
	if d.serverConfig.TlsCertFile != "" && d.serverConfig.TlsKeyFile != "" {
		err = d.webServerInstance.ListenAndServeTLS(d.serverConfig.TlsCertFile, d.serverConfig.TlsKeyFile)
	} else {
		err = d.webServerInstance.ListenAndServe()
	}

	return err
}

func (d *WebSocketMessageDispatcher) handleConnection(w http.ResponseWriter, r *http.Request) {
	if d.GetApp().IsClosing() || d.GetApp().IsClosed() {
		http.Error(w, "Application is closing or closed", http.StatusServiceUnavailable)
		return
	}

	d.sessionLock.Lock()
	defer d.sessionLock.Unlock()

	if len(d.sessions) >= int(d.wsConfig.MaxConnections) {
		http.Error(w, "Max connections reached", http.StatusBadGateway)
		return
	}

	d.upgraderLock.RLock()
	defer d.upgraderLock.RUnlock()

	conn, err := d.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		d.GetApp().GetDefaultLogger().Error("WebSocket upgrade failed", "error", err)
		return
	}
	conn.SetReadLimit(int64(d.wsConfig.MaxMessageSize))

	session := &WebSocketSession{
		SessionId:  d.AllocateSessionId(),
		Connection: conn,
		Authorized: false,
		sendQueue:  make(chan *public_protocol_extension.CSMsg, d.wsConfig.MaxWriteMessageCount),
	}

	session.runningContext, session.runningCancel = context.WithCancel(d.GetApp().GetAppContext())

	go d.handleSessionRead(session)
	go d.handleSessionWrite(session)
}

func (d *WebSocketMessageDispatcher) addSession(session *WebSocketSession) {
	d.sessionLock.Lock()
	defer d.sessionLock.Unlock()

	onNewSession := d.onNewSession.Load()
	if onNewSession != nil {
		err := onNewSession.(WebSocketCallbackOnNewSession)(d.CreateRpcContext(d), session)
		if err != nil {
			d.GetApp().GetDefaultLogger().Error("OnNewSession callback error", "error", err, "session_id", session.SessionId)
			d.Close(d.CreateRpcContext(d), session, websocket.CloseServiceRestart, "Service shutdown")

			if session.runningCancel != nil {
				session.runningCancel()
				session.runningCancel = nil
			}
			return
		}
	}

	d.sessions[session.SessionId] = session
}

func (d *WebSocketMessageDispatcher) removeSession(session *WebSocketSession) {
	d.sessionLock.Lock()
	defer d.sessionLock.Unlock()

	delete(d.sessions, session.SessionId)

	onRemoveSession := d.onRemoveSession.Load()
	if onRemoveSession != nil {
		onRemoveSession.(WebSocketCallbackOnRemoveSession)(d.CreateRpcContext(d), session)
	}
}

func (d *WebSocketMessageDispatcher) handleSessionRead(session *WebSocketSession) {
	defer d.Close(d.CreateRpcContext(d), session, websocket.CloseGoingAway, "Session closed by peer")

	for {
		_, messageData, err := session.Connection.ReadMessage()
		if err != nil {
			d.increaseErrorCounter(session)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				d.GetApp().GetDefaultLogger().Error("WebSocket unexpected close", "error", err, "session_id", session.SessionId)
			}
			break
		}

		d.GetApp().GetDefaultLogger().Debug("Websocket session read message", "session_id", session.SessionId, "message_size", len(messageData))

		msg := &public_protocol_extension.CSMsg{}
		err = proto.Unmarshal(messageData, msg)
		if err != nil {
			d.increaseErrorCounter(session)
			d.GetApp().GetDefaultLogger().Error("Failed to unmarshal message", "error", err, "session_id", session.SessionId)
			continue
		}

		session.resetErrorCounter()

		onNewMessage := d.onNewMessage.Load()
		if onNewMessage != nil {
			err = onNewMessage.(WebSocketCallbackOnNewMessage)(session, msg)
			if err != nil {
				d.GetApp().GetDefaultLogger().Error("OnNewMessage callback error", "error", err, "session_id", session.SessionId)
			}
		} else {
			d.GetApp().GetDefaultLogger().Debug("OnNewMessage without callback and will be dropped", "session_id", session.SessionId, "message", msg.String())
		}

		if err == nil {
			d.OnReceiveMessage(d, session.runningContext, &DispatcherRawMessage{
				Type:     d.GetInstanceIdent(),
				Instance: msg,
			}, session, d.AllocSequence())
		}
	}
}

func (d *WebSocketMessageDispatcher) handleSessionWrite(session *WebSocketSession) {
	d.addSession(session)
	defer d.removeSession(session)
	defer session.Connection.Close()

	authTimeoutContext, cancelFn := context.WithTimeout(session.runningContext, d.wsConfig.HandshakeTimeout.AsDuration())

	cleanTimeout := func() {
		if cancelFn != nil {
			cancelFn()
			cancelFn = nil
		}
	}
	defer cleanTimeout()

	for {
		// 已认证，不需要判定超时
		if session.Authorized || authTimeoutContext == nil {
			select {
			case <-session.runningContext.Done():
				d.Close(d.CreateRpcContext(d), session, websocket.CloseServiceRestart, "Service shutdown")
				break
			case writeMessage, ok := <-session.sendQueue:
				if !ok {
					d.Close(d.CreateRpcContext(d), session, websocket.CloseGoingAway, "Session closing")
					break
				}
				d.writeMessageToConnection(session, writeMessage)
			}
		} else {
			select {
			case <-authTimeoutContext.Done():
				if !session.Authorized {
					d.Close(d.CreateRpcContext(d), session, websocket.CloseNormalClosure, "Authentication timeout")
				}
				authTimeoutContext = nil
				cleanTimeout()

			case <-session.runningContext.Done():
				d.Close(d.CreateRpcContext(d), session, websocket.CloseServiceRestart, "Service shutdown")
				break
			case writeMessage, ok := <-session.sendQueue:
				if !ok {
					d.Close(d.CreateRpcContext(d), session, websocket.CloseGoingAway, "Session closing")
					break
				}
				d.writeMessageToConnection(session, writeMessage)
			}

			if session.Authorized {
				authTimeoutContext = nil
				cleanTimeout()
			}
		}
	}
}

func (d *WebSocketMessageDispatcher) WriteMessage(session *WebSocketSession, message *public_protocol_extension.CSMsg) error {
	if message == nil || session == nil {
		return fmt.Errorf("message or session is nil")
	}

	select {
	case session.sendQueue <- message:
		return nil
	default:
		d.increaseErrorCounter(session)
		d.GetApp().GetDefaultLogger().Error("Send queue full, dropping message", "session_id", session.SessionId)
		return fmt.Errorf("send queue full")
	}
}

func (d *WebSocketMessageDispatcher) writeMessageToConnection(session *WebSocketSession, message *public_protocol_extension.CSMsg) error {
	messageData, err := proto.Marshal(message)
	if err != nil {
		d.increaseErrorCounter(session)
		d.GetApp().GetDefaultLogger().Error("Failed to marshal message", "error", err, "session_id", session.SessionId)
		return err
	}

	err = session.Connection.WriteMessage(websocket.BinaryMessage, messageData)
	if err != nil {
		d.increaseErrorCounter(session)
		d.GetApp().GetDefaultLogger().Error("Failed to write message", "error", err, "session_id", session.SessionId)
		return err
	}

	session.resetErrorCounter()

	d.GetApp().GetDefaultLogger().Debug("Websocket session sent message", "session_id", session.SessionId, "message_size", len(messageData))
	return nil
}

func (d *WebSocketMessageDispatcher) Close(_ctx *RpcContext, session *WebSocketSession, closeCode int, text string) {
	if !session.sentCloseMessage {
		d.GetApp().GetDefaultLogger().Info("Closing WebSocket session", "session_id", session.SessionId, "reason", text)

		session.sentCloseMessage = true
		session.Connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, text))
		close(session.sendQueue)
	}

	if session.runningCancel != nil {
		session.runningCancel()
		session.runningCancel = nil
	}
}

func (d *WebSocketMessageDispatcher) increaseErrorCounter(session *WebSocketSession) {
	session.errorCounter++

	if session.errorCounter > 10 {
		d.Close(d.CreateRpcContext(d), session, websocket.ClosePolicyViolation, "Too many errors")
	}
}

func (s *WebSocketSession) resetErrorCounter() {
	s.errorCounter = 0
}

func (d *WebSocketMessageDispatcher) Reload() error {
	err := d.DispatcherBase.Reload()
	if err != nil {
		return err
	}

	// TODO: reload from config

	d.serverConfig = &private_protocol_config.WebserverCfg{
		Host:         "",
		Port:         7001,
		ReadTimeout:  durationpb.New(15 * time.Second),
		WriteTimeout: durationpb.New(15 * time.Second),
		IdleTimeout:  durationpb.New(60 * time.Second),
	}
	d.wsConfig = &private_protocol_config.WebsocketServerCfg{
		MaxConnections:       50000,
		ReadBufferSize:       4096,
		WriteBufferSize:      4096,
		HandshakeTimeout:     durationpb.New(10 * time.Second),
		PongWait:             durationpb.New(60 * time.Second),
		PingPeriod:           durationpb.New(54 * time.Second),
		WriteWait:            durationpb.New(10 * time.Second),
		MaxMessageSize:       2 * 1024 * 1024,
		MaxWriteMessageCount: 256,
		EnableCompression:    true,
		Path:                 "/ws/v1",
	}

	if d.IsActived() {
		return d.setupListen()
	}

	return nil
}

func (d *WebSocketMessageDispatcher) Stop() (bool, error) {
	if d.stopContext == nil {
		d.stopContext, d.stopCancel = context.WithTimeout(context.Background(), 5*time.Second)

		if d.webServerInstance != nil {
			d.webServerInstance.Shutdown(d.stopContext)
		}

		d.sessionLock.Lock()
		defer d.sessionLock.Unlock()

		for _, session := range d.sessions {
			if session.runningCancel != nil {
				session.runningCancel()
				session.runningCancel = nil
			}
		}
	}

	if d.stopCancel != nil {
		d.stopCancel()
		d.stopContext = nil
		d.stopCancel = nil
	}

	ret := true
	if len(d.sessions) > 0 {
		ret = false
	}

	return ret, nil
}

// This callback only will be call once after all module stopped
func (d *WebSocketMessageDispatcher) Cleanup() {
	if d.webServerInstance != nil {
		d.webServerInstance.Close()
	}

	if d.stopCancel != nil {
		d.stopCancel()
		d.stopContext = nil
		d.stopCancel = nil
	}
}

func (d *WebSocketMessageDispatcher) PickMessageTaskId(msg *DispatcherRawMessage) uint64 {
	// CS消息，不允许携带任务ID
	return 0
}

func (d *WebSocketMessageDispatcher) PickMessageRpcName(msg *DispatcherRawMessage) string {
	if msg == nil || msg.Type != d.GetInstanceIdent() {
		return ""
	}

	if csMsg, ok := msg.Instance.(*public_protocol_extension.CSMsg); ok {
		if csMsg.Head == nil {
			return ""
		}

		req := csMsg.Head.GetRpcRequest()
		if req != nil {
			return req.RpcName
		}

		rsp := csMsg.Head.GetRpcResponse()
		if rsp != nil {
			return rsp.RpcName
		}

		stream := csMsg.Head.GetRpcStream()
		if stream != nil {
			return stream.RpcName
		}
	}

	return ""
}

func (d *WebSocketMessageDispatcher) AllocateSessionId() uint64 {
	return d.sessionIdAllocator.Add(1)
}

func (d *WebSocketMessageDispatcher) SetOnNewSession(callback WebSocketCallbackOnNewSession) {
	if callback == nil {
		d.onNewSession.Store(nil)
		return
	}

	d.onNewSession.Store(callback)
}

func (d *WebSocketMessageDispatcher) SetOnRemoveSession(callback WebSocketCallbackOnRemoveSession) {
	if callback == nil {
		d.onRemoveSession.Store(nil)
		return
	}

	d.onRemoveSession.Store(callback)
}

func (d *WebSocketMessageDispatcher) SetOnNewMessage(callback WebSocketCallbackOnNewMessage) {
	if callback == nil {
		d.onNewMessage.Store(nil)
		return
	}

	d.onNewMessage.Store(callback)
}
