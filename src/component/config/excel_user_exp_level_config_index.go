package atframework_component_config

import (
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"

	custom_index_type "github.com/atframework/atsf4g-go/component-config/custom_index"
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

	group.UserLevelExpIndex.MaxLevel = uint32(maxLevel)
	group.UserLevelExpIndex.MaxExp = maxExp
	return nil
}

func GetExcelUserExpLevelConfigIndex(g *generate_config.ConfigGroup) *custom_index_type.ExcelConfigUserLevelExpIndex {
	if g == nil {
		return nil
	}

	return &g.UserLevelExpIndex
}
