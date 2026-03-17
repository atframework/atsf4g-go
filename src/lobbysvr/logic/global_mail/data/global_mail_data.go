package lobbysvr_logic_global_mail_data

import (
	"math"
	"time"

	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
	mail_data "github.com/atframework/atsf4g-go/service-lobbysvr/logic/mail/data"
)

// 全局邮件相关常量
// 这些常量对应C++代码中的 EN_CL_MAIL_GLOBAL_* 定义
const (
	// EN_CL_MAIL_GLOBAL_JOBS_TASK_INTERVAL 全局邮件异步任务间隔（秒）
	EN_CL_MAIL_GLOBAL_JOBS_TASK_INTERVAL int64 = 60

	// EN_CL_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT 全局邮件泄漏检查超时（秒）
	EN_CL_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT int64 = 300

	// EN_CL_MAIL_GLOBAL_LEAK_CHECK_TIMEOUT 全局邮件泄漏检查超时（秒）
	EN_CL_MAIL_GLOBAL_TIME_TOLERATE int64 = 300

	// 默认的compact_delivery_time_max_offset（30天）
	DEFAULT_COMPACT_DELIVERY_TIME_MAX_OFFSET int64 = 30 * 24 * 60 * 60

	// EN_CL_MAIL_PLAYER_TOLERANCE_ERROR_COUNT 邮件拉取容错次数
	EN_CL_MAIL_PLAYER_TOLERANCE_ERROR_COUNT uint32 = 3
)

// GlobalMailBoxEntry 全局邮件箱条目 (zone_id, mail_data)
type GlobalMailBoxEntry struct {
	ZoneId   uint32
	MailData *mail_data.MailData
}

// GlobalMailBox 全局邮件箱，按邮件ID索引
// map[mail_id] -> GlobalMailBoxEntry
type GlobalMailBox map[int64]*GlobalMailBoxEntry

// GlobalMailBoxByType 按类型分类的全局邮件箱
// map[zone_id] -> map[major_type] -> MailBox
type GlobalMailBoxByType map[uint32]map[int32]*mail_data.MailBox

// GlobalMailRecordEntry 全局邮件记录条目 (zone_id, record)
type GlobalMailRecordEntry struct {
	ZoneId uint32
	Record *public_protocol_pbdesc.DMailRecord
}

// GlobalMailRecordMap 全局邮件记录映射（用于待移除队列）
// map[mail_id] -> GlobalMailRecordEntry
type GlobalMailRecordMap map[int64]*GlobalMailRecordEntry

// MailPaddingDayTime 将时间填充到当天结束
func MailPaddingDayTime(t int64) int64 {
	if t <= 0 {
		return t
	}
	// 将时间戳向上取整到当天的23:59:59
	tm := time.Unix(t, 0).UTC()
	endOfDay := time.Date(tm.Year(), tm.Month(), tm.Day(), 23, 59, 59, 0, time.UTC)
	return endOfDay.Unix()
}

// RefreshGlobalMailBoxFutureCache 刷新全局邮件箱的未来邮件缓存
func RefreshGlobalMailBoxFutureCache(mailBox *mail_data.MailBox) {
	if mailBox == nil {
		return
	}
	now := time.Now().Unix()
	if now < mailBox.FutureCacheExpire {
		return
	}

	mailBox.FutureCacheExpire = math.MaxInt64
	mailBox.FutureMailCount = 0
	for _, mail := range mailBox.Mails {
		if mail == nil || mail.Record == nil {
			continue
		}
		if mail.Record.GetStartTime() <= now {
			continue
		}
		if mail.Record.GetStartTime() < mailBox.FutureCacheExpire {
			mailBox.FutureCacheExpire = mail.Record.GetStartTime()
		}
		mailBox.FutureMailCount++
	}
}
