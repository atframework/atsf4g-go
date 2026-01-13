package atframework_component_dispatcher

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	libatapp "github.com/atframework/libatapp-go"

	private_protocol_config "github.com/atframework/atsf4g-go/component-protocol-private/config/protocol/config"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_extension "github.com/atframework/atsf4g-go/component-protocol-public/extension/protocol/extension"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

var websocketMessageDispatcherReflectType reflect.Type

func init() {
	var _ libatapp.AppModuleImpl = (*WebSocketMessageDispatcher)(nil)
	websocketMessageDispatcherReflectType = lu.GetStaticReflectType[WebSocketMessageDispatcher]()
}

type closeParam struct {
	closeCode int
	text      string
}

type WebSocketSession struct {
	SessionId uint64

	Connection *websocket.Conn

	Authorized atomic.Bool

	runningContext context.Context
	runningCancel  context.CancelFunc

	sendQueue chan *public_protocol_extension.CSMsg

	sendQueueClose   chan closeParam
	sentCloseMessage atomic.Bool

	errorCounter atomic.Int32

	PrivateData interface{}
}

type (
	WebSocketCallbackOnNewSession    = func(ctx RpcContext, session *WebSocketSession) error
	WebSocketCallbackOnRemoveSession = func(ctx RpcContext, session *WebSocketSession)
	WebSocketCallbackOnNewMessage    = func(session *WebSocketSession, message *public_protocol_extension.CSMsg) error
)

type WebSocketMessageDispatcher struct {
	DispatcherBase

	webServerConfigurePath       string
	webSocketServerConfigurePath string

	serverConfig *private_protocol_config.Readonly_WebserverCfg
	wsConfig     *private_protocol_config.Readonly_WebsocketServerCfg
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

func CreateCSMessageWebsocketDispatcher(owner libatapp.AppImpl, webServerConfigurePath string, webSocketServerConfigurePath string) *WebSocketMessageDispatcher {
	// 使用时间戳作为初始值, 避免与重启前的值冲突
	ret := &WebSocketMessageDispatcher{
		DispatcherBase:               CreateDispatcherBase(owner),
		webServerConfigurePath:       webServerConfigurePath,
		webSocketServerConfigurePath: webSocketServerConfigurePath,

		sessions:           make(map[uint64]*WebSocketSession),
		sessionIdAllocator: atomic.Uint64{},
	}
	ret.DispatcherBase.impl = ret

	ret.sessionIdAllocator.Store(uint64(time.Since(time.Unix(int64(private_protocol_pbdesc.EnSystemLimit_EN_SL_TIMESTAMP_FOR_ID_ALLOCATOR_OFFSET), 0)).Nanoseconds()))
	return ret
}

func (d *WebSocketMessageDispatcher) Name() string { return "WebSocketMessageDispatcher" }

func (m *WebSocketMessageDispatcher) GetReflectType() reflect.Type {
	return websocketMessageDispatcherReflectType
}

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
		HandshakeTimeout: d.wsConfig.GetHandshakeTimeout().AsDuration(),
		ReadBufferSize:   int(d.wsConfig.GetReadBufferSize()),
		WriteBufferSize:  int(d.wsConfig.GetWriteBufferSize()),
		Subprotocols:     d.wsConfig.GetSubProtocols(),
		CheckOrigin: func(r *http.Request) bool {
			// Configure origin checking for production
			return true
		},
		EnableCompression: d.wsConfig.GetEnableCompression(),
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
			if !strings.HasPrefix(r.URL.Path, d.wsConfig.GetPath()) {
				http.NotFound(w, r)
				return
			}

			d.handleConnection(w, r)
		})
	}

	if d.webServerInstance != nil && d.webServerAddress != d.serverConfig.GetHost()+":"+fmt.Sprintf("%d", d.serverConfig.GetPort()) {
		d.webServerInstance.Close()
		d.webServerInstance = nil
	}

	if d.webServerInstance == nil {
		d.webServerAddress = d.serverConfig.GetHost() + ":" + fmt.Sprintf("%d", d.serverConfig.GetPort())
		d.webServerInstance = &http.Server{
			Addr:         d.webServerAddress,
			Handler:      d.webServerHandle,
			ReadTimeout:  d.serverConfig.GetReadTimeout().AsDuration(),
			WriteTimeout: d.serverConfig.GetWriteTimeout().AsDuration(),
			IdleTimeout:  d.serverConfig.GetIdleTimeout().AsDuration(),
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
	if d.serverConfig.GetTlsCertFile() != "" && d.serverConfig.GetTlsKeyFile() != "" {
		err = d.webServerInstance.ListenAndServeTLS(d.serverConfig.GetTlsCertFile(), d.serverConfig.GetTlsKeyFile())
	} else {
		err = d.webServerInstance.ListenAndServe()
	}

	if err != nil {
		d.GetApp().GetDefaultLogger().LogError("Web server error", "error", err)
		d.GetApp().Stop()
	}

	return nil
}

func (d *WebSocketMessageDispatcher) handleConnection(w http.ResponseWriter, r *http.Request) {
	if d.GetApp().IsClosing() || d.GetApp().IsClosed() {
		http.Error(w, "Application is closing or closed", http.StatusServiceUnavailable)
		return
	}

	d.sessionLock.Lock()
	defer d.sessionLock.Unlock()

	if len(d.sessions) >= int(d.wsConfig.GetMaxConnections()) {
		http.Error(w, "Max connections reached", http.StatusBadGateway)
		return
	}

	d.upgraderLock.RLock()
	defer d.upgraderLock.RUnlock()

	conn, err := d.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Could not open websocket connection", http.StatusBadRequest)
		d.GetApp().GetDefaultLogger().LogError("WebSocket upgrade failed", "error", err)
		return
	}
	conn.SetReadLimit(int64(d.wsConfig.GetMaxMessageSize()))

	session := &WebSocketSession{
		SessionId:      d.AllocateSessionId(),
		Connection:     conn,
		sendQueue:      make(chan *public_protocol_extension.CSMsg, d.wsConfig.GetMaxWriteMessageCount()),
		sendQueueClose: make(chan closeParam, 1),
	}

	session.runningContext, session.runningCancel = context.WithCancel(d.GetApp().GetAppContext())

	go d.handleSessionIO(session)
}

func (d *WebSocketMessageDispatcher) addSession(session *WebSocketSession) {
	d.sessionLock.Lock()
	defer d.sessionLock.Unlock()

	onNewSession := d.onNewSession.Load()
	if onNewSession != nil {
		err := onNewSession.(WebSocketCallbackOnNewSession)(d.CreateRpcContext(), session)
		if err != nil {
			d.GetApp().GetDefaultLogger().LogError("OnNewSession callback error", "error", err, "session_id", session.SessionId)
			d.AsyncClose(d.CreateRpcContext(), session, websocket.CloseServiceRestart, "Service shutdown")
			return
		}
	}

	d.sessions[session.SessionId] = session

	d.GetApp().GetDefaultLogger().LogInfo("New WebSocket session added", "client", session.Connection.RemoteAddr().String(), "session_id", session.SessionId)
}

func (d *WebSocketMessageDispatcher) removeSession(session *WebSocketSession) {
	d.sessionLock.Lock()
	defer d.sessionLock.Unlock()

	delete(d.sessions, session.SessionId)

	onRemoveSession := d.onRemoveSession.Load()
	if onRemoveSession != nil {
		ctx := d.CreateRpcContext()
		ctx.SetContext(context.Background())
		onRemoveSession.(WebSocketCallbackOnRemoveSession)(ctx, session)
	}

	d.GetApp().GetDefaultLogger().LogInfo("WebSocket session removed", "client", session.Connection.RemoteAddr().String(), "session_id", session.SessionId)
}

func (d *WebSocketMessageDispatcher) handleSessionRead(session *WebSocketSession) {
	defer d.AsyncClose(d.CreateRpcContext(), session, websocket.CloseGoingAway, "Session closed by peer")

	for {
		_, messageData, err := session.Connection.ReadMessage()
		if err != nil {
			d.increaseErrorCounter(session)
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				d.GetApp().GetDefaultLogger().LogError("WebSocket unexpected close", "error", err, "session_id", session.SessionId)
			}
			break
		}

		msg := &public_protocol_extension.CSMsg{}
		err = proto.Unmarshal(messageData, msg)
		if err != nil {
			d.increaseErrorCounter(session)
			d.GetApp().GetDefaultLogger().LogError("Failed to unmarshal message", "error", err, "session_id", session.SessionId)
			continue
		}

		session.resetErrorCounter()

		onNewMessage := d.onNewMessage.Load()
		if onNewMessage != nil {
			err = onNewMessage.(WebSocketCallbackOnNewMessage)(session, msg)
			if err != nil {
				d.GetApp().GetDefaultLogger().LogError("OnNewMessage callback error", "error", err, "session_id", session.SessionId)
			}
		}

		d.GetApp().GetDefaultLogger().LogDebug("handleSessionRead", "session_id", session.SessionId, "err", err, "rpc_name", msg.Head.GetRpcRequest().GetRpcName())

		if err == nil {
			d.OnReceiveMessage(session.runningContext, &DispatcherRawMessage{
				Type:     d.GetInstanceIdent(),
				Instance: msg,
			}, session, d.AllocSequence())
		}
	}
}

func (d *WebSocketMessageDispatcher) handleSessionIO(session *WebSocketSession) {
	d.addSession(session)
	// Read
	go d.handleSessionRead(session)
	// Write
	defer d.removeSession(session)
	defer session.Connection.Close()

	authTimeoutContext, cancelFn := context.WithTimeout(session.runningContext, d.wsConfig.GetHandshakeTimeout().AsDuration())

	cleanTimeout := func() {
		if cancelFn != nil {
			cancelFn()
			cancelFn = nil
		}
	}
	defer cleanTimeout()

	for {
		// 已认证，不需要判定超时
		if session.Authorized.Load() || authTimeoutContext == nil {
			select {
			case <-session.runningContext.Done():
				d.closeSession(d.CreateRpcContext(), session, websocket.CloseServiceRestart, "Service shutdown")
				break
			case writeMessage, ok := <-session.sendQueue:
				if !ok {
					d.closeSession(d.CreateRpcContext(), session, websocket.CloseGoingAway, "Session closing")
					break
				}
				d.writeMessageToConnection(session, writeMessage)
			case closeParam, ok := <-session.sendQueueClose:
				if !ok {
					d.closeSession(d.CreateRpcContext(), session, websocket.CloseGoingAway, "Session closing")
					break
				}
				d.closeSession(d.CreateRpcContext(), session, closeParam.closeCode, closeParam.text)
				break
			}
		} else {
			select {
			case <-authTimeoutContext.Done():
				if !session.Authorized.Load() {
					d.closeSession(d.CreateRpcContext(), session, websocket.CloseNormalClosure, "Authentication timeout")
				}
				authTimeoutContext = nil
				cleanTimeout()

			case <-session.runningContext.Done():
				d.closeSession(d.CreateRpcContext(), session, websocket.CloseServiceRestart, "Service shutdown")
				break

			case writeMessage, ok := <-session.sendQueue:
				if !ok {
					d.closeSession(d.CreateRpcContext(), session, websocket.CloseGoingAway, "Session closing")
					break
				}
				d.writeMessageToConnection(session, writeMessage)
			case closeParam, ok := <-session.sendQueueClose:
				if !ok {
					d.closeSession(d.CreateRpcContext(), session, websocket.CloseGoingAway, "Session closing")
					break
				}
				d.closeSession(d.CreateRpcContext(), session, closeParam.closeCode, closeParam.text)
				break
			}

			if session.Authorized.Load() {
				authTimeoutContext = nil
				cleanTimeout()
			}
		}

		if session.sentCloseMessage.Load() {
			break
		}
	}
}

func (d *WebSocketMessageDispatcher) WriteMessage(session *WebSocketSession, message *public_protocol_extension.CSMsg) error {
	if message == nil || session == nil {
		return fmt.Errorf("message or session is nil")
	}

	if d == nil {
		return fmt.Errorf("dispatcher is nil")
	}

	if session.sentCloseMessage.Load() {
		// session is closing, do not send more messages
		d.GetApp().GetDefaultLogger().LogWarn("Attempted to send message on closing session", "session_id", session.SessionId)
		return fmt.Errorf("session is closing")
	}

	select {
	case session.sendQueue <- message:
		return nil
	default:
		d.increaseErrorCounter(session)
		d.GetApp().GetDefaultLogger().LogError("Send queue full, dropping message", "session_id", session.SessionId)
		return fmt.Errorf("send queue full")
	}
}

func (d *WebSocketMessageDispatcher) writeMessageToConnection(session *WebSocketSession, message *public_protocol_extension.CSMsg) error {
	messageData, err := proto.Marshal(message)
	if err != nil {
		d.increaseErrorCounter(session)
		d.GetApp().GetDefaultLogger().LogError("Failed to marshal message", "error", err, "session_id", session.SessionId)
		return err
	}

	err = session.Connection.WriteMessage(websocket.BinaryMessage, messageData)
	if err != nil {
		d.increaseErrorCounter(session)
		d.GetApp().GetDefaultLogger().LogError("Failed to write message", "error", err, "session_id", session.SessionId)
		return err
	}

	session.resetErrorCounter()
	return nil
}

// 会由多个线程调用 需要线程安全
func (d *WebSocketMessageDispatcher) AsyncClose(_ctx RpcContext, session *WebSocketSession, closeCode int, text string) {
	if session.sentCloseMessage.CompareAndSwap(false, true) {
		d.GetApp().GetDefaultLogger().LogInfo("AsyncClose WebSocket session", "session_id", session.SessionId, "reason", text)

		// close(session.sendQueue) // 不能关闭channel,可能有并发写入,由channel通知关闭
		session.sendQueueClose <- closeParam{
			closeCode: closeCode,
			text:      text,
		}
	}
}

// 需要保证只在一个线程调用
func (d *WebSocketMessageDispatcher) closeSession(_ctx RpcContext, session *WebSocketSession, closeCode int, text string) {
	session.sentCloseMessage.Store(true)

	d.GetApp().GetDefaultLogger().LogInfo("Close WebSocket session", "session_id", session.SessionId, "reason", text)
	session.Connection.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, text))

	if session.runningCancel != nil {
		fn := session.runningCancel
		session.runningCancel = nil
		fn()
	}
}

func (d *WebSocketMessageDispatcher) increaseErrorCounter(session *WebSocketSession) {
	if session.errorCounter.Add(1) > 10 {
		d.AsyncClose(d.CreateRpcContext(), session, websocket.ClosePolicyViolation, "Too many errors")
	}
}

func (s *WebSocketSession) resetErrorCounter() {
	s.errorCounter.Store(0)
}

func (d *WebSocketMessageDispatcher) Reload() error {
	err := d.DispatcherBase.Reload()
	if err != nil {
		return err
	}

	// reload from config
	serverConfig := &private_protocol_config.WebserverCfg{}
	wsConfig := &private_protocol_config.WebsocketServerCfg{}

	loadErr := d.GetApp().LoadConfigByPath(serverConfig, d.webServerConfigurePath,
		strings.ToUpper(strings.ReplaceAll(d.webServerConfigurePath, ".", "_")), nil, "")
	if loadErr != nil {
		d.GetLogger().LogError("Failed to load web server config", "error", loadErr)
		return loadErr
	}

	loadErr = d.GetApp().LoadConfigByPath(wsConfig, d.webSocketServerConfigurePath,
		strings.ToUpper(strings.ReplaceAll(d.webSocketServerConfigurePath, ".", "_")), nil, "")
	if loadErr != nil {
		d.GetLogger().LogError("Failed to load websocket server config", "error", loadErr)
		return loadErr
	}

	if serverConfig.GetPort() <= 0 || serverConfig.GetPort() > 65535 {
		err = fmt.Errorf("invalid web server port: %d", serverConfig.GetPort())
		d.GetLogger().LogError("invalid web server port: ", "Port", serverConfig.GetPort())
		return err
	}

	d.serverConfig = serverConfig.ToReadonly()
	d.wsConfig = wsConfig.ToReadonly()

	if d.IsActived() {
		return d.setupListen()
	}

	return err
}

func (d *WebSocketMessageDispatcher) Stop() (bool, error) {
	if d.stopContext == nil {
		d.stopContext, d.stopCancel = context.WithTimeout(context.Background(), 5*time.Second)

		if d.webServerInstance != nil {
			d.webServerInstance.Shutdown(d.stopContext)
		}

		d.sessionLock.Lock()
		for _, session := range d.sessions {
			d.AsyncClose(d.CreateRpcContext(), session, websocket.CloseServiceRestart, "Server is stopping")
		}
		d.sessionLock.Unlock()
	}

	if d.stopCancel != nil {
		d.stopCancel()
		d.stopContext = nil
		d.stopCancel = nil
	}

	ret := true
	d.sessionLock.Lock()
	if len(d.sessions) > 0 {
		ret = false
	}
	d.sessionLock.Unlock()

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
