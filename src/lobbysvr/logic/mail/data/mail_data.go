package lobbysvr_logic_mail_data

import (
	"math"
	"sort"
	"time"

	mail_utils "github.com/atframework/atsf4g-go/component-mail"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

// 邮件相关常量
const (
	EN_CL_MAIL_PLAYER_TOLERANCE_ERROR_COUNT uint32 = 3 // 邮件拉取容错次数
)

// MailData 邮件数据结构，包含邮件记录和内容
type MailData struct {
	Record  *public_protocol_pbdesc.DMailRecord
	Content *public_protocol_pbdesc.DMailContent
}

// MailBox 邮件箱数据结构，按major_type分类
type MailBox struct {
	Mails             map[int64]*MailData // mail_id -> MailData
	FutureMailCount   int32               // 未来邮件数量
	FutureCacheExpire int64               // 未来邮件缓存过期时间
}

// NewMailBox 创建新的邮件箱
func NewMailBox() *MailBox {
	return &MailBox{
		Mails:             make(map[int64]*MailData),
		FutureMailCount:   0,
		FutureCacheExpire: 0,
	}
}

// MailIndex 邮件索引类型 map[mail_id] -> MailData
type MailIndex map[int64]*MailData

// MailRecordMap 邮件记录映射类型 map[mail_id] -> DMailRecord
type MailRecordMap map[int64]*public_protocol_pbdesc.DMailRecord

// CopyDefenceItem 拷贝保护道具信息
type CopyDefenceItem struct {
	TypeId int32
	Guid   int64
	Count  int32
}

// CopyDefenceCache 拷贝保护缓存
type CopyDefenceCache struct {
	AttachmentGuidBelongToMailId  map[int64]int64                      // attachment_guid -> mail_id
	CheckedAttachmentGuidInMailId map[int64]map[int64]*CopyDefenceItem // mail_id -> (guid -> item)
}

// NewCopyDefenceCache 创建拷贝保护缓存
func NewCopyDefenceCache() *CopyDefenceCache {
	return &CopyDefenceCache{
		AttachmentGuidBelongToMailId:  make(map[int64]int64),
		CheckedAttachmentGuidInMailId: make(map[int64]map[int64]*CopyDefenceItem),
	}
}

// DirtyCache 脏数据缓存
type DirtyCache struct {
	DirtyMails map[int64]*public_protocol_pbdesc.DMailRecord
	NewMails   map[int64]struct{}
}

// NewDirtyCache 创建脏数据缓存
func NewDirtyCache() *DirtyCache {
	return &DirtyCache{
		DirtyMails: make(map[int64]*public_protocol_pbdesc.DMailRecord),
		NewMails:   make(map[int64]struct{}),
	}
}

// MailDataSlice 用于排序的邮件数据切片
type MailDataSlice []*MailData

func (s MailDataSlice) Len() int      { return len(s) }
func (s MailDataSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// MailDataByCompactTime 按compact time排序（升序，最早的排在前面）
type MailDataByCompactTime struct {
	MailDataSlice
	DeliveryTimeMaxOffset int64
}

func (s MailDataByCompactTime) Less(i, j int) bool {
	return s.getCompactTime(s.MailDataSlice[i]) < s.getCompactTime(s.MailDataSlice[j])
}

func (s MailDataByCompactTime) getCompactTime(mail *MailData) int64 {
	if mail == nil || mail.Record == nil {
		return 0
	}
	compactTime := mail.Record.GetStartTime()
	if mail.Record.GetDeliveryTime() > compactTime {
		compactTime = mail.Record.GetDeliveryTime()
	} else if compactTime > mail.Record.GetDeliveryTime()+s.DeliveryTimeMaxOffset {
		compactTime = mail.Record.GetDeliveryTime() + s.DeliveryTimeMaxOffset
	}
	return compactTime
}

// ---- 邮件工具函数 ----

// RefreshMailBoxFutureCache 刷新邮件箱的未来邮件缓存
func RefreshMailBoxFutureCache(mailBox *MailBox) {
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

// SortMailDataByCompactTime 按compact time对邮件进行排序
func SortMailDataByCompactTime(mails []*MailData, deliveryTimeMaxOffset int64) {
	sorter := MailDataByCompactTime{
		MailDataSlice:         mails,
		DeliveryTimeMaxOffset: deliveryTimeMaxOffset,
	}
	sort.Sort(sorter)
}

// IsMailDataShown 检查邮件数据是否应该显示
func IsMailDataShown(mail *MailData) bool {
	if mail == nil || mail.Record == nil || mail.Content == nil {
		return false
	}
	return mail_utils.IsMailShown(mail.Content, mail.Record)
}
