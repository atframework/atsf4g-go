package atframework_component_config

import (
	"time"

	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	libatapp "github.com/atframework/libatapp-go"

	logical_time "github.com/atframework/atsf4g-go/component-logical_time"
)

func initExcelConstConfigIndex(group *generate_config.ConfigGroup, logger *libatapp.Logger) error {
	// 这边初始化自定义索引
	source := make(map[string]interface{})
	for _, v := range *group.GetExcelOriginConstConfigAllOfKey() {
		// 把 KV 转为 Map 然后使用解析PB的工具
		source[v.GetKey()] = v.GetValue()
	}

	err := libatapp.ParseMessage(source, group.GetCustomIndex().GetConstIndex(), logger)
	if err != nil {
		baseTimeSec := group.GetCustomIndex().GetConstIndex().GetTimezoneBaseTimestamp().Seconds
		logical_time.SetGlobalBaseTime(time.Unix(baseTimeSec, 0))
	}
	return err
}
