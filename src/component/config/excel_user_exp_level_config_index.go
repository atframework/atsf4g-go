package atframework_component_config

import (
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
)

func InitExcelUserExpLevelConfigIndex(group *generate_config.ConfigGroup) error {
	// 这边初始化自定义索引
	maxLevel := int32(0)
	maxExp := int64(0)

	for _, v := range *group.ExcelUserLevel.GetAllOfLevel() {
		if v == nil {
			continue
		}

		if v.Level > maxLevel {
			maxLevel = v.Level
			maxExp = v.Exp
		}
	}

	group.GetCustomIndex().GetUserExpLevelConfigIndex().MaxLevel = uint32(maxLevel)
	group.GetCustomIndex().GetUserExpLevelConfigIndex().MaxExp = maxExp
	return nil
}
