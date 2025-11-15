package lobbysvr_logic_quest_data

import (
	logic_quest "github.com/atframework/atsf4g-go/service-lobbysvr/logic/quest"
)

// 重新导出公开包中的类型，方便 impl 包内使用
type TriggerParams = logic_quest.TriggerParams

// 一些常用的构造帮助函数，模仿 C++ 中不同的构造函数用法。
// 这些函数返回 TriggerParams 的值（非指针），按需可改为返回 *TriggerParams。

// NewTriggerParamsWithXYAndItem 用于：输入 x,y 和 item change reason 以及 limit option。
func NewTriggerParamsWithXYAndItem(x, y int64) TriggerParams {
	return TriggerParams{
		X:                 x,
		HasX:              true,
		Y:                 y,
		HasY:              true,
		StrVal:            "",
		HasStrVal:         false,
		SpecifyQuestID:    0,
		HasSpecifyQuestID: false,
	}
}

// NewTriggerParamsWithXAndLimitAndQuest 用于：输入 x, limit, 可选 quest id
func NewTriggerParamsWithXAndLimitAndQuest(x int64, questID int32) TriggerParams {
	return TriggerParams{
		X:                 x,
		HasX:              true,
		Y:                 0,
		HasY:              false,
		StrVal:            "",
		HasStrVal:         false,
		SpecifyQuestID:    questID,
		HasSpecifyQuestID: questID != 0,
	}
}

// NewTriggerParamsWithXYSAndLimitAndQuest 用于：输入 x,y,str,limit,可选 quest id
func NewTriggerParamsWithXYSAndLimitAndQuest(x, y int64, s string, questID int32) TriggerParams {
	return TriggerParams{
		X:                 x,
		HasX:              true,
		Y:                 y,
		HasY:              true,
		StrVal:            s,
		HasStrVal:         true,
		SpecifyQuestID:    questID,
		HasSpecifyQuestID: questID != 0,
	}
}

// NewTriggerParamsWithStrAndLimitAndQuest 用于：只用字符串和 limit 的构造
func NewTriggerParamsWithStrAndLimitAndQuest(s string, questID int32) TriggerParams {
	return TriggerParams{
		X:                 0,
		HasX:              false,
		Y:                 0,
		HasY:              false,
		StrVal:            s,
		HasStrVal:         true,
		SpecifyQuestID:    questID,
		HasSpecifyQuestID: questID != 0,
	}
}
