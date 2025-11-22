package atframework_component_logical_time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGlobalBaseTime(t *testing.T) {
	now := time.Now()
	SetGlobalBaseTime(now)
	assert.Equal(t, now.Unix(), GetGlobalBaseTime().Unix())
}

func TestGlobalLogicalOffset(t *testing.T) {
	offset := time.Hour
	SetGlobalLogicalOffset(offset)
	assert.Equal(t, int64(offset.Seconds()), int64(GetGlobalLogicalOffset().Seconds()))
}

func TestLogicalNow(t *testing.T) {
	offset := time.Hour
	SetGlobalLogicalOffset(offset)
	sysNow := GetSysNow()
	logicalNow := GetLogicalNow()
	// Allow small delta for execution time
	assert.InDelta(t, sysNow.Add(offset).Unix(), logicalNow.Unix(), 1)
}

func TestCalculateDayStart(t *testing.T) {
	// Set base time to a known Monday 00:00:00
	// 2023-01-02 is a Monday
	baseTime := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	SetGlobalBaseTime(baseTime)
	SetGlobalLogicalOffset(0)

	// Test case 1: Same day
	now := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)
	dayStart := CalculateDayStart(now)
	assert.Equal(t, baseTime.Unix(), dayStart.Unix())

	// Test case 2: Next day
	now = time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC)
	expectedStart := time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)
	dayStart = CalculateDayStart(now)
	assert.Equal(t, expectedStart.Unix(), dayStart.Unix())

	// Test case 3: With offset
	offset := time.Hour * 5 // 5 AM refresh
	dayStartWithOffset := CalculateAnyDayOffset(now, &offset)
	expectedStartWithOffset := time.Date(2023, 1, 3, 5, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedStartWithOffset.Unix(), dayStartWithOffset.Unix())
}

func TestCalculateWeekStart(t *testing.T) {
	// Set base time to a known Monday 00:00:00
	// 2023-01-02 is a Monday
	baseTime := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	SetGlobalBaseTime(baseTime)
	SetGlobalLogicalOffset(0)

	// Test case 1: Same week (Wednesday)
	now := time.Date(2023, 1, 4, 12, 0, 0, 0, time.UTC)
	weekStart := CalculateWeekStart(now)
	assert.Equal(t, baseTime.Unix(), weekStart.Unix())

	// Test case 2: Next week (Monday)
	now = time.Date(2023, 1, 9, 12, 0, 0, 0, time.UTC)
	expectedStart := time.Date(2023, 1, 9, 0, 0, 0, 0, time.UTC)
	weekStart = CalculateWeekStart(now)
	assert.Equal(t, expectedStart.Unix(), weekStart.Unix())

	// Test case 3: With offset
	offset := time.Hour * 5 // 5 AM refresh
	weekStartWithOffset := CalculateAnyWeekOffset(now, &offset)
	expectedStartWithOffset := time.Date(2023, 1, 9, 5, 0, 0, 0, time.UTC)
	assert.Equal(t, expectedStartWithOffset.Unix(), weekStartWithOffset.Unix())
}

func TestGetDayIdAndSameDay(t *testing.T) {
	baseTime := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	SetGlobalBaseTime(baseTime)

	t1 := time.Date(2023, 1, 2, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 1, 2, 20, 0, 0, 0, time.UTC)
	t3 := time.Date(2023, 1, 3, 10, 0, 0, 0, time.UTC)

	assert.True(t, IsSameDay(t1, t2, nil))
	assert.False(t, IsSameDay(t1, t3, nil))

	assert.Equal(t, int64(0), GetDayId(t1, nil))
	assert.Equal(t, int64(1), GetDayId(t3, nil))

	// Test with offset
	offset := 5 * time.Hour
	// 2023-01-02 04:00 is previous day with 5h offset (day starts at 05:00)
	t4 := time.Date(2023, 1, 2, 4, 0, 0, 0, time.UTC)
	// 2023-01-02 06:00 is current day with 5h offset
	t5 := time.Date(2023, 1, 2, 6, 0, 0, 0, time.UTC)

	assert.False(t, IsSameDay(t4, t5, &offset))
	assert.Equal(t, int64(-1), GetDayId(t4, &offset))
	assert.Equal(t, int64(0), GetDayId(t5, &offset))
}

func TestGetWeekIdAndSameWeek(t *testing.T) {
	baseTime := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	SetGlobalBaseTime(baseTime)

	t1 := time.Date(2023, 1, 2, 10, 0, 0, 0, time.UTC) // Monday
	t2 := time.Date(2023, 1, 8, 20, 0, 0, 0, time.UTC) // Sunday
	t3 := time.Date(2023, 1, 9, 10, 0, 0, 0, time.UTC) // Next Monday

	assert.True(t, IsSameWeek(t1, t2, nil))
	assert.False(t, IsSameWeek(t1, t3, nil))

	assert.Equal(t, int64(0), GetWeekId(t1, nil))
	assert.Equal(t, int64(1), GetWeekId(t3, nil))

	// Test with offset
	offset := 5 * time.Hour
	// 2023-01-02 04:00 is previous week with 5h offset (week starts at Monday 05:00)
	t4 := time.Date(2023, 1, 2, 4, 0, 0, 0, time.UTC)
	// 2023-01-02 06:00 is current week with 5h offset
	t5 := time.Date(2023, 1, 2, 6, 0, 0, 0, time.UTC)

	assert.False(t, IsSameWeek(t4, t5, &offset))
	assert.Equal(t, int64(-1), GetWeekId(t4, &offset))
	assert.Equal(t, int64(0), GetWeekId(t5, &offset))
}

func TestGetStartTimepoints(t *testing.T) {
	baseTime := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
	SetGlobalBaseTime(baseTime)
	SetGlobalLogicalOffset(0)

	// We can't easily mock GetLogicalNow() because it calls time.Now().
	// But we can check if the returned time is reasonable.

	todayStart := GetTodayStartTimepoint(nil)
	nextDayStart := GetNextDayStartTimepoint(nil)

	now := GetLogicalNow()
	assert.True(t, now.Before(nextDayStart))
	assert.True(t, todayStart.Before(now) || now.Equal(todayStart))

	assert.Equal(t, todayStart.Add(24*time.Hour).Unix(), nextDayStart.Unix())

	weekStart := GetCurrentWeekStartTimepoint(nil)
	nextWeekStart := GetNextWeekStartTimepoint(nil)

	assert.Equal(t, weekStart.Add(7*24*time.Hour).Unix(), nextWeekStart.Unix())
}
