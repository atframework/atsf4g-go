package atframework_component_config_custom_index_type

import (
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
)

type ExcelConfigCustomIndex struct {
	ConstIndex        ExcelConfigConstIndex
	UserLevelExpIndex ExcelConfigUserLevelExpIndex
	RandomPoolIndex   map[int32]*ExcelConfigRandomPool
	QuestSequence     []*public_protocol_config.Readonly_ExcelQuestList
	UnlockIndex       map[public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID]interface{} // map[int32][]atframework_component_config.UnlockValueFunction
}

type ExcelConfigCustomIndexLastBuildTime struct {
	QuestCustomIndexLastBuildTime  int64
	UnlockCustomIndexLastBuildTime int64
}

type ExcelConfigRandomPool struct {
	PoolId           int32
	Times            int32
	RandomType       public_protocol_config.EnRandomPoolType
	Elements         []*public_protocol_config.Readonly_DRandomPoolElement
	ObtainedElements map[int32]struct{}
}

type ExcelConfigUserLevelExpIndex struct {
	MaxLevel uint32
	MaxExp   int64
}

type QuestUnlockConditionPair struct {
	Value   int64
	QuestId int32
}

// 此处定义自定义索引的类型.
type ExcelConfigConstIndex struct {
	ExcelConstConfig public_protocol_config.Readonly_ExcelConstConfig
}

type UnlockValueFunction struct {
	Value     int64
	Functions []FunctionUnlockID
}

type FunctionUnlockID struct {
	FunctionID public_protocol_common.EnUnlockFunctionID
	UnlockIDs  []*FunctionUnlockUnit
}

type FunctionUnlockUnit struct {
	ID               int32
	UnlockConditions []*public_protocol_common.Readonly_DFunctionUnlockCondition
}

func (i *ExcelConfigCustomIndex) GetConstIndex() *public_protocol_config.Readonly_ExcelConstConfig {
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
