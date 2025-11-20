package atframework_component_config

import (
	"sort"

	custom_index_type "github.com/atframework/atsf4g-go/component-config/custom_index"
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
)

const (
	// 解锁条件类型：开始时间点
	unlockConditionTypeStartTimepoint = 1
)

// 构建任务解锁条件映射、任务序列和任务触发器参数映射
func InitExcelQuestConfigIndex(group *generate_config.ConfigGroup) error {
	tmpUnlockConditionMap := make(map[public_protocol_common.DQuestUnlockConditionItem_EnUnlockTypeID][]custom_index_type.QuestUnlockConditionPair)
	questSequence := make([]*public_protocol_config.Readonly_ExcelQuestList, 0)

	// 遍历所有任务配置
	for _, questConf := range *group.GetExcelQuestListAllOfId() {
		if questConf == nil {
			continue
		}

		// 跳过已下架的任务
		if !questConf.GetOn() {
			continue
		}

		for _, unlockCondition := range questConf.GetUnlockConditions() {
			if unlockCondition == nil {
				continue
			}

			conditionTypeCase := unlockCondition.GetUnlockTypeOneofCase()

			condValue := getUnlockConditionValue(unlockCondition.ToMessage())
			tmpUnlockConditionMap[conditionTypeCase] = append(
				tmpUnlockConditionMap[conditionTypeCase],
				custom_index_type.QuestUnlockConditionPair{
					Value:   condValue,
					QuestId: questConf.GetId(),
				},
			)
		}

		// 处理特定时间段的可用性
		if questConf.GetAvailablePeriod().GetSpecificPeriod().GetStart() != nil {
			startTime := questConf.GetAvailablePeriod().GetSpecificPeriod().GetStart().Seconds
			tmpUnlockConditionMap[unlockConditionTypeStartTimepoint] = append(
				tmpUnlockConditionMap[unlockConditionTypeStartTimepoint],
				custom_index_type.QuestUnlockConditionPair{
					Value:   startTime,
					QuestId: questConf.GetId(),
				},
			)
		}

		// 构建任务序列
		questSequence = append(questSequence, questConf)
	}

	// 按ID排序任务序列
	sort.Slice(questSequence, func(i, j int) bool {
		return questSequence[i].GetId() < questSequence[j].GetId()
	})

	// 排序每个分类中的解锁条件对
	for _, pairs := range tmpUnlockConditionMap {
		sort.Slice(pairs, func(i, j int) bool {
			if pairs[i].Value == pairs[j].Value {
				return pairs[i].QuestId < pairs[j].QuestId
			}
			return pairs[i].Value < pairs[j].Value
		})
	}

	group.GetCustomIndex().QuestUnlockConditionMap = tmpUnlockConditionMap
	group.GetCustomIndex().QuestSequence = questSequence

	// 预处理任务触发器参数
	predealQuestTriggerArgs(group)

	return nil
}

// GetBoundUnlockQuestIds 获取在指定范围内解锁的任务ID列表
// previous: 前置值（不包含）
// newValue: 新值（包含）
// 返回在(previous, newValue]范围内解锁的任务ID
func GetBoundUnlockQuestIds(group *generate_config.ConfigGroup, unlockConditionType public_protocol_common.DQuestUnlockConditionItem_EnUnlockTypeID, previous, newValue int64) []int32 {
	if group == nil {
		return nil
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return nil
	}

	pairs, ok := customIndex.QuestUnlockConditionMap[unlockConditionType]
	if !ok {
		return nil
	}

	var unlockQuestIds []int32

	// 使用二分查找获取范围 (previous, newValue]
	startIdx := sort.Search(len(pairs), func(i int) bool {
		return pairs[i].Value > previous
	})

	endIdx := sort.Search(len(pairs), func(i int) bool {
		return pairs[i].Value > newValue
	})

	for i := startIdx; i < endIdx && i < len(pairs); i++ {
		unlockQuestIds = append(unlockQuestIds, pairs[i].QuestId)
	}

	if len(unlockQuestIds) == 0 {
		return nil
	}

	return unlockQuestIds
}

// GetEqualUnlockQuestIds 获取精确值解锁的任务ID列表
// 返回解锁条件值等于指定值的任务ID
func GetEqualUnlockQuestIds(group *generate_config.ConfigGroup, unlockConditionType public_protocol_common.DQuestUnlockConditionItem_EnUnlockTypeID, value int64) []int32 {
	if group == nil {
		return nil
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return nil
	}

	pairs, ok := customIndex.QuestUnlockConditionMap[unlockConditionType]
	if !ok {
		return nil
	}

	var unlockQuestIds []int32

	// 使用二分查找获取精确值
	startIdx := sort.Search(len(pairs), func(i int) bool {
		return pairs[i].Value >= value
	})

	endIdx := sort.Search(len(pairs), func(i int) bool {
		return pairs[i].Value > value
	})

	for i := startIdx; i < endIdx && i < len(pairs); i++ {
		unlockQuestIds = append(unlockQuestIds, pairs[i].QuestId)
	}

	if len(unlockQuestIds) == 0 {
		return nil
	}

	return unlockQuestIds
}

// GetQuestSequence 获取排序后的任务序列
func GetQuestSequence(group *generate_config.ConfigGroup) []*public_protocol_config.Readonly_ExcelQuestList {
	if group == nil {
		return nil
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return nil
	}

	return customIndex.QuestSequence
}

// CheckIsInQuestTriggerArgs 检查实体参数是否在指定触发器类型的触发器参数中
// 若任何关键信息不存在，则默认返回true（允许通过）
func CheckIsInQuestTriggerArgs(group *generate_config.ConfigGroup, triggerType int32, entityArg int32) bool {
	if group == nil {
		return true // 若group为nil，默认返回true
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return true // 若customIndex为nil，默认返回true
	}

	triggerArgsMap, ok := customIndex.QuestTriggerArgsPredealMap[triggerType]
	if !ok {
		return true // 若触发器类型未找到，默认返回true
	}

	_, exists := triggerArgsMap[entityArg]
	return exists
}

// getUnlockConditionValue 获取解锁条件的值
// 根据oneof字段的类型提取对应的值，如果没有字段被设置则返回0
func getUnlockConditionValue(unlockCondition *public_protocol_common.DQuestUnlockConditionItem) int64 {
	if unlockCondition == nil {
		return 0
	}

	// 使用各个解锁类型的getter方法
	if v := unlockCondition.GetStartTimepoint(); unlockCondition.GetUnlockType() == (*public_protocol_common.DQuestUnlockConditionItem_StartTimepoint)(nil) || v != 0 {
		return v
	}
	if v := unlockCondition.GetPlayerLevel(); unlockCondition.GetUnlockType() == (*public_protocol_common.DQuestUnlockConditionItem_PlayerLevel)(nil) || v != 0 {
		return v
	}
	if v := unlockCondition.GetPreviousQuest(); unlockCondition.GetUnlockType() == (*public_protocol_common.DQuestUnlockConditionItem_PreviousQuest)(nil) || v != 0 {
		return v
	}
	if v := unlockCondition.GetAddByOutBusinessSystem(); unlockCondition.GetUnlockType() == (*public_protocol_common.DQuestUnlockConditionItem_AddByOutBusinessSystem)(nil) || v != 0 {
		return v
	}
	if v := unlockCondition.GetCharacterHistoryMaxLevel(); unlockCondition.GetUnlockType() == (*public_protocol_common.DQuestUnlockConditionItem_CharacterHistoryMaxLevel)(nil) || v != 0 {
		return v
	}
	if v := unlockCondition.GetHasSpecifyCharacter(); unlockCondition.GetUnlockType() == (*public_protocol_common.DQuestUnlockConditionItem_HasSpecifyCharacter)(nil) || v != 0 {
		return v
	}

	return 0
}

// 构建触发器类型与其对应的所有实体参数的映射
func predealQuestTriggerArgs(group *generate_config.ConfigGroup) {
	questProgressTypeToTriggerMap := make(map[int32]int32)

	// 构建任务进度类型到触发器类型的映射
	for _, triggerConf := range *group.GetExcelQuestTriggerEventTypeAllOfId() {
		if triggerConf == nil {
			continue
		}

		for _, progressType := range triggerConf.GetProgressTypes() {
			questProgressTypeToTriggerMap[int32(progressType)] = int32(triggerConf.GetId())
		}
	}

	// 构建触发器参数预处理映射
	tmpQuestTriggerArgsPredealMap := make(map[int32]map[int32]bool)

	for _, questConf := range *group.GetExcelQuestListAllOfId() {
		if questConf == nil {
			continue
		}

		// 跳过已下架的任务
		if !questConf.GetOn() {
			continue
		}

		for _, progress := range questConf.GetProgress() {
			if progress == nil {
				continue
			}

			triggerTypeId := questProgressTypeToTriggerMap[int32(progress.GetTypeId())]
			if tmpQuestTriggerArgsPredealMap[triggerTypeId] == nil {
				tmpQuestTriggerArgsPredealMap[triggerTypeId] = make(map[int32]bool)
			}

			for _, arg := range progress.GetEntityArgs() {
				tmpQuestTriggerArgsPredealMap[triggerTypeId][arg] = true
			}
		}
	}

	group.GetCustomIndex().QuestTriggerArgsPredealMap = tmpQuestTriggerArgsPredealMap
}
