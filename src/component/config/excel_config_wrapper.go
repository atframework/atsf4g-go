package atframework_component_config

import (
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
)

func ExcelConfigCallbackOnLoad(group *generate_config.ConfigGroup) (err error) {
	// 这边初始化自定义索引
	err = InitExcelConstConfigIndex(group)
	if err != nil {
		return
	}

	err = InitExcelUserExpLevelConfigIndex(group)
	if err != nil {
		return
	}

	return
}
