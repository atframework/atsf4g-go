package atframework_component_config

import (
	"sort"

	custom_index_type "github.com/atframework/atsf4g-go/component-config/custom_index"
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
)

const (
	kQuestDefSeason = -1
	// 解锁条件类型：开始时间点
	unlockConditionTypeStartTimepoint = 1
)

// InitExcelQuestConfigIndex 初始化任务配置索引
// 构建任务解锁条件映射、任务序列和任务触发器参数映射
func InitExcelQuestConfigIndex(group *generate_config.ConfigGroup) error {
	tmpUnlockConditionMap := make(map[int32]map[int32][]custom_index_type.QuestUnlockConditionPair)
	maxQuestId := int32(0)
	questSequence := make([]*public_protocol_config.ExcelQuestList, 0)

	// 遍历所有任务配置
	for _, questConf := range *group.ExcelQuestList.GetAllOfId() {
		if questConf == nil {
			continue
		}

		// 跳过已下架的任务（下架就当做删除来处理）
		if !questConf.On {
			continue
		}

		// 处理解锁条件
		for _, unlockCondition := range questConf.UnlockConditions {
			if unlockCondition == nil {
				continue
			}

			conditionTypeCase := getUnlockConditionTypeCase(unlockCondition)
			seasonId := unlockCondition.SeasonId

			if tmpUnlockConditionMap[conditionTypeCase] == nil {
				tmpUnlockConditionMap[conditionTypeCase] = make(map[int32][]custom_index_type.QuestUnlockConditionPair)
			}

			condValue := getUnlockConditionValue(unlockCondition)
			tmpUnlockConditionMap[conditionTypeCase][seasonId] = append(
				tmpUnlockConditionMap[conditionTypeCase][seasonId],
				custom_index_type.QuestUnlockConditionPair{
					Value:   condValue,
					QuestId: questConf.Id,
				},
			)
		}

		// 处理特定时间段的可用性
		if questConf.AvailablePeriod != nil && questConf.AvailablePeriod.GetSpecificPeriod() != nil {
			seasonId := getQuestSessionId(questConf)
			if tmpUnlockConditionMap[unlockConditionTypeStartTimepoint] == nil {
				tmpUnlockConditionMap[unlockConditionTypeStartTimepoint] = make(map[int32][]custom_index_type.QuestUnlockConditionPair)
			}

			startTime := questConf.AvailablePeriod.GetSpecificPeriod().Start.Seconds
			tmpUnlockConditionMap[unlockConditionTypeStartTimepoint][seasonId] = append(
				tmpUnlockConditionMap[unlockConditionTypeStartTimepoint][seasonId],
				custom_index_type.QuestUnlockConditionPair{
					Value:   startTime,
					QuestId: questConf.Id,
				},
			)
		}

		// 记录最大任务ID
		if questConf.Id > maxQuestId {
			maxQuestId = questConf.Id
		}

		// 构建任务序列
		questSequence = append(questSequence, questConf)
	}

	// 按ID排序任务序列
	sort.Slice(questSequence, func(i, j int) bool {
		return questSequence[i].Id < questSequence[j].Id
	})

	// 排序每个分类中的解锁条件对
	for _, condMap := range tmpUnlockConditionMap {
		for _, pairs := range condMap {
			sort.Slice(pairs, func(i, j int) bool {
				if pairs[i].Value == pairs[j].Value {
					return pairs[i].QuestId < pairs[j].QuestId
				}
				return pairs[i].Value < pairs[j].Value
			})
		}
	}

	// 保存到自定义索引中
	group.GetCustomIndex().QuestCurrentMaxId = maxQuestId
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
func GetBoundUnlockQuestIds(group *generate_config.ConfigGroup, unlockConditionType int32, previous, newValue int64) []int32 {
	if group == nil {
		return nil
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return nil
	}

	condTypeMap, ok := customIndex.QuestUnlockConditionMap[unlockConditionType]
	if !ok {
		return nil
	}

	seasonIds := []int32{kQuestDefSeason, 0}

	var unlockQuestIds []int32

	for _, season := range seasonIds {
		pairs, ok := condTypeMap[season]
		if !ok {
			continue
		}

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
	}

	if len(unlockQuestIds) == 0 {
		return nil
	}

	return unlockQuestIds
}

// GetEqualUnlockQuestIds 获取精确值解锁的任务ID列表
// 返回解锁条件值等于指定值的任务ID
func GetEqualUnlockQuestIds(group *generate_config.ConfigGroup, unlockConditionType, seasonId int32, value int64) []int32 {
	if group == nil {
		return nil
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return nil
	}

	condTypeMap, ok := customIndex.QuestUnlockConditionMap[unlockConditionType]
	if !ok {
		return nil
	}

	seasonIds := map[int32]bool{
		kQuestDefSeason: true,
		seasonId:        true,
	}

	var unlockQuestIds []int32

	for season := range seasonIds {
		pairs, ok := condTypeMap[season]
		if !ok {
			continue
		}

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
	}

	if len(unlockQuestIds) == 0 {
		return nil
	}

	return unlockQuestIds
}

// GetCurrentMaxQuestId 获取当前最大的任务ID
func GetCurrentMaxQuestId(group *generate_config.ConfigGroup) int32 {
	if group == nil {
		return 0
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return 0
	}

	return customIndex.QuestCurrentMaxId
}

// GetQuestSequence 获取排序后的任务序列
func GetQuestSequence(group *generate_config.ConfigGroup) []*public_protocol_config.ExcelQuestList {
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

// 辅助函数

// getUnlockConditionTypeCase 获取解锁条件的oneof字段类型（字段编号）
// 返回当前设置的oneof字段的编号，如果没有字段被设置则返回0
func getUnlockConditionTypeCase(unlockCondition *public_protocol_common.DQuestUnlockConditionItem) int32 {
	if unlockCondition == nil {
		return 0
	}

	// 获取oneof字段 - 当前设置的字段
	// 这对应于protobuf中的unlock_type_case()
	descriptor := unlockCondition.ProtoReflect().Descriptor()
	oneofDesc := descriptor.Oneofs().ByName("unlock_type")
	if oneofDesc == nil {
		return 0
	}

	field := unlockCondition.ProtoReflect().WhichOneof(oneofDesc)
	if field == nil {
		return 0
	}

	// 返回当前设置的oneof字段的编号
	return int32(field.Number())
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

// getQuestSessionId 获取任务对应的赛季ID
// 从任务进度中获取第一个进度的赛季ID
func getQuestSessionId(quest *public_protocol_config.ExcelQuestList) int32 {
	if quest == nil || len(quest.Progress) == 0 {
		return 0
	}

	// 从第一个进度获取赛季ID
	for _, progress := range quest.Progress {
		if progress != nil {
			return progress.SeasonId
		}
	}

	return 0
}

// predealQuestTriggerArgs 预处理任务触发器参数
// 构建触发器类型与其对应的所有实体参数的映射
func predealQuestTriggerArgs(group *generate_config.ConfigGroup) {
	questProgressTypeToTriggerMap := make(map[int32]int32)

	// 构建任务进度类型到触发器类型的映射
	for _, triggerConf := range *group.ExcelQuestTriggerEventType.GetAllOfId() {
		if triggerConf == nil {
			continue
		}

		for _, progressType := range triggerConf.ProgressTypes {
			questProgressTypeToTriggerMap[int32(progressType)] = int32(triggerConf.Id)
		}
	}

	// 构建触发器参数预处理映射
	tmpQuestTriggerArgsPredealMap := make(map[int32]map[int32]bool)

	for _, questConf := range *group.ExcelQuestList.GetAllOfId() {
		if questConf == nil {
			continue
		}

		// 跳过已下架的任务
		if !questConf.On {
			continue
		}

		for _, progress := range questConf.Progress {
			if progress == nil {
				continue
			}

			triggerTypeId := questProgressTypeToTriggerMap[int32(progress.TypeId)]
			if tmpQuestTriggerArgsPredealMap[triggerTypeId] == nil {
				tmpQuestTriggerArgsPredealMap[triggerTypeId] = make(map[int32]bool)
			}

			for _, arg := range progress.EntityArgs {
				tmpQuestTriggerArgsPredealMap[triggerTypeId][arg] = true
			}
		}
	}

	group.GetCustomIndex().QuestTriggerArgsPredealMap = tmpQuestTriggerArgsPredealMap
}
