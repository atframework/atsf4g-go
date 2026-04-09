package atframework_component_config

import (
	"strconv"
	"strings"

	generate_config "github.com/atframework/atsf4g-go/component/config/generate_config"
	public_protocol_common "github.com/atframework/atsf4g-go/component/protocol/public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component/protocol/public/config/protocol/config"
)

func getEnumValueByName(enumName string) (int32, bool) {
	enumDescriptor := public_protocol_common.EnMailMajorType_name
	for value, name := range enumDescriptor {
		if name == enumName {
			return value, true
		}
	}

	return 0, false
}

func getAllValidEnumValues() []int32 {
	var validValues []int32
	enumDescriptor := public_protocol_common.EnMailMajorType_name

	for value := range enumDescriptor {
		if value > 0 {
			validValues = append(validValues, value)
		}
	}

	return validValues
}

func insertMailMajorType(output map[int32]struct{}, input string) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}

	// 解析为数字
	if len(input) > 0 && input[0] >= '0' && input[0] <= '9' {
		if value, err := strconv.ParseInt(input, 10, 32); err == nil && value > 0 {
			output[int32(value)] = struct{}{}
		}
		return
	}

	// 反射获取枚举值
	if value, found := getEnumValueByName(input); found && value > 0 {
		output[value] = struct{}{}
		return
	}

	if strings.HasPrefix(input, "EN_MAIL_MAJOR_") {
		if value, found := getEnumValueByName(input); found && value > 0 {
			output[value] = struct{}{}
		}
	} else {
		// 尝试添加前缀
		fullEnumName := "EN_MAIL_MAJOR_" + input
		if value, found := getEnumValueByName(fullEnumName); found && value > 0 {
			output[value] = struct{}{}
		}
	}
}

func setupMailConfig(group *generate_config.ConfigGroup) error {
	if group == nil {
		return nil
	}

	mailUserMajorTypes := make(map[int32]struct{})
	mailGlobalMajorTypes := make(map[int32]struct{})

	mailConfig := GetConfigManager().GetCurrentConfigGroup().GetSectionConfig().GetMail()

	allValidEnumValues := getAllValidEnumValues()
	for _, enumValue := range allValidEnumValues {
		mailUserMajorTypes[enumValue] = struct{}{}
	}

	if mailConfig != nil {
		// 自定义用户邮件类型配置
		for _, customType := range mailConfig.GetUserMailMajorTypes() {
			insertMailMajorType(mailUserMajorTypes, customType)
		}

		// 自定义全局邮件类型配置
		for _, customType := range mailConfig.GetGlobalMailMajorTypes() {
			insertMailMajorType(mailGlobalMajorTypes, customType)
		}
	}

	if len(mailGlobalMajorTypes) == 0 {
		defaultGlobalTypeNames := []string{
			"EN_MAIL_MAJOR_SYSTEM_LOGIC",
			"EN_MAIL_MAJOR_SYSTEM_SECURITY",
		}

		for _, typeName := range defaultGlobalTypeNames {
			insertMailMajorType(mailGlobalMajorTypes, typeName)
		}
	}

	for globalType := range mailGlobalMajorTypes {
		mailUserMajorTypes[globalType] = struct{}{}
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return nil
	}

	if customIndex.MailUserMajorTypes == nil {
		customIndex.MailUserMajorTypes = make([]int32, 0, len(mailUserMajorTypes))
	} else {
		customIndex.MailUserMajorTypes = customIndex.MailUserMajorTypes[:0]
	}
	for majorType := range mailUserMajorTypes {
		customIndex.MailUserMajorTypes = append(customIndex.MailUserMajorTypes, majorType)
	}

	if customIndex.MailGlobalMajorTypes == nil {
		customIndex.MailGlobalMajorTypes = make([]int32, 0, len(mailGlobalMajorTypes))
	} else {
		customIndex.MailGlobalMajorTypes = customIndex.MailGlobalMajorTypes[:0]
	}
	for majorType := range mailGlobalMajorTypes {
		customIndex.MailGlobalMajorTypes = append(customIndex.MailGlobalMajorTypes, majorType)
	}

	return nil
}

func initExcelMailConfigIndex(group *generate_config.ConfigGroup) error {
	return setupMailConfig(group)
}

func IsValidUserMailInner(group *generate_config.ConfigGroup, majorType int32) bool {
	if group == nil || majorType <= 0 {
		return false
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return false
	}

	for _, validType := range customIndex.MailUserMajorTypes {
		if validType == majorType {
			return true
		}
	}
	return false
}

func IsValidUserMail(majorType int32) bool {
	group := GetConfigManager().GetCurrentConfigGroup()
	if group == nil {
		return false
	}
	return IsValidUserMailInner(group, majorType)
}

func GetAllUserMailMajorTypes(group *generate_config.ConfigGroup) []int32 {
	if group == nil {
		return nil
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return nil
	}

	return customIndex.MailUserMajorTypes
}

func GetAllUserMailMajorTypesCurrent() []int32 {
	group := GetConfigManager().GetCurrentConfigGroup()
	if group == nil {
		return nil
	}
	return GetAllUserMailMajorTypes(group)
}

func IsValidGlobalMailInner(group *generate_config.ConfigGroup, majorType int32) bool {
	if group == nil || majorType <= 0 {
		return false
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return false
	}

	for _, validType := range customIndex.MailGlobalMajorTypes {
		if validType == majorType {
			return true
		}
	}
	return false
}

func IsValidGlobalMail(majorType int32) bool {
	group := GetConfigManager().GetCurrentConfigGroup()
	if group == nil {
		return false
	}
	return IsValidGlobalMailInner(group, majorType)
}

func GetAllGlobalMailMajorTypes(group *generate_config.ConfigGroup) []int32 {
	if group == nil {
		return nil
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return nil
	}

	return customIndex.MailGlobalMajorTypes
}

func GetAllGlobalMailMajorTypesCurrent() []int32 {
	group := GetConfigManager().GetCurrentConfigGroup()
	if group == nil {
		return nil
	}
	return GetAllGlobalMailMajorTypes(group)
}

func GetGlobalMailAllMajorTypeSequence(group *generate_config.ConfigGroup) []*public_protocol_config.Readonly_ExcelQuestList {
	if group == nil {
		return nil
	}

	customIndex := group.GetCustomIndex()
	if customIndex == nil {
		return nil
	}

	return customIndex.QuestSequence
}
