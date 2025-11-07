package atframework_component_dispatcher

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"unsafe"

	"sync/atomic"

	private_protocol_config "github.com/atframework/atsf4g-go/component-protocol-private/config/protocol/config"
	libatapp "github.com/atframework/libatapp-go"
	"github.com/redis/go-redis/v9"
)

type RedisLog struct {
	app libatapp.AppImpl
}

func (l *RedisLog) Printf(ctx context.Context, format string, v ...interface{}) {
	libatapp.LogInner(l.app.GetLogger(1), libatapp.GetCaller(1), ctx, slog.LevelInfo, fmt.Sprintf(format, v...))
}

type RedisMessageDispatcher struct {
	DispatcherBase
	log RedisLog

	redisCfg      private_protocol_config.LogicRedisCfg
	redisInstance *redis.Client
	sequence      atomic.Uint64
}

func CreateRedisMessageDispatcher(owner libatapp.AppImpl) *RedisMessageDispatcher {
	// 使用时间戳作为初始值, 避免与重启前的值冲突
	ret := &RedisMessageDispatcher{
		DispatcherBase: CreateDispatcherBase(owner),
		log:            RedisLog{app: owner},
	}
	return ret
}

func (d *RedisMessageDispatcher) Name() string { return "RedisMessageDispatcher" }

func (d *RedisMessageDispatcher) Init(initCtx context.Context) error {
	err := d.DispatcherBase.Init(initCtx)
	if err != nil {
		return err
	}

	loadErr := libatapp.LoadConfigFromOriginData(d.GetApp().GetConfig().ConfigOriginData, "lobbysvr.redis", &d.redisCfg, d.GetLogger())
	if loadErr != nil {
		d.GetLogger().Warn("Failed to load websocket server config", "error", loadErr)
	}

	if d.redisCfg.Addr == "" {
		d.DispatcherBase.GetLogger().Error("redis config error empty")
		return fmt.Errorf("redis config error empty")
	}

	redis.SetLogger(&d.log)
	d.redisInstance = redis.NewClient(&redis.Options{
		Addr:     d.redisCfg.Addr,
		Password: d.redisCfg.Password,
		PoolSize: int(d.redisCfg.PoolSize),
	})

	if d.redisInstance == nil {
		d.DispatcherBase.GetLogger().Error("Create Redis Cluster Client Failed")
		return fmt.Errorf("create redis cluster client failed")
	}

	_, err = d.redisInstance.Ping(initCtx).Result()
	if err != nil {
		d.DispatcherBase.GetLogger().Error("check redis cluster client failed", "err", err)
		return err
	}
	return nil
}

func (d *RedisMessageDispatcher) Cleanup() {
	if d.redisInstance != nil {
		d.redisInstance.Close()
	}
	d.redisInstance = nil
}

func (d *RedisMessageDispatcher) GetRedisInstance() *redis.Client {
	if d == nil {
		return nil
	}
	return d.redisInstance
}

func (d *RedisMessageDispatcher) CreateDispatcherAwaitOptions() *DispatcherAwaitOptions {
	return &DispatcherAwaitOptions{
		Type:     uint64(uintptr(unsafe.Pointer(d))),
		Sequence: d.sequence.Add(1),
		Timeout:  time.Duration(0),
	}
}
