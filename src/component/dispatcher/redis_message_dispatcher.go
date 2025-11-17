package atframework_component_dispatcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"sync/atomic"

	private_protocol_config "github.com/atframework/atsf4g-go/component-protocol-private/config/protocol/config"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
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
	recordPrefix  string
	casLuaSHA     string
}

// GetStableHostID 返回一个稳定的 8 位字符串
// 同一台机器固定，不同机器大概率不同
func GetStableHostID() string {
	var parts []string

	// 1️⃣ 加入操作系统类型（防止不同平台同名主机冲突）
	parts = append(parts, runtime.GOOS)

	// 2️⃣ 加入主机名
	hostname, err := os.Hostname()
	if err == nil && hostname != "" {
		parts = append(parts, hostname)
	}

	// 3️⃣ 加入第一个可用的 MAC 地址
	mac := getFirstMAC()
	if mac != "" {
		parts = append(parts, mac)
	}

	// 4️⃣ 拼接后哈希
	base := strings.Join(parts, "_")
	sum := sha256.Sum256([]byte(base))
	return hex.EncodeToString(sum[:])[:8]
}

// 获取第一个有效的 MAC 地址（跨平台）
func getFirstMAC() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if len(iface.HardwareAddr) == 0 {
			continue
		}
		return iface.HardwareAddr.String()
	}
	return ""
}

func CreateRedisMessageDispatcher(owner libatapp.AppImpl) *RedisMessageDispatcher {
	// 使用时间戳作为初始值, 避免与重启前的值冲突
	ret := &RedisMessageDispatcher{
		DispatcherBase: CreateDispatcherBase(owner),
		log:            RedisLog{app: owner},
	}
	ret.DispatcherBase.impl = ret
	ret.sequence.Store(uint64(time.Since(time.Unix(int64(private_protocol_pbdesc.EnSystemLimit_EN_SL_TIMESTAMP_FOR_ID_ALLOCATOR_OFFSET), 0)).Nanoseconds()))

	return ret
}

func (d *RedisMessageDispatcher) Name() string { return "RedisMessageDispatcher" }

var CASLuaScript = `local real_version_str = redis.call('HGET', KEYS[1], ARGV[1])
local real_version = 0
if real_version_str ~= false and real_version_str ~= nil then
    real_version = tonumber(real_version_str)
end

local expect_version = tonumber(ARGV[2])
local unpack_fn = table.unpack or unpack  -- Lua 5.1 - 5.3

if real_version == 0 or expect_version == 0 or expect_version == real_version then
    ARGV[2] = real_version + 1;
    redis.call('HSET', KEYS[1], unpack_fn(ARGV));
    return  { ok = tostring(ARGV[2]) };
else
    return  { ok = tostring(real_version) };
end`

func (d *RedisMessageDispatcher) Init(initCtx context.Context) error {
	err := d.DispatcherBase.Init(initCtx)
	if err != nil {
		return err
	}

	loadErr := libatapp.LoadConfigFromOriginData(d.GetApp().GetConfig().ConfigOriginData, "lobbysvr.redis", &d.redisCfg, d.GetLogger())
	if loadErr != nil {
		d.GetLogger().Warn("Failed to load websocket server config", "error", loadErr)
	}

	if d.redisCfg.GetAddr() == "" {
		d.DispatcherBase.GetLogger().Error("redis config error empty")
		return fmt.Errorf("redis config error empty")
	}

	if d.redisCfg.GetRecordPrefix() != "" {
		d.recordPrefix = d.redisCfg.GetRecordPrefix()
	} else if d.redisCfg.GetRandomPrefix() {
		d.recordPrefix = GetStableHostID()
	}

	d.DispatcherBase.GetLogger().Info("Redis Prefix", "Prefix", d.recordPrefix)

	redis.SetLogger(&d.log)
	d.redisInstance = redis.NewClient(&redis.Options{
		Addr:     d.redisCfg.GetAddr(),
		Password: d.redisCfg.GetPassword(),
		PoolSize: int(d.redisCfg.GetPoolSize()),
	})

	if d.redisInstance == nil {
		d.DispatcherBase.GetLogger().Error("Create Redis Cluster Client Failed")
		return fmt.Errorf("create redis cluster client failed")
	}

	sha, err := d.redisInstance.ScriptLoad(initCtx, CASLuaScript).Result()
	if err != nil {
		d.GetLogger().Error("register CAS lua script failed", "err", err)
		return err
	}
	d.casLuaSHA = sha
	d.GetLogger().Info("CAS lua script registered", "sha", sha)

	d.DispatcherBase.GetLogger().Info("Init Redis success")
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

func (d *RedisMessageDispatcher) GetRecordPrefix() string {
	return d.recordPrefix
}

func (d *RedisMessageDispatcher) PickMessageRpcName(msg *DispatcherRawMessage) string {
	return ""
}

func (d *RedisMessageDispatcher) PickMessageTaskId(msg *DispatcherRawMessage) uint64 {
	return 0
}

func (d *RedisMessageDispatcher) GetCASLuaSHA() string {
	return d.casLuaSHA
}
