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
	"sync/atomic"
	"time"

	log "github.com/atframework/atframe-utils-go/log"
	config "github.com/atframework/atsf4g-go/component-config"
	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	libatapp "github.com/atframework/libatapp-go"
	"github.com/redis/go-redis/v9"
)

func init() {
	var _ libatapp.AppModuleImpl = (*RedisMessageDispatcher)(nil)
}

type RedisLog struct {
	app libatapp.AppImpl
}

func (l *RedisLog) Printf(ctx context.Context, format string, v ...interface{}) {
	l.app.GetLogger(1).LogInner(l.app.GetSysNow(), log.GetCaller(1), ctx, slog.LevelInfo, fmt.Sprintf(format, v...))
}

type RedisMessageDispatcher struct {
	DispatcherBase
	log RedisLog

	redisInstance *redis.ClusterClient
	sequence      atomic.Uint64
	recordPrefix  string
	casLuaSHA     string
	listAddLuaSHA string
}

const (
	RedisDataVersion = 1 // 改动这个值等于清库
)

// GetStableHostID 返回一个稳定的 8 位字符串
// 同一台机器固定，不同机器大概率不同
func GetStableHostID() string {
	var parts []string

	// 加入操作系统类型（防止不同平台同名主机冲突）
	parts = append(parts, runtime.GOOS)

	// 加入主机名
	hostname, err := os.Hostname()
	if err == nil && hostname != "" {
		parts = append(parts, hostname)
	}

	// 加入第一个可用的 MAC 地址
	mac := getFirstMAC()
	if mac != "" {
		parts = append(parts, mac)
	}

	parts = append(parts, fmt.Sprintf("%d", RedisDataVersion))

	// 拼接后哈希
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

if real_version == 0 or expect_version == -1 or expect_version == real_version then
    ARGV[2] = real_version + 1;
    redis.call('HSET', KEYS[1], unpack_fn(ARGV));
    return  { ok = tostring(ARGV[2]) };
else
    return  { ok = tostring(real_version) };
end`

var ListAddLuaScript = `local max_len = tonumber(ARGV[1])
local index_field = "index_number"

local fields = redis.call('HKEYS', KEYS[1])
local current_len = #fields

if current_len >= max_len + 1 then
    local min_idx = nil
    for _, field in ipairs(fields) do
        if field ~= index_field then
            local num = tonumber(field)
            if num ~= nil then
                if min_idx == nil or num < min_idx then
                    min_idx = num
                end
            end
        end
    end
    if min_idx ~= nil then
        redis.call('HDEL', KEYS[1], tostring(min_idx))
    end
end

local new_idx = redis.call('HINCRBY', KEYS[1], index_field, 1)
redis.call('HSET', KEYS[1], tostring(new_idx), ARGV[2])
return { ok = tostring(new_idx) }`

func (d *RedisMessageDispatcher) Init(initCtx context.Context) error {
	err := d.DispatcherBase.Init(initCtx)
	if err != nil {
		return err
	}

	redisCfg := config.GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetRedis()
	if len(redisCfg.GetAddrs()) == 0 {
		d.DispatcherBase.GetLogger().LogError("redis config error empty")
		return fmt.Errorf("redis config error empty")
	}

	if redisCfg.GetRecordPrefix() != "" {
		d.recordPrefix = redisCfg.GetRecordPrefix()
	} else if redisCfg.GetRandomPrefix() {
		d.recordPrefix = GetStableHostID()
	} else {
		d.recordPrefix = "default"
	}

	d.DispatcherBase.GetLogger().LogInfo("Redis Prefix", "Prefix", d.recordPrefix)

	redis.SetLogger(&d.log)
	d.redisInstance = redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    redisCfg.GetAddrs(),
		Password: redisCfg.GetPassword(),
		PoolSize: int(redisCfg.GetPoolSize()),
	})

	if d.redisInstance == nil {
		d.DispatcherBase.GetLogger().LogError("Create Redis Cluster Client Failed")
		return fmt.Errorf("create redis cluster client failed")
	}

	sha, err := d.redisInstance.ScriptLoad(initCtx, CASLuaScript).Result()
	if err != nil {
		d.GetLogger().LogError("register CAS lua script failed", "err", err)
		return err
	}
	d.casLuaSHA = sha
	d.GetLogger().LogInfo("CAS lua script registered", "sha", sha)
	sha, err = d.redisInstance.ScriptLoad(initCtx, ListAddLuaScript).Result()
	if err != nil {
		d.GetLogger().LogError("register list add CAS lua script failed", "err", err)
		return err
	}
	d.listAddLuaSHA = sha
	d.GetLogger().LogInfo("CAS lua script registered", "sha", sha)

	d.DispatcherBase.GetLogger().LogInfo("Init Redis success")
	return nil
}

func (d *RedisMessageDispatcher) Cleanup() {
	if d.redisInstance != nil {
		d.redisInstance.Close()
	}
	d.redisInstance = nil
}

func (d *RedisMessageDispatcher) GetRedisInstance() *redis.ClusterClient {
	if d == nil {
		return nil
	}
	return d.redisInstance
}

func (d *RedisMessageDispatcher) CreateDispatcherAwaitOptions() *DispatcherAwaitOptions {
	return &DispatcherAwaitOptions{
		Type:     d.GetInstanceIdent(),
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

func (d *RedisMessageDispatcher) GetListAddLuaSHA() string {
	return d.listAddLuaSHA
}
