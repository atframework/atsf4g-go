package atframework_component_operation_support_system

import (
	"context"
	"reflect"
	"runtime"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
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

var operationSupportSystemReflectType reflect.Type

type OperationSupportSystem struct {
	libatapp.AppModuleBase
	ossLogWriter *libatapp.LogBufferedRotatingWriter
	monLogWriter *libatapp.LogBufferedRotatingWriter

	startTimestamp uint64
}

func init() {
	operationSupportSystemReflectType = lu.GetStaticReflectType[OperationSupportSystem]()
	var _ libatapp.AppModuleImpl = (*OperationSupportSystem)(nil)
}

func (m *OperationSupportSystem) GetReflectType() reflect.Type {
	return operationSupportSystemReflectType
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
	return true, nil
}

//////////////////////////////////////////////////////////////////////////////////

func (m *OperationSupportSystem) initLogWritter() error {
	cfg := config.GetConfigManager().GetCurrentConfigGroup().GetServerConfig().GetOperationSupportSystem()
	if cfg.GetOssCfg().GetEnable() {
		writer, err := libatapp.NewLogBufferedRotatingWriter(m.GetApp(), cfg.GetOssCfg().GetFile(), cfg.GetOssCfg().GetWritingAlias(),
			cfg.GetOssCfg().GetRotate().GetSize(), cfg.GetOssCfg().GetRotate().GetNumber(), cfg.GetOssCfg().GetFlushInterval().AsDuration())
		if err != nil {
			return err
		}
		runtime.SetFinalizer(writer, func(writer *libatapp.LogBufferedRotatingWriter) {
			writer.Close()
		})
		m.ossLogWriter = writer
	}
	if cfg.GetMonCfg().GetEnable() {
		writer, err := libatapp.NewLogBufferedRotatingWriter(m.GetApp(), cfg.GetMonCfg().GetFile(), cfg.GetMonCfg().GetWritingAlias(),
			cfg.GetMonCfg().GetRotate().GetSize(), cfg.GetMonCfg().GetRotate().GetNumber(), cfg.GetMonCfg().GetFlushInterval().AsDuration())
		if err != nil {
			return err
		}
		runtime.SetFinalizer(writer, func(writer *libatapp.LogBufferedRotatingWriter) {
			writer.Close()
		})
		m.monLogWriter = writer
	}
	return nil
}

func (m *OperationSupportSystem) sendOssLog(ossLog *private_protocol_log.OperationSupportSystemLog) {
	if m == nil || m.ossLogWriter == nil || ossLog.GetDetailOneofCase() == 0 {
		return
	}
	// 处理头
	ossLog.MutableBasic().Timestamp = uint64(logical_time.GetSysNow().Unix())
	ossLog.MutableBasic().LogType = uint32(ossLog.GetDetailOneofCase())
	ossLog.MutableBasic().AppId = m.GetApp().GetConfig().ConfigPb.GetName()
	// 写入
	m.ossLogWriter.Write(lu.StringtoBytes(protojson.MarshalOptions{Multiline: false, UseEnumNumbers: true}.Format(ossLog)))
	m.ossLogWriter.Flush()
}

func (m *OperationSupportSystem) sendMonLog(monLog *private_protocol_log.MonitorLog) {
	if m == nil || m.monLogWriter == nil || monLog.GetDetailOneofCase() == 0 {
		return
	}
	// 处理头
	monLog.MutableBasic().Timestamp = uint64(logical_time.GetSysNow().Unix())
	monLog.MutableBasic().LogType = uint32(monLog.GetDetailOneofCase())
	monLog.MutableBasic().AppId = m.GetApp().GetConfig().ConfigPb.GetName()
	// 写入
	m.monLogWriter.Write(lu.StringtoBytes(protojson.MarshalOptions{Multiline: false, UseEnumNumbers: true}.Format(monLog)))
	m.monLogWriter.Flush()
}
