package atframework_component_config

import (
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	libatapp "github.com/atframework/libatapp-go"
)

func ExcelConfigCallbackOnLoad(group *generate_config.ConfigGroup, logger *libatapp.Logger) (err error) {
	// 这边初始化自定义索引
	err = initExcelConstConfigIndex(group, logger)
	if err != nil {
		return
	}

	err = initExcelUserExpLevelConfigIndex(group)
	if err != nil {
		return
	}

	err = initExcelRandomPoolConfigIndex(group)
	if err != nil {
		return
	}

	err = InitExcelQuestConfigIndex(group)
	if err != nil {
		return
	}
	return
}
