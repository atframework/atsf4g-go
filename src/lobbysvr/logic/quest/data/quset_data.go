package lobbysvr_logic_quest_data

import (
	"sort"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

type EventQueueItem struct {
	EventType private_protocol_pbdesc.QuestTriggerParams_EnParamID
	Params    *private_protocol_pbdesc.QuestTriggerParams
}

type ProgressKey struct {
	QuestID          int32
	ProgressUniqueID int32
}

type QuestTimePointEntry struct {
	QuestID   int32
	Timepoint int64
}

type QuestStatusEntry struct {
	QuestID int32
	Status  public_protocol_common.EnQuestStatus
}

// 类型->param1->questID列表
type UserProgreesKeyIndex = map[public_protocol_config.DQuestConditionProgress_EnProgressParamID]map[int32]map[int32]*ProgressKey

type QuestMap = map[int32]*public_protocol_pbdesc.DQuestData

type QuestTimePointEntrySortQueue struct {
	Entries  []QuestTimePointEntry
	isSorted bool
}

func (q *QuestTimePointEntrySortQueue) Insert(entry QuestTimePointEntry) {
	q.Entries = append(q.Entries, entry)
	q.isSorted = false
}

func (q *QuestTimePointEntrySortQueue) Top() *QuestTimePointEntry {
	if !q.isSorted {
		sort.Slice(q.Entries, func(i, j int) bool {
			return q.Entries[i].Timepoint < q.Entries[j].Timepoint
		})
		q.isSorted = true
	}
	return &q.Entries[0]
}

func (q *QuestTimePointEntrySortQueue) Pop() *QuestTimePointEntry {
	if len(q.Entries) == 0 {
		return nil
	}
	top := q.Top()
	q.Entries = q.Entries[1:]
	return top
}

func (q *QuestTimePointEntrySortQueue) Remove(questID int32) {
	for i, entry := range q.Entries {
		if entry.QuestID == questID {
			q.Entries = append(q.Entries[:i], q.Entries[i+1:]...)
			break
		}
	}
}

func (q *QuestTimePointEntrySortQueue) Len() int {
	return len(q.Entries)
}

type UserQuestListData struct {
	ProgressingQuests QuestMap
	CompletedQuests   QuestMap
	ReceivedQuests    QuestMap
	ExpiredQuestsID   []int32
	DeleteCache       map[int32]*private_protocol_pbdesc.QuestDeleteCache
}
