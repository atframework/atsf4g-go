package atframework_component_config

import (
	"sort"
	"time"

	custom_index_type "github.com/atframework/atsf4g-go/component-config/custom_index"
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
)

// UnlockRow 统一抽象可参与解锁索引的配置行
type UnlockRow interface {
	GetUniqueId() int32
	GetUnlockCondition() []*public_protocol_common.Readonly_DFunctionUnlockCondition
}

// UnlockRowSource 提供某一类解锁行的获取方式与功能ID
type UnlockRowSource struct {
	FunctionID public_protocol_common.EnUnlockFunctionID
	Fetch      func(*generate_config.ConfigGroup) []UnlockRow
}

// moduleUnlockRow
type moduleUnlockRow struct {
	*public_protocol_config.Readonly_ExcelModuleUnlockType
}

func (m moduleUnlockRow) GetUniqueId() int32 { return m.Readonly_ExcelModuleUnlockType.GetModuleId() }
func (m moduleUnlockRow) GetUnlockCondition() []*public_protocol_common.Readonly_DFunctionUnlockCondition {
	return m.Readonly_ExcelModuleUnlockType.GetUnlockCondition()
}

// qauestListkRow
type qauestListkRow struct {
	*public_protocol_config.Readonly_ExcelQuestList
}

func (m qauestListkRow) GetUniqueId() int32 { return m.Readonly_ExcelQuestList.GetId() }
func (m qauestListkRow) GetUnlockCondition() []*public_protocol_common.Readonly_DFunctionUnlockCondition {
	ret := m.Readonly_ExcelQuestList.GetUnlockConditions()

	// 如果有可用期限设置为具体时间段，需要将其转换为解锁条件
	if m.Readonly_ExcelQuestList.GetAvailablePeriod().GetValueOneofCase() == public_protocol_config.DQuestAvailablePeriodType_EnValueID_SpecificPeriod {
		specificPeriod := m.Readonly_ExcelQuestList.GetAvailablePeriod().GetSpecificPeriod()
		if specificPeriod != nil && specificPeriod.GetStart() != nil {
			// 构造可变版本，添加时间解锁条件
			condition := &public_protocol_common.DFunctionUnlockCondition{}
			condition.MutableUnlockTimepoint().MutableStartTimepoint().Seconds = specificPeriod.GetStart().Seconds

			// 转换为只读版本
			readonlyCondition := condition.ToReadonly()
			ret = append(ret, readonlyCondition)
			return ret
		}
		for _, cond := range m.Readonly_ExcelQuestList.GetUnlockConditions() {
			if cond == nil {
				continue
			}
			ret = append(ret, cond)
		}
		return ret
	}

	return m.Readonly_ExcelQuestList.GetUnlockConditions()
}

func GetUnlockRowSources() []UnlockRowSource {
	sources := []UnlockRowSource{}

	// 模块解锁行来源
	sources = append(sources, UnlockRowSource{
		FunctionID: public_protocol_common.EnUnlockFunctionID_EN_UNLOCK_FUNCTION_ID_MODULE,
		Fetch: func(g *generate_config.ConfigGroup) []UnlockRow {
			ret := []UnlockRow{}
			rows := g.GetExcelModuleUnlockTypeAllOfModuleId()
			if rows == nil {
				return ret
			}
			for _, row := range *rows {
				if row == nil || len(row.GetUnlockCondition()) == 0 {
					continue
				}
				ret = append(ret, moduleUnlockRow{row})
			}
			return ret
		},
	})

	// 任务解锁行来源
	sources = append(sources, UnlockRowSource{
		FunctionID: public_protocol_common.EnUnlockFunctionID_EN_UNLOCK_FUNCTION_ID_QUEST,
		Fetch: func(g *generate_config.ConfigGroup) []UnlockRow {
			ret := []UnlockRow{}
			rows := g.GetExcelQuestListAllOfId()
			if rows == nil {
				return ret
			}
			for _, row := range *rows {
				if row == nil || (len(row.GetUnlockConditions()) == 0 &&
					row.GetAvailablePeriod().GetValueOneofCase() != public_protocol_config.DQuestAvailablePeriodType_EnValueID_SpecificPeriod) {
					continue
				}
				ret = append(ret, qauestListkRow{row})
			}
			return ret
		},
	})

	return sources
}

// initExcelUnlockConfigIndex 初始化所有解锁索引
func initExcelUnlockConfigIndex(group *generate_config.ConfigGroup) error {
	if group == nil {
		return nil
	}
	BuildAllUnlockIndexes(group)
	return nil
}

func RebuildExcelUnlockConfigIndex(group *generate_config.ConfigGroup) error {
	if group == nil {
		return nil
	}

	lastBuildTime := GetConfigManager().GetCurrentConfigGroup().GetCustomIndexLastBuildTime().UnlockCustomIndexLastBuildTime
	var now time.Time
	app := GetConfigManager().GetApp()
	if app != nil {
		now = app.GetSysNow()
	} else {
		now = time.Now()
	}

	unlockIndexRefreshInterval := GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetUnlockIndexRefreshInterval().GetSeconds()
	if unlockIndexRefreshInterval == 0 {
		unlockIndexRefreshInterval = defultQuestIndexRefreshInterval
	}

	if now.Unix() < lastBuildTime+unlockIndexRefreshInterval {
		return nil
	}
	GetConfigManager().GetLogger().LogInfo("RebuildExcelUnlockConfigIndex", "lastBuildTime", lastBuildTime)
	GetConfigManager().GetCurrentConfigGroup().GetCustomIndexLastBuildTime().UnlockCustomIndexLastBuildTime = now.Unix()

	BuildAllUnlockIndexes(group)
	return nil
}

// BuildAllUnlockIndexes 构建所有注册的解锁索引
func BuildAllUnlockIndexes(group *generate_config.ConfigGroup) {
	if group == nil {
		return
	}

	questIndexRefreshInterval := GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetQuestIndexRefreshInterval().GetSeconds()
	if questIndexRefreshInterval == 0 {
		questIndexRefreshInterval = defultQuestIndexRefreshInterval
	}

	questIndexMaxCacheTime := GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetQuestIndexMaxCacheTime().GetSeconds()
	if questIndexMaxCacheTime < questIndexRefreshInterval {
		questIndexMaxCacheTime = questIndexRefreshInterval * 2
	}

	idx := make(map[public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID]interface{})
	group.GetCustomIndex().UnlockIndex = idx

	for _, src := range GetUnlockRowSources() {
		rows := src.Fetch(group)
		buildUnlockIndexRows(rows, src.FunctionID, idx, questIndexMaxCacheTime)
	}

	// 排序每个条件类型的 value 列表
	for condType, v := range idx {
		if arr, ok := v.([]custom_index_type.UnlockValueFunction); ok {
			sort.Slice(arr, func(i, j int) bool { return arr[i].Value < arr[j].Value })
			idx[condType] = arr
		}
	}
}

func buildUnlockIndexRows(rows []UnlockRow, functionID public_protocol_common.EnUnlockFunctionID, idx map[public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID]interface{}, maxCacheTime int64) {
	now := time.Time{}
	atapp := GetConfigManager().GetApp()
	if atapp != nil {
		now = atapp.GetSysNow()
	} else {
		now = time.Now()
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		id := row.GetUniqueId()
		conds := row.GetUnlockCondition()
		if len(conds) == 0 {
			continue
		}

		for _, unit := range conds {
			if unit == nil {
				continue
			}

			condType := unit.GetConditionTypeOneofCase()
			if int32(condType) == 0 { // 未设置
				continue
			}

			if condType == public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_UnlockByOtherSystem ||
				condType == public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_Activate {
				// 跳过无法索引的条件类型
				continue
			}

			value := getUnlockValue(unit)

			if condType == public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_UnlockTimepoint {
				// 对于时间点类型的解锁条件，进行最大缓存时间的限制
				if value > now.Unix()+maxCacheTime {
					continue
				}
			}

			var valueData []custom_index_type.UnlockValueFunction
			if v, ok := idx[condType]; ok {
				if cast, ok2 := v.([]custom_index_type.UnlockValueFunction); ok2 {
					valueData = cast
				}
			}

			valueIdx := -1
			for i := range valueData {
				if valueData[i].Value == value {
					valueIdx = i
					break
				}
			}
			if valueIdx == -1 {
				valueData = append(valueData, custom_index_type.UnlockValueFunction{Value: value, Functions: []custom_index_type.FunctionUnlockID{}})
				valueIdx = len(valueData) - 1
			}

			funcIdx := -1
			for i := range valueData[valueIdx].Functions {
				if valueData[valueIdx].Functions[i].FunctionID == functionID {
					funcIdx = i
					break
				}
			}

			unlockUnit := &custom_index_type.FunctionUnlockUnit{ID: id, UnlockConditions: conds}
			if funcIdx == -1 {
				valueData[valueIdx].Functions = append(valueData[valueIdx].Functions, custom_index_type.FunctionUnlockID{FunctionID: functionID, UnlockIDs: []*custom_index_type.FunctionUnlockUnit{unlockUnit}})
			} else {
				valueData[valueIdx].Functions[funcIdx].UnlockIDs = append(valueData[valueIdx].Functions[funcIdx].UnlockIDs, unlockUnit)
			}

			idx[condType] = valueData
		}
	}
}

func getUnlockValue(unlockCondition *public_protocol_common.Readonly_DFunctionUnlockCondition) int64 {
	if unlockCondition == nil {
		return 0
	}

	// 根据不同的条件类型返回对应的值
	switch unlockCondition.GetConditionTypeOneofCase() {
	case public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_UnlockTimepoint:
		return unlockCondition.GetUnlockTimepoint().GetStartTimepoint().GetSeconds()
	case public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_PlayerLevel:
		return unlockCondition.GetPlayerLevel()
	// case public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_UnlockByOtherSystem:
	// 	return unlockCondition.GetUnlockByOtherSystem()
	case public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_QuestFinish:
		return unlockCondition.GetQuestFinish()
	case public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID_QuestReceived:
		return unlockCondition.GetQuestReceived()
	default:
		return 0
	}
}

func GetUnlockData(unlockType public_protocol_common.DFunctionUnlockCondition_EnConditionTypeID, previous int64, newValue int64) []custom_index_type.UnlockValueFunction {
	group := GetConfigManager().GetCurrentConfigGroup()
	if group == nil {
		return nil
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil || customIndex.UnlockIndex == nil {
		return nil
	}

	v, ok := customIndex.UnlockIndex[unlockType]
	if !ok {
		return nil
	}

	valueList, ok := v.([]custom_index_type.UnlockValueFunction)
	if !ok {
		return nil
	}

	// 使用二分查找找到范围 (previous, newValue]
	startIdx := sort.Search(len(valueList), func(i int) bool {
		return valueList[i].Value > previous
	})

	endIdx := sort.Search(len(valueList), func(i int) bool {
		return valueList[i].Value > newValue
	})

	if startIdx >= len(valueList) {
		return nil
	}

	result := make([]custom_index_type.UnlockValueFunction, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx && i < len(valueList); i++ {
		result = append(result, valueList[i])
	}

	return result
}
