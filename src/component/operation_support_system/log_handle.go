package atframework_component_operation_support_system

import (
	"context"
	"sync/atomic"
	"time"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	log "github.com/atframework/atframe-utils-go/log"
	config "github.com/atframework/atsf4g-go/component-config"
	logical_time "github.com/atframework/atsf4g-go/component-logical_time"
	private_protocol_log "github.com/atframework/atsf4g-go/component-protocol-private/log/protocol/log"
	libatapp "github.com/atframework/libatapp-go"
	"google.golang.org/protobuf/encoding/protojson"
)

func SendOssLog(app libatapp.AppImpl, ossLog *private_protocol_log.OperationSupportSystemLog) {
	libatapp.AtappGetModule[*OperationSupportSystem](app).sendOssLog(ossLog)
}

func SendMonLog(app libatapp.AppImpl, monLog *private_protocol_log.MonitorLog) {
	libatapp.AtappGetModule[*OperationSupportSystem](app).sendMonLog(monLog)
}

////////////////////////////////////////////////////////////////////////

type OperationSupportSystem struct {
	libatapp.AppModuleBase
	ossLogWriter *log.LogBufferedRotatingWriter
	monLogWriter *log.LogBufferedRotatingWriter

	startTimestamp uint64
}

func init() {
	var _ libatapp.AppModuleImpl = (*OperationSupportSystem)(nil)
}

func (m *OperationSupportSystem) Init(parent context.Context) error {
	return m.initLogWritter()
}

func (m *OperationSupportSystem) Name() string {
	return "OperationSupportSystem"
}

func CreateOperationSupportSystem(app libatapp.AppImpl) *OperationSupportSystem {
	ret := &OperationSupportSystem{
		AppModuleBase: libatapp.CreateAppModuleBase(app),
	}
	return ret
}

func (m *OperationSupportSystem) Ready() {
	m.startTimestamp = uint64(logical_time.GetSysNow().Unix())
	log := private_protocol_log.MonitorLog{}
	flow := log.MutableServerOperationFlow()
	flow.OperationType = private_protocol_log.MONServerOperationFlow_EN_SERVER_OPERATION_TYPE_START
	flow.StartTimestamp = m.startTimestamp
	flow.ServerVersion = m.GetApp().GetAppVersion()
	m.sendMonLog(&log)
}

func (m *OperationSupportSystem) Reload() error {
	log := private_protocol_log.MonitorLog{}
	flow := log.MutableServerOperationFlow()
	flow.OperationType = private_protocol_log.MONServerOperationFlow_EN_SERVER_OPERATION_TYPE_RELOAD
	flow.StartTimestamp = m.startTimestamp
	flow.ServerVersion = m.GetApp().GetAppVersion()
	m.sendMonLog(&log)
	return nil
}

func (m *OperationSupportSystem) Stop() (bool, error) {
	log := private_protocol_log.MonitorLog{}
	flow := log.MutableServerOperationFlow()
	flow.OperationType = private_protocol_log.MONServerOperationFlow_EN_SERVER_OPERATION_TYPE_STOP
	flow.StartTimestamp = m.startTimestamp
	flow.ServerVersion = m.GetApp().GetAppVersion()
	m.sendMonLog(&log)
	m.monLogWriter.Flush()
	m.ossLogWriter.Flush()
	return true, nil
}

//////////////////////////////////////////////////////////////////////////////////

func (m *OperationSupportSystem) initLogWritter() error {
	cfg := config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetOperationSupportSystem()
	if cfg.GetOssCfg().GetEnable() {
		writer, err := log.NewLogBufferedRotatingWriter(m.GetApp(), cfg.GetOssCfg().GetFile(), cfg.GetOssCfg().GetWritingAlias(),
			cfg.GetOssCfg().GetRotate().GetSize(), cfg.GetOssCfg().GetRotate().GetNumber(), cfg.GetOssCfg().GetFlushInterval().AsDuration(), 0)
		if err != nil {
			return err
		}
		m.ossLogWriter = writer
	}
	if cfg.GetMonCfg().GetEnable() {
		writer, err := log.NewLogBufferedRotatingWriter(m.GetApp(), cfg.GetMonCfg().GetFile(), cfg.GetMonCfg().GetWritingAlias(),
			cfg.GetMonCfg().GetRotate().GetSize(), cfg.GetMonCfg().GetRotate().GetNumber(), cfg.GetMonCfg().GetFlushInterval().AsDuration(), 0)
		if err != nil {
			return err
		}
		m.monLogWriter = writer
	}
	return nil
}

type timestampCacheEntry struct {
	second int64
	prefix string
}

var appendTimestampCache atomic.Value

func init() {
	appendTimestampCache.Store(timestampCacheEntry{second: -1})
}

func (m *OperationSupportSystem) sendOssLog(ossLog *private_protocol_log.OperationSupportSystemLog) {
	if m == nil || m.ossLogWriter == nil || ossLog.GetDetailOneofName() == "" {
		return
	}

	entry := appendTimestampCache.Load().(timestampCacheEntry)
	now := logical_time.GetSysNow()
	second := now.Unix()
	if entry.second != second {
		prefix := now.Format(time.DateTime)
		entry = timestampCacheEntry{second: second, prefix: prefix}
		appendTimestampCache.Store(entry)
	}

	// 处理头
	ossLog.LogType = ossLog.GetDetailOneofName()
	ossLog.MutableBasic().AppId = m.GetApp().GetConfig().ConfigPb.GetName()
	// 写入
	b, _ := protojson.MarshalOptions{Multiline: false, UseEnumNumbers: true, UseProtoNames: true}.MarshalAppend(
		lu.StringtoBytes(entry.prefix), ossLog)
	m.ossLogWriter.Write(b)
	m.ossLogWriter.Flush()
}

func (m *OperationSupportSystem) sendMonLog(monLog *private_protocol_log.MonitorLog) {
	if m == nil || m.monLogWriter == nil || monLog.GetDetailOneofName() == "" {
		return
	}

	entry := appendTimestampCache.Load().(timestampCacheEntry)
	now := logical_time.GetSysNow()
	second := now.Unix()
	if entry.second != second {
		prefix := now.Format(time.DateTime)
		entry = timestampCacheEntry{second: second, prefix: prefix}
		appendTimestampCache.Store(entry)
	}

	// 处理头
	monLog.LogType = monLog.GetDetailOneofName()
	monLog.MutableBasic().AppId = m.GetApp().GetConfig().ConfigPb.GetName()
	// 写入
	b, _ := protojson.MarshalOptions{Multiline: false, UseEnumNumbers: true, UseProtoNames: true}.MarshalAppend(
		lu.StringtoBytes(entry.prefix), monLog)
	m.monLogWriter.Write(b)
	m.monLogWriter.Flush()
}
