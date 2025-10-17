package atframework_component_config

import (
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	libatapp "github.com/atframework/libatapp-go"
)

func InitExcelConstConfigIndex(group *generate_config.ConfigGroup) error {
	// 这边初始化自定义索引
	source := make(map[string]interface{})
	for _, v := range *group.ExcelOriginConstConfig.GetAllOfKey() {
		// 把 KV 转为 Map 然后使用解析PB的工具
		source[v.Key] = v.Value
	}

	return libatapp.ParseMessage(source, &group.ExcelConstConfig.ExcelConstConfig, nil)
}
