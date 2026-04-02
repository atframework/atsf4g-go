package lobbysvr_logic_mail_data

import (
	"math"
	"sort"
	"time"

	mail_utils "github.com/atframework/atsf4g-go/component/mail"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"
)

const (
	EN_CL_MAIL_PLAYER_TOLERANCE_ERROR_COUNT uint32 = 3 // 邮件拉取容错次数
)

type MailData struct {
	Record  *public_protocol_pbdesc.DMailRecord
	Content *public_protocol_pbdesc.DMailContent
}

// MailDataLessFunc 邮件排序比较函数，返回 true 表示 a 应排在 b 前面
type MailDataLessFunc func(a, b *MailData) bool

// MailBox 邮件箱数据结构，按major_type分类
type MailBox struct {
	mails             []*MailData         // 邮件切片
	mailIndex         map[int64]*MailData // mail_id -> *MailData
	LessFunc          MailDataLessFunc    // 自定义排序比较函数（nil 则不排序）
	sortDirty         bool                // 切片顺序是否需要重排
	FutureMailCount   int32               // 未来邮件数量
	FutureCacheExpire int64               // 未来邮件缓存过期时间
}

func NewMailBox() *MailBox {
	return &MailBox{
		mailIndex:         make(map[int64]*MailData),
		FutureMailCount:   0,
		FutureCacheExpire: 0,
	}
}

func NewMailBoxWithSort(less MailDataLessFunc) *MailBox {
	return &MailBox{
		mailIndex:         make(map[int64]*MailData),
		LessFunc:          less,
		FutureMailCount:   0,
		FutureCacheExpire: 0,
	}
}

func (mb *MailBox) Len() int {
	return len(mb.mailIndex)
}

func (mb *MailBox) Get(mailId int64) *MailData {
	return mb.mailIndex[mailId]
}

func (mb *MailBox) Has(mailId int64) bool {
	_, ok := mb.mailIndex[mailId]
	return ok
}

func (mb *MailBox) Push(mailId int64, data *MailData) {
	if _, ok := mb.mailIndex[mailId]; ok {
		return
	}
	mb.mails = append(mb.mails, data)
	mb.mailIndex[mailId] = data
	if mb.LessFunc != nil {
		mb.sortDirty = true
	}
}

// InsertSorted 插入邮件并标记需要重排序
// 实际排序延迟到 Range / ToSlice / Sort 调用时执行。
func (mb *MailBox) InsertSorted(mailId int64, data *MailData) {
	if _, ok := mb.mailIndex[mailId]; ok {
		return
	}
	mb.mails = append(mb.mails, data)
	mb.mailIndex[mailId] = data
	if mb.LessFunc != nil {
		mb.sortDirty = true
	}
}

// Remove 移除邮件
func (mb *MailBox) Remove(mailId int64) *MailData {
	data, ok := mb.mailIndex[mailId]
	if !ok {
		return nil
	}
	delete(mb.mailIndex, mailId)
	for i, d := range mb.mails {
		if d == data {
			last := len(mb.mails) - 1
			mb.mails[i] = mb.mails[last]
			mb.mails[last] = nil
			mb.mails = mb.mails[:last]
			if mb.LessFunc != nil && i != last {
				mb.sortDirty = true
			}
			break
		}
	}
	return data
}

// Range 按排序顺序遍历所有邮件（如有 LessFunc 且 dirty 则先重排）。callback 返回 false 时停止遍历。
func (mb *MailBox) Range(callback func(mailId int64, data *MailData) bool) {
	mb.ensureSorted()
	for _, data := range mb.mails {
		mailId := int64(0)
		if data != nil && data.Record != nil {
			mailId = data.Record.GetMailId()
		}
		if !callback(mailId, data) {
			return
		}
	}
}

// RangeUnordered 遍历所有邮件，不保证顺序（不触发排序）。callback 返回 false 时停止遍历。
func (mb *MailBox) RangeUnordered(callback func(mailId int64, data *MailData) bool) {
	for _, data := range mb.mails {
		mailId := int64(0)
		if data != nil && data.Record != nil {
			mailId = data.Record.GetMailId()
		}
		if !callback(mailId, data) {
			return
		}
	}
}

// ToSlice 返回按排序顺序排列的邮件切片副本（如有 LessFunc 且 dirty 则先重排）
func (mb *MailBox) ToSlice() []*MailData {
	mb.ensureSorted()
	result := make([]*MailData, len(mb.mails))
	copy(result, mb.mails)
	return result
}

// Sort 使用当前 LessFunc 重新排序。LessFunc 为 nil 时不做任何操作。
func (mb *MailBox) Sort() {
	mb.sortDirty = true
	mb.ensureSorted()
}

// ensureSorted 如果 sortDirty 为 true 且 LessFunc 不为 nil，则原地重排切片。
func (mb *MailBox) ensureSorted() {
	if !mb.sortDirty || mb.LessFunc == nil || len(mb.mails) <= 1 {
		mb.sortDirty = false
		return
	}
	mb.sortDirty = false
	sort.SliceStable(mb.mails, func(i, j int) bool {
		return mb.LessFunc(mb.mails[i], mb.mails[j])
	})
}

// MailDataLessByMailIdDesc 按 mail_id 降序排列（新邮件在前）
func MailDataLessByMailIdDesc(l, r *MailData) bool {
	if l == nil || l.Record == nil {
		return false
	}
	if r == nil || r.Record == nil {
		return false
	}

	if max(l.Record.DeliveryTime, l.Record.StartTime) == max(r.Record.DeliveryTime, r.Record.StartTime) {
		return l.Record.GetMailId() > r.Record.GetMailId()
	}
	return max(l.Record.DeliveryTime, l.Record.StartTime) > max(r.Record.DeliveryTime, r.Record.StartTime)
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

type CopyDefenceCache struct {
	AttachmentGuidBelongToMailId  map[int64]int64                      // attachment_guid -> mail_id
	CheckedAttachmentGuidInMailId map[int64]map[int64]*CopyDefenceItem // mail_id -> (guid -> item)
}

func NewCopyDefenceCache() *CopyDefenceCache {
	return &CopyDefenceCache{
		AttachmentGuidBelongToMailId:  make(map[int64]int64),
		CheckedAttachmentGuidInMailId: make(map[int64]map[int64]*CopyDefenceItem),
	}
}

type DirtyCache struct {
	DirtyMails map[int64]*public_protocol_pbdesc.DMailRecord
	NewMails   map[int64]struct{}
}

func NewDirtyCache() *DirtyCache {
	return &DirtyCache{
		DirtyMails: make(map[int64]*public_protocol_pbdesc.DMailRecord),
		NewMails:   make(map[int64]struct{}),
	}
}

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
	mailBox.RangeUnordered(func(_ int64, mail *MailData) bool {
		if mail == nil || mail.Record == nil {
			return true
		}
		if mail.Record.GetStartTime() <= now {
			return true
		}
		if mail.Record.GetStartTime() < mailBox.FutureCacheExpire {
			mailBox.FutureCacheExpire = mail.Record.GetStartTime()
		}
		mailBox.FutureMailCount++
		return true
	})
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
func IsMailDataShown(now int64, mail *MailData) bool {
	if mail == nil || mail.Record == nil || mail.Content == nil {
		return false
	}
	return mail_utils.IsMailShown(now, mail.Content, mail.Record)
}
