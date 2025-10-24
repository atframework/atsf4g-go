package atframework_component_config_custom_index_type

import (
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
)

type ExcelConfigUserLevelExpIndex struct {
	MaxLevel uint32
	MaxExp   int64
}

type ExcelConfigCustomIndex struct {
	ConstIndex        ExcelConfigConstIndex
	UserLevelExpIndex ExcelConfigUserLevelExpIndex
}

// 此处定义自定义索引的类型
type ExcelConfigConstIndex struct {
	ExcelConstConfig public_protocol_config.ExcelConstConfig
}

func (i *ExcelConfigCustomIndex) GetConstIndex() *ExcelConfigConstIndex {
	if i == nil {
		return nil
	}
	return &i.ConstIndex
}

func (i *ExcelConfigCustomIndex) GetUserExpLevelConfigIndex() *ExcelConfigUserLevelExpIndex {
	if i == nil {
		return nil
	}

	return &i.UserLevelExpIndex
}
