package atframework_component_config

import (
	"sort"
	"time"

	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
)

const defultQuestIndexRefreshInterval = 7 * 24 * 3600

// 构建任务解锁条件映射、任务序列和任务触发器参数映射
func InitExcelQuestConfigIndex(group *generate_config.ConfigGroup) error {
	BuildExcelQuestConfigIndex(group)
	return nil
}

func RebuildExcelQuestConfigIndex(group *generate_config.ConfigGroup) error {
	lastBuildTime := GetConfigManager().GetCurrentConfigGroup().GetCustomIndexLastBuildTime().QuestCustomIndexLastBuildTime
	var now time.Time
	app := GetConfigManager().GetApp()
	if app != nil {
		now = app.GetSysNow()
	} else {
		now = time.Now()
	}

	questIndexRefreshInterval := GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetQuestIndexRefreshInterval().GetSeconds()
	if questIndexRefreshInterval == 0 {
		questIndexRefreshInterval = defultQuestIndexRefreshInterval
	}

	if now.Unix() < lastBuildTime+questIndexRefreshInterval {
		return nil
	}
	GetConfigManager().GetLogger().LogInfo("RebuildExcelQuestConfigIndex", "lastBuildTime", lastBuildTime)
	GetConfigManager().GetCurrentConfigGroup().GetCustomIndexLastBuildTime().QuestCustomIndexLastBuildTime = now.Unix()

	BuildExcelQuestConfigIndex(group)
	return nil
}

func BuildExcelQuestConfigIndex(group *generate_config.ConfigGroup) error {
	// 从 ConfigManager 获取 app 实例

	var now time.Time
	app := GetConfigManager().GetApp()
	if app != nil {
		now = app.GetSysNow()
	} else {
		now = time.Now()
	}

	questIndexRefreshInterval := GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetQuestIndexRefreshInterval().GetSeconds()
	if questIndexRefreshInterval == 0 {
		questIndexRefreshInterval = defultQuestIndexRefreshInterval
	}

	questSequence := make([]*public_protocol_config.Readonly_ExcelQuestList, 0)

	questIndexMaxCacheTime := GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetConstIndex().GetQuestIndexMaxCacheTime().GetSeconds()
	if questIndexMaxCacheTime < questIndexRefreshInterval {
		questIndexMaxCacheTime = questIndexRefreshInterval * 2
	}

	// 遍历所有任务配置
	for _, questConf := range *group.GetExcelQuestListAllOfId() {
		if questConf == nil {
			continue
		}

		// 跳过已下架的任务
		if !questConf.GetOn() {
			continue
		}

		// 跳过预埋很久的的任务（如果有特定时间段限制）
		if questIndexMaxCacheTime != 0 && questConf.GetAvailablePeriod().GetValueOneofCase() == public_protocol_config.DQuestAvailablePeriodType_EnValueID_SpecificPeriod {
			if questConf.GetAvailablePeriod().GetSpecificPeriod().GetStart() != nil &&
				questConf.GetAvailablePeriod().GetSpecificPeriod().GetStart().GetSeconds() >
					now.Unix()+questIndexMaxCacheTime {
				continue
			}
		}

		// 构建任务序列
		questSequence = append(questSequence, questConf)
	}

	// 按ID排序任务序列
	sort.Slice(questSequence, func(i, j int) bool {
		return questSequence[i].GetId() < questSequence[j].GetId()
	})

	group.GetCustomIndex().QuestSequence = questSequence

	return nil
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
