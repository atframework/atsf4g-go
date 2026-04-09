package atframework_component_dispatcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	libatapp "github.com/atframework/libatapp-go"

	private_protocol_config "github.com/atframework/atsf4g-go/component/protocol/private/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
)

func init() {
	var _ libatapp.AppModuleImpl = (*HttpClientDispatcher)(nil)
}

type httpQueryWithDispatcherContext struct {
	originRequest  *http.Request
	originResponse *http.Response

	method string
	url    string
	body   io.Reader

	ctx    context.Context
	cancel context.CancelFunc
}

type HttpQuery = httpQueryWithDispatcherContext

func (q *httpQueryWithDispatcherContext) GetContext() context.Context {
	if q == nil {
		return nil
	}

	return q.ctx
}

func (q *httpQueryWithDispatcherContext) GetHttpRequest() *http.Request {
	if q == nil {
		return nil
	}

	return q.originRequest
}

func (q *httpQueryWithDispatcherContext) GetHttpResponse() *http.Response {
	if q == nil {
		return nil
	}

	return q.originResponse
}

func (q *httpQueryWithDispatcherContext) Cancel() bool {
	if q == nil {
		return false
	}

	if q.cancel == nil {
		return false
	}

	cancelFn := q.cancel
	q.cancel = nil
	cancelFn()
	return true
}

type HttpClientDispatcher struct {
	DispatcherBase

	initialized atomic.Bool

	mu                  sync.Mutex
	clientConfigurePath string

	clientConfig    *private_protocol_config.Readonly_HttpClientCfg
	client          *http.Client
	runningRequests map[uint64]*HttpQuery
}

func CreateHttpClientDispatcher(owner libatapp.AppImpl, clientConfigurePath string) *HttpClientDispatcher {
	// 使用时间戳作为初始值, 避免与重启前的值冲突
	ret := &HttpClientDispatcher{
		DispatcherBase:      CreateDispatcherBase(owner),
		initialized:         atomic.Bool{},
		clientConfigurePath: clientConfigurePath,
	}

	return ret
}

func (d *HttpClientDispatcher) Name() string { return "HttpClientDispatcher" }

func (d *HttpClientDispatcher) Init(initCtx context.Context) error {
	if d == nil {
		return fmt.Errorf("HttpClientDispatcher is nil")
	}

	err := d.DispatcherBase.Init(initCtx)
	if err != nil {
		return err
	}

	d.initialized.Store(true)
	return nil
}

func (d *HttpClientDispatcher) Reload() error {
	if d == nil {
		return fmt.Errorf("HttpClientDispatcher is nil")
	}

	err := d.DispatcherBase.Reload()
	if err != nil {
		return err
	}

	clientConfig := &private_protocol_config.HttpClientCfg{}

	loadErr := d.GetApp().LoadConfigByPath(clientConfig, d.clientConfigurePath,
		strings.ToUpper(strings.ReplaceAll(d.clientConfigurePath, ".", "_")), nil, "")
	if loadErr != nil {
		d.GetLogger().LogError("Failed to load client config", "error", loadErr)
		return loadErr
	}

	d.clientConfig = clientConfig.ToReadonly()
	return nil
}

func (d *HttpClientDispatcher) Cleanup() {
	if d == nil {
		return
	}

	if d.client != nil {
		// No explicit cleanup needed for http.Client
		d.mu.Lock()

		runningRequests := d.runningRequests
		d.runningRequests = make(map[uint64]*HttpQuery)

		d.mu.Unlock()

		for _, query := range runningRequests {
			query.Cancel()
		}
	}

	d.initialized.Store(false)
}

func (d *HttpClientDispatcher) CreateDispatcherAwaitOptions() *DispatcherAwaitOptions {
	if d == nil {
		return nil
	}

	timeout := d.clientConfig.GetTimeout().AsDuration()
	if timeout.Milliseconds() <= 1 {
		timeout = time.Second * 15
	}

	return &DispatcherAwaitOptions{
		Type:     d.GetInstanceIdent(),
		Sequence: d.AllocSequence(),
		Timeout:  timeout,
	}
}

func (d *HttpClientDispatcher) insertRequestAndStart(query *HttpQuery, awaitOption *DispatcherAwaitOptions) RpcResult {
	if d == nil || query == nil || awaitOption == nil {
		return CreateRpcResultError(fmt.Errorf("invalid parameter"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	d.mu.Lock()

	if d.client == nil {
		d.client = &http.Client{
			Timeout: d.clientConfig.GetTimeout().AsDuration(),
		}
	}

	if d.runningRequests == nil {
		d.runningRequests = make(map[uint64]*HttpQuery)
	}

	d.runningRequests[awaitOption.Sequence] = query

	client := d.client
	app := d.GetApp()
	d.mu.Unlock()

	err := app.PushAction(func(action *libatapp.AppActionData) error {
		var err error
		query.originResponse, err = client.Do(query.originRequest)
		if err != nil {
			d.GetLogger().LogError("HTTP request failed", "error", err, "method", query.method, "url", query.url)
		}
		return err
	}, nil, query)
	if err != nil {
		d.GetLogger().LogError("Failed to push HTTP request action", "error", err)
		return CreateRpcResultError(fmt.Errorf("failed to start HTTP request"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	return CreateRpcResultOk()
}

func (d *HttpClientDispatcher) removeRequestAndCleanup(query *HttpQuery, awaitOption *DispatcherAwaitOptions) RpcResult {
	if d == nil || query == nil || awaitOption == nil {
		return CreateRpcResultError(fmt.Errorf("invalid parameter"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	d.mu.Lock()

	delete(d.runningRequests, awaitOption.Sequence)
	query.Cancel()

	d.mu.Unlock()

	return CreateRpcResultOk()
}

func (d *HttpClientDispatcher) CreateQuery(ctx AwaitableContext, method string, url string, body io.Reader) (*HttpQuery, error) {
	if d == nil {
		return nil, fmt.Errorf("HttpClientDispatcher is nil")
	}

	newCtx, cancelFn := context.WithCancel(ctx.GetContext())
	req, err := http.NewRequestWithContext(newCtx, method, url, body)
	if err != nil {
		cancelFn()
		return nil, err
	}

	return &HttpQuery{
		originRequest:  req,
		originResponse: nil,
		method:         req.Method,
		url:            req.URL.String(),
		body:           body,
		ctx:            newCtx,
		cancel:         cancelFn,
	}, nil
}

func (d *HttpClientDispatcher) StartQuery(ctx AwaitableContext, query *HttpQuery) RpcResult {
	if d == nil || query == nil {
		return CreateRpcResultError(fmt.Errorf("HTTP query can not be nil"), public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if d.initialized.Load() == false {
		return CreateRpcResultError(fmt.Errorf("HttpClientDispatcher is not initialized"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	currentAction := ctx.GetAction()
	if lu.IsNil(currentAction) {
		ctx.LogError("not in context action")
		return CreateRpcResultError(fmt.Errorf("action not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}
	if currentAction.GetRpcContext() == nil || lu.IsNil(currentAction.GetRpcContext().GetContext()) {
		ctx.LogError("not found context")
		return CreateRpcResultError(fmt.Errorf("context not found"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	awaitOption := d.CreateDispatcherAwaitOptions()

	resumeData, retResult := YieldTaskAction(ctx, currentAction, awaitOption, &YieldTaskHookSet{
		PreYield: func(ctx RpcContext) RpcResult {
			return d.insertRequestAndStart(query, awaitOption)
		},
		PostYield: func(ctx RpcContext, resume *DispatcherResumeData, result RpcResult) RpcResult {
			return d.removeRequestAndCleanup(query, awaitOption)
		},
	})
	if retResult.IsError() {
		return retResult
	}
	if resumeData.Result.IsError() {
		retResult = resumeData.Result
	}

	return retResult
}
