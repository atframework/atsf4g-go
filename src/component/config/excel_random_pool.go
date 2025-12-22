package atframework_component_config

import (
	"fmt"
	"math/rand/v2"
	"sort"

	atframework_component_config_custom_index_type "github.com/atframework/atsf4g-go/component-config/custom_index"
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

// ---------------- 接口 ----------------

func RandomWithPool(poolID int32, count int64, customIndex []int32) (ret public_protocol_pbdesc.EnErrorCode, result []*public_protocol_common.DItemOffset) {
	if !IsRandomPool(poolID) {
		ret = public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_NOT_RANDOM_POOL
		return
	}

	if count == 1 {
		used := make(map[int32]struct{})
		ret, result = randomWithPoolInternal(poolID, used, customIndex)
		if ret != 0 {
			return
		}
	} else {
		for i := int64(0); i < count; i++ {
			used := make(map[int32]struct{})
			var onceResult []*public_protocol_common.DItemOffset
			ret, onceResult = randomWithPoolInternal(poolID, used, customIndex)
			if ret != 0 {
				return
			}
			result = append(result, onceResult...)
		}
	}
	return
}

func RandomPoolGetObtainedElements(group *generate_config.ConfigGroup, poolID int32) map[int32]struct{} {
	if group == nil {
		return nil
	}
	row := group.GetCustomIndex().GetRandomPool(poolID)
	if row == nil {
		return nil
	}
	return row.ObtainedElements
}

// ---------------- 内部随机逻辑 ----------------

func initExcelRandomPoolConfigIndex(group *generate_config.ConfigGroup) error {
	// 初始化随机池自定义索引
	group.GetCustomIndex().RandomPoolIndex = make(map[int32]*atframework_component_config_custom_index_type.ExcelConfigRandomPool)
	// 处理随机
	for k, v := range *group.GetExcelRandomPoolAllOfPoolId() {
		if len(v) == 0 {
			continue
		}
		// 合并相同PoolId的数据
		data, ok := group.GetCustomIndex().RandomPoolIndex[k.PoolId]
		if !ok {
			data = &atframework_component_config_custom_index_type.ExcelConfigRandomPool{
				PoolId:     k.PoolId,
				Times:      v[0].GetTimes(),
				RandomType: v[0].GetRandomType(),
				Elements:   make([]*public_protocol_config.Readonly_DRandomPoolElement, 0),
			}
			group.GetCustomIndex().RandomPoolIndex[k.PoolId] = data
		}
		for _, rows := range v {
			for _, row := range rows.GetElements() {
				if row.GetWeight() <= 0 {
					continue
				}
				data.Elements = append(data.Elements, row)
			}
		}
	}
	// 处理自选数据
	for _, v := range group.GetCustomIndex().RandomPoolIndex {
		if v.RandomType != public_protocol_config.EnRandomPoolType_EN_RANDOM_POOL_TYPE_CUSTOM {
			continue
		}
		sort.SliceStable(v.Elements, func(i, j int) bool {
			return v.Elements[i].GetIndex() < v.Elements[j].GetIndex()
		})
		for index, element := range v.Elements {
			if element.GetIndex() != int32(index) {
				return fmt.Errorf("random pool id: %d element index error %d != %d", v.PoolId, element.GetIndex(), index)
			}
		}
	}
	// 处理内容物数据
	for _, randomPool := range group.GetCustomIndex().RandomPoolIndex {
		if randomPool.ObtainedElements != nil {
			continue
		}
		used := make(map[int32]struct{})
		err := initObtainedElements(group.GetCustomIndex().RandomPoolIndex, randomPool, used, true)
		if err != nil {
			return fmt.Errorf("random pool id: %d init error %v", randomPool.PoolId, err)
		}
	}

	return nil
}

func initObtainedElements(config map[int32]*atframework_component_config_custom_index_type.ExcelConfigRandomPool, randomPool *atframework_component_config_custom_index_type.ExcelConfigRandomPool, used map[int32]struct{}, firstLevel bool) error {
	if _, exists := used[randomPool.PoolId]; exists {
		return fmt.Errorf("RandomPool RECURSIVE")
	}
	if !firstLevel && randomPool.RandomType == public_protocol_config.EnRandomPoolType_EN_RANDOM_POOL_TYPE_CUSTOM {
		// 自选不允许在第二层
		return fmt.Errorf("RandomPool Custom not in first level")
	}
	used[randomPool.PoolId] = struct{}{}

	if randomPool.ObtainedElements != nil {
		return nil
	}
	randomPool.ObtainedElements = make(map[int32]struct{})
	for _, element := range randomPool.Elements {
		if IsRandomPool(element.GetTypeId()) {
			subPool, ok := config[element.GetTypeId()]
			if !ok {
				return fmt.Errorf("RandomPool NOT FOUND")
			}
			err := initObtainedElements(config, subPool, used, false)
			if err != nil {
				return err
			}
			for obtainedElementTypeId := range subPool.ObtainedElements {
				randomPool.ObtainedElements[obtainedElementTypeId] = struct{}{}
			}
			continue
		}
		randomPool.ObtainedElements[element.GetTypeId()] = struct{}{}
	}
	return nil
}

func IsRandomPool(id int32) bool {
	// 根据类型判断是否为随机池ID
	return id >= int32(public_protocol_common.EnItemTypeRange_EN_ITEM_TYPE_RANGE_RANDOM_POOL_BEGIN) && id < int32(public_protocol_common.EnItemTypeRange_EN_ITEM_TYPE_RANGE_RANDOM_POOL_END)
}

func randomWithPoolInternal(poolID int32, used map[int32]struct{},
	customIndex []int32,
) (ret public_protocol_pbdesc.EnErrorCode, result []*public_protocol_common.DItemOffset) {
	row := GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetRandomPool(poolID)
	if row == nil {
		ret = public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_NOT_FOUND
		return
	}

	if row.Times <= 0 {
		return
	}

	if _, exists := used[poolID]; exists {
		ret = public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_RECURSIVE
		return
	}

	used[poolID] = struct{}{}

	switch row.RandomType {
	case public_protocol_config.EnRandomPoolType_EN_RANDOM_POOL_TYPE_ALL:
		ret, result = handleAll(row.Elements, row.Times, used)
	case public_protocol_config.EnRandomPoolType_EN_RANDOM_POOL_TYPE_INDEPENDENT:
		ret, result = handleIndependent(row.Elements, row.Times, used)
	case public_protocol_config.EnRandomPoolType_EN_RANDOM_POOL_TYPE_EXCLUSIVE:
		ret, result = handleExclusive(row.Elements, row.Times, used)
	case public_protocol_config.EnRandomPoolType_EN_RANDOM_POOL_TYPE_CUSTOM:
		ret, result = handleCustom(row.Elements, row.Times, used, customIndex)
	default:
		ret = public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_TYPE_NOT_SUPPORT
		return
	}
	delete(used, poolID)
	return
}

// N选1, 执行M次
func handleIndependent(elements []*public_protocol_config.Readonly_DRandomPoolElement, times int32, used map[int32]struct{}) (ret public_protocol_pbdesc.EnErrorCode, result []*public_protocol_common.DItemOffset) {
	var sumWeight int32

	for _, e := range elements {
		if validElement(e) {
			sumWeight += e.GetWeight()
		}
	}

	if sumWeight <= 0 {
		return
	}

	for i := 0; i < int(times); i++ {
		selectWeight := rand.Int32N(sumWeight)
		for _, e := range elements {
			if !validElement(e) {
				continue
			}
			if selectWeight < e.GetWeight() {
				var currentResult []*public_protocol_common.DItemOffset
				ret, currentResult = addRandomResult(e, used)
				if ret != 0 {
					return
				}
				result = append(result, currentResult...)
				break
			}
			selectWeight -= e.GetWeight()
		}
	}
	return
}

// N选M 不重复
func handleExclusive(elements []*public_protocol_config.Readonly_DRandomPoolElement, times int32, used map[int32]struct{}) (ret public_protocol_pbdesc.EnErrorCode, result []*public_protocol_common.DItemOffset) {
	valid := make([]*public_protocol_config.Readonly_DRandomPoolElement, 0)
	var sumWeight int32
	for _, e := range elements {
		if validElement(e) {
			valid = append(valid, e)
			sumWeight += e.GetWeight()
		}
	}

	if len(valid) == 0 {
		return
	}

	if int(times) >= len(valid) {
		for _, e := range valid {
			var currentResult []*public_protocol_common.DItemOffset
			ret, currentResult = addRandomResult(e, used)
			if ret != 0 {
				return
			}
			result = append(result, currentResult...)
		}
		return
	}

	for i := 0; i < int(times) && len(valid) > 0; i++ {
		selectWeight := rand.Int32N(sumWeight)
		idx := 0
		for ; idx < len(valid); idx++ {
			if selectWeight < valid[idx].GetWeight() {
				break
			}
			selectWeight -= valid[idx].GetWeight()
		}
		if idx >= len(valid) {
			idx = len(valid) - 1
		}

		opt := valid[idx]

		var currentResult []*public_protocol_common.DItemOffset
		ret, currentResult = addRandomResult(opt, used)
		if ret != 0 {
			return
		}
		result = append(result, currentResult...)
		sumWeight -= opt.GetWeight()
		// 交换两个位置
		valid[idx] = valid[len(valid)-1]
		valid = valid[:len(valid)-1]
	}
	return
}

// 自选池
func handleCustom(elements []*public_protocol_config.Readonly_DRandomPoolElement, times int32, used map[int32]struct{},
	customIndex []int32,
) (ret public_protocol_pbdesc.EnErrorCode, result []*public_protocol_common.DItemOffset) {
	if len(customIndex) != int(times) {
		ret = public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_CUSTOM_TIMES_NOT_MATCH
		return
	}
	var usedIndex map[int32]struct{}
	if times > 1 {
		usedIndex = make(map[int32]struct{})
	}

	for i := 0; i < int(times); i++ {
		if int(customIndex[i]) >= len(elements) || customIndex[i] < 0 {
			ret = public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_CUSTOM_INDEX_NOT_MATCH
			return
		}
		if usedIndex != nil {
			if _, exists := usedIndex[customIndex[i]]; exists {
				ret = public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_CUSTOM_INDEX_REPEAT
				return
			}
			usedIndex[customIndex[i]] = struct{}{}
		}
		if validElement(elements[customIndex[i]]) {
			var currentResult []*public_protocol_common.DItemOffset
			ret, currentResult = addRandomResult(elements[customIndex[i]], used)
			if ret != 0 {
				return
			}
			result = append(result, currentResult...)
		}
	}
	return
}

// 下发所有道具
func handleAll(elements []*public_protocol_config.Readonly_DRandomPoolElement, times int32, used map[int32]struct{}) (ret public_protocol_pbdesc.EnErrorCode, result []*public_protocol_common.DItemOffset) {
	for i := 0; i < int(times); i++ {
		for _, e := range elements {
			if validElement(e) {
				var currentResult []*public_protocol_common.DItemOffset
				ret, currentResult = addRandomResult(e, used)
				if ret != 0 {
					return
				}
				result = append(result, currentResult...)
			}
		}
	}
	return
}

// ---------------- 工具函数 ----------------

func addRandomResult(item *public_protocol_config.Readonly_DRandomPoolElement, used map[int32]struct{}) (ret public_protocol_pbdesc.EnErrorCode, result []*public_protocol_common.DItemOffset) {
	minCount := item.GetCount().GetMinCount()
	maxCount := item.GetCount().GetMaxCount()
	if minCount > maxCount {
		maxCount = minCount
	}

	var count int64 = 0
	if minCount == maxCount {
		count = minCount
	} else {
		count = rand.Int64N(maxCount-minCount+1) + minCount
	}
	if IsRandomPool(item.GetTypeId()) {
		for i := int32(0); i < int32(count); i++ {
			var currentResult []*public_protocol_common.DItemOffset
			ret, currentResult = randomWithPoolInternal(item.GetTypeId(), used, nil)
			if ret != 0 {
				return
			}
			result = append(result, currentResult...)
		}
		return
	}
	result = append(result, &public_protocol_common.DItemOffset{
		TypeId: item.GetTypeId(),
		Count:  count,
	})
	return
}

type itemCountExpire struct {
	count        int64
	expireOffset int64
}

func validElement(element *public_protocol_config.Readonly_DRandomPoolElement) bool {
	return element != nil && element.GetTypeId() != 0 && element.GetCount().GetMinCount() > 0 && element.GetWeight() > 0
}
