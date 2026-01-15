package atframework_component_config

import (
	log "github.com/atframework/atframe-utils-go/log"
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
)

func ExcelConfigCallbackOnLoad(group *generate_config.ConfigGroup, logger *log.Logger) (err error) {
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

	err = initExcelUnlockConfigIndex(group)
	if err != nil {
		return
	}

	err = InitExcelQuestConfigIndex(group)
	if err != nil {
		return
	}

	err = initExcelMallConfigIndex(group)
	if err != nil {
		return
	}

	return
}

func ExcelConfigCallbackRebuild(group *generate_config.ConfigGroup, logger *log.Logger) (err error) {
	err = RebuildExcelQuestConfigIndex(group)
	if err != nil {
		return
	}

	err = RebuildExcelUnlockConfigIndex(group)
	if err != nil {
		return
	}
	return
}
