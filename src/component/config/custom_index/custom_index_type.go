package atframework_component_config_custom_index_type

import (
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
)

type ExcelConfigCustomIndex struct {
	ConstIndex                 ExcelConfigConstIndex
	UserLevelExpIndex          ExcelConfigUserLevelExpIndex
	RandomPoolIndex            map[int32]*ExcelConfigRandomPool
	QuestUnlockConditionMap    map[public_protocol_common.DQuestUnlockConditionItem_EnUnlockTypeID][]QuestUnlockConditionPair
	QuestSequence              []*public_protocol_config.Readonly_ExcelQuestList
	QuestTriggerArgsPredealMap map[int32]map[int32]bool
}

type ExcelConfigRandomPool struct {
	Times      int32
	RandomType public_protocol_config.EnRandomPoolType
	Elements   []*public_protocol_config.Readonly_DRandomPoolElement
}

type ExcelConfigUserLevelExpIndex struct {
	MaxLevel uint32
	MaxExp   int64
}

type QuestUnlockConditionPair struct {
	Value   int64
	QuestId int32
}

// 此处定义自定义索引的类型
type ExcelConfigConstIndex struct {
	ExcelConstConfig public_protocol_config.ExcelConstConfig
}

func (i *ExcelConfigCustomIndex) GetConstIndex() *public_protocol_config.ExcelConstConfig {
	if i == nil {
		return nil
	}
	return &i.ConstIndex.ExcelConstConfig
}

func (i *ExcelConfigCustomIndex) GetUserExpLevelConfigIndex() *ExcelConfigUserLevelExpIndex {
	if i == nil {
		return nil
	}

	return &i.UserLevelExpIndex
}

func (i *ExcelConfigCustomIndex) GetRandomPool(typeId int32) *ExcelConfigRandomPool {
	if i == nil {
		return nil
	}

	return i.RandomPoolIndex[typeId]
}
