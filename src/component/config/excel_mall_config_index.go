package atframework_component_config

import (
	"fmt"

	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
)

func initExcelMallConfigIndex(group *generate_config.ConfigGroup) error {
	if group.GetExcelMallAllOfMallType() == nil {
		return nil
	}
	if group.GetExcelMallSheetAllOfMallSheetId() == nil {
		return nil
	}

	index := make(map[int32]*public_protocol_config.Readonly_ExcelMall)
	for _, v := range *group.GetExcelMallAllOfMallType() {
		if v.GetMallType() == public_protocol_common.EnMallType_EN_MALL_TYPE_INVALID {
			continue
		}
		for _, sheetId := range v.GetMallSheetIds() {
			_, ok := index[sheetId]
			if ok {
				return fmt.Errorf("mall sheet id dup, sheet %d, mall %d", sheetId, v.GetMallType())
			}
			index[sheetId] = v
		}
	}

	group.GetCustomIndex().MallIndex.MallSheetMallIndex = index
	return nil
}
