package atframework_component_logical_time

import (
	"sync/atomic"
	"time"
)

const (
	DaySeconds  time.Duration = time.Hour * 24
	WeekSeconds time.Duration = DaySeconds * 7
)

type nextCheckpointCacheEntry struct {
	// 全局逻辑时间偏移量，用于GM设置时间测试各类刷新功能
	globalLogicalOffset atomic.Int64
	// 全局基准时间点，标识周一零点的时间，用于计算各类刷新时间点
	globalBaseTime atomic.Int64
	// 全局基准时间点+逻辑时间偏移量，标识周一零点的时间，用于计算各类刷新时间点
	globalBaseTimeWithOffset atomic.Int64

	currentDayStart  atomic.Int64
	nextDayStart     atomic.Int64
	currentWeekStart atomic.Int64
	nextWeekStart    atomic.Int64
}

var sharedNextCheckpointCache nextCheckpointCacheEntry

func GetGlobalBaseTime() time.Time {
	return time.Unix(sharedNextCheckpointCache.globalBaseTime.Load(), 0)
}

// 设置全局基准时间点，标识周一零点的时间，用于计算各类刷新时间点
// 精确到秒
func SetGlobalBaseTime(t time.Time) {
	sharedNextCheckpointCache.globalBaseTime.Store(t.Unix())

	sharedNextCheckpointCache.globalBaseTimeWithOffset.Store(
		sharedNextCheckpointCache.globalBaseTime.Load() + sharedNextCheckpointCache.globalLogicalOffset.Load())
}

func GetGlobalLogicalOffset() time.Duration {
	return time.Duration(sharedNextCheckpointCache.globalLogicalOffset.Load())
}

// 设置全局逻辑时间偏移量，用于GM设置时间测试各类刷新功能
// 精确到秒
func SetGlobalLogicalOffset(t time.Duration) {
	t = t.Truncate(time.Second)
	sharedNextCheckpointCache.globalLogicalOffset.Store(int64(t.Seconds()))

	sharedNextCheckpointCache.globalBaseTimeWithOffset.Store(
		sharedNextCheckpointCache.globalBaseTime.Load() + sharedNextCheckpointCache.globalLogicalOffset.Load())
}

func GetSysNow() time.Time {
	return time.Now()
}

func GetLogicalNow() time.Time {
	return GetSysNow().Add(GetGlobalLogicalOffset())
}

func CalculateAnyDayOffsetWithBase(now time.Time, baseUnixSec int64, offset *time.Duration) time.Time {
	nowSec := now.Unix()

	checked := nowSec - baseUnixSec
	checked -= checked % int64(DaySeconds)
	checked += baseUnixSec

	if offset == nil {
		return time.Unix(checked, 0)
	}

	return time.Unix(checked, 0).Add(*offset)
}

func CalculateAnyDayOffset(now time.Time, offset *time.Duration) time.Time {
	return CalculateAnyDayOffsetWithBase(now, sharedNextCheckpointCache.globalBaseTime.Load(), offset)
}

func CalculateDayStart(now time.Time) time.Time {
	return CalculateAnyDayOffset(now, nil)
}

func CalculateAnyWeekOffsetWithBase(now time.Time, baseUnixSec int64, offset *time.Duration) time.Time {
	nowSec := now.Unix()

	checked := nowSec - baseUnixSec
	checked -= checked % int64(WeekSeconds)
	checked += baseUnixSec

	if offset == nil {
		return time.Unix(checked, 0)
	}

	return time.Unix(checked, 0).Add(*offset)
}

func CalculateAnyWeekOffset(now time.Time, offset *time.Duration) time.Time {
	return CalculateAnyWeekOffsetWithBase(now, sharedNextCheckpointCache.globalBaseTime.Load(), offset)
}

func CalculateWeekStart(now time.Time) time.Time {
	return CalculateAnyWeekOffset(now, nil)
}

func refreshDayCache(now time.Time) {
	// 考虑GM回改时间和校时抖动
	currentDayStartSec := sharedNextCheckpointCache.currentDayStart.Load()
	nextDayStartSec := sharedNextCheckpointCache.nextDayStart.Load()
	if now.Unix()+int64(time.Minute.Seconds()) >= currentDayStartSec && now.Unix() < nextDayStartSec {
		return
	}

	currentDayStartSec = CalculateDayStart(now).Unix()
	sharedNextCheckpointCache.currentDayStart.Store(currentDayStartSec)
	sharedNextCheckpointCache.nextDayStart.Store(currentDayStartSec + int64(DaySeconds.Seconds()))
}

func refreshWeekCache(now time.Time) {
	// 考虑GM回改时间和校时抖动
	currentWeekStartSec := sharedNextCheckpointCache.currentWeekStart.Load()
	nextWeekStartSec := sharedNextCheckpointCache.nextWeekStart.Load()
	if now.Unix()+int64(time.Minute.Seconds()) >= currentWeekStartSec && now.Unix() < nextWeekStartSec {
		return
	}

	currentWeekStartSec = CalculateWeekStart(now).Unix()
	sharedNextCheckpointCache.currentWeekStart.Store(currentWeekStartSec)
	sharedNextCheckpointCache.nextWeekStart.Store(currentWeekStartSec + int64(WeekSeconds.Seconds()))
}

func GetTodayStartTimepoint(offset *time.Duration) time.Time {
	refreshDayCache(GetLogicalNow())

	if offset == nil {
		return time.Unix(sharedNextCheckpointCache.currentDayStart.Load(), 0)
	}
	return time.Unix(sharedNextCheckpointCache.currentDayStart.Load(), 0).Add(*offset)
}

func GetNextDayStartTimepoint(offset *time.Duration) time.Time {
	refreshDayCache(GetLogicalNow())

	if offset == nil {
		return time.Unix(sharedNextCheckpointCache.nextDayStart.Load(), 0)
	}
	return time.Unix(sharedNextCheckpointCache.nextDayStart.Load(), 0).Add(*offset)
}

func GetCurrentWeekStartTimepoint(offset *time.Duration) time.Time {
	refreshWeekCache(GetLogicalNow())

	if offset == nil {
		return time.Unix(sharedNextCheckpointCache.currentWeekStart.Load(), 0)
	}
	return time.Unix(sharedNextCheckpointCache.currentWeekStart.Load(), 0).Add(*offset)
}

func GetNextWeekStartTimepoint(offset *time.Duration) time.Time {
	refreshWeekCache(GetLogicalNow())

	if offset == nil {
		return time.Unix(sharedNextCheckpointCache.nextWeekStart.Load(), 0)
	}
	return time.Unix(sharedNextCheckpointCache.nextWeekStart.Load(), 0).Add(*offset)
}

func GetDayId(now time.Time, refreshStartOffset *time.Duration) int64 {
	nowSec := now.Unix()

	if refreshStartOffset == nil {
		nowSec -= int64(refreshStartOffset.Seconds())
	}

	baseTimeSec := sharedNextCheckpointCache.globalBaseTime.Load()
	checked := nowSec - baseTimeSec
	checked /= int64(DaySeconds.Seconds())

	return checked
}

func IsSameDay(l time.Time, r time.Time, refreshStartOffset *time.Duration) bool {
	return GetDayId(l, refreshStartOffset) == GetDayId(r, refreshStartOffset)
}

func GetWeekId(now time.Time, refreshStartOffset *time.Duration) int64 {
	nowSec := now.Unix()

	if refreshStartOffset == nil {
		nowSec -= int64(refreshStartOffset.Seconds())
	}

	baseTimeSec := sharedNextCheckpointCache.globalBaseTime.Load()
	checked := nowSec - baseTimeSec
	checked /= int64(WeekSeconds.Seconds())

	return checked
}

func IsSameWeek(l time.Time, r time.Time, refreshStartOffset *time.Duration) bool {
	return GetWeekId(l, refreshStartOffset) == GetWeekId(r, refreshStartOffset)
}
