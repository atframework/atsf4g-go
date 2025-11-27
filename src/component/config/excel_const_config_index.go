package atframework_component_config

import (
	"fmt"

	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	libatapp "github.com/atframework/libatapp-go"
)

func initExcelConstConfigIndex(group *generate_config.ConfigGroup, logger *libatapp.Logger) error {
	// 这边初始化自定义索引
	source := make(map[string]interface{})
	for _, v := range *group.GetExcelOriginConstConfigAllOfKey() {
		// 把 KV 转为 Map 然后使用解析PB的工具
		source[v.GetKey()] = v.GetValue()
	}

	if len(*group.GetExcelConstConfigAllOfFakeKey()) <= 0 {
		return fmt.Errorf("excel const config is empty")
	}

	if len(*group.GetExcelConstConfigAllOfFakeKey()) > 1 {
		return fmt.Errorf("excel const config is not unique")
	}

	if group.GetCustomIndex().GetConstIndex() == nil {
		return fmt.Errorf("excel const config index is nil")
	}

	for _, v := range *group.GetExcelConstConfigAllOfFakeKey() {
		*group.GetCustomIndex().GetConstIndex() = *v
		break
	}

	return nil
}
