package libatframe_utils_time_utility

import "time"

func IsSameDay(l time.Time, r time.Time, offset time.Duration) bool {
	if offset == 0 {
		return l.Year() == r.Year() && l.YearDay() == r.YearDay()
	}
	lAdj := l.Add(offset)
	rAdj := r.Add(offset)
	return lAdj.Year() == rAdj.Year() && lAdj.YearDay() == rAdj.YearDay()
}
