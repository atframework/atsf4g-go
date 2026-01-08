package atframework_component_config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	libatapp "github.com/atframework/libatapp-go"

	logical_time "github.com/atframework/atsf4g-go/component-logical_time"
)

func initExcelConstConfigIndex(group *generate_config.ConfigGroup, logger *libatapp.Logger) error {
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

	baseTimeSec := group.GetCustomIndex().GetConstIndex().GetTimezoneBaseTimestamp().Seconds
	logical_time.SetGlobalBaseTime(time.Unix(baseTimeSec, 0))
	loadTimeOffsetConfig()
	return nil
}

// loadTimeOffsetConfig 从文件加载时间偏移配置并应用到 dispatcher
func loadTimeOffsetConfig() {
	// ../../timeOffset.txt
	ex, err := os.Executable()
	if err != nil {
		return
	}

	exePath := filepath.Dir(ex)
	timeOffsetPath := filepath.Join(exePath, "..", "..", "timeOffset.txt")

	timeOffsetPath, err = filepath.Abs(timeOffsetPath)
	if err != nil {
		return
	}

	data, err := os.ReadFile(timeOffsetPath)
	if err != nil {
		if os.IsNotExist(err) {
		}
		return
	}

	offset, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return
	}

	// 应用时间偏移（offset是秒数）
	offsetDuration := time.Duration(offset) * time.Second
	logical_time.SetGlobalLogicalOffset(offsetDuration)
}
