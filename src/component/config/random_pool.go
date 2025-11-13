package atframework_component_config

import (
	"math/rand/v2"

	atframework_component_config_custom_index_type "github.com/atframework/atsf4g-go/component-config/custom_index"
	generate_config "github.com/atframework/atsf4g-go/component-config/generate_config"
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"
)

// ---------------- 接口 ----------------

func RandomWithPool(poolID int32, customIndex []int32) (ret int32, result []*public_protocol_common.DItemOffset) {
	if !isRandomPool(poolID) {
		ret = int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_NOT_RANDOM_POOL)
		return
	}

	used := make(map[int32]struct{})
	ret, result = randomWithPoolInternal(poolID, used, customIndex)
	if ret == 0 {
		mergeResult(&result)
	}
	return
}

// ---------------- 内部随机逻辑 ----------------

func InitExcelRandomPoolConfigIndex(group *generate_config.ConfigGroup) error {
	// 初始化随机池自定义索引
	group.GetCustomIndex().RandomPoolIndex = make(map[int32]*atframework_component_config_custom_index_type.ExcelConfigRandomPool)
	for _, v := range *group.GetExcelRandomPoolAllOfPoolId() {
		var poolIdData *atframework_component_config_custom_index_type.ExcelConfigRandomPool
		for _, row := range v {
			if row.Element.GetWeight() <= 0 {
				continue
			}
			if poolIdData == nil {
				poolIdData = &atframework_component_config_custom_index_type.ExcelConfigRandomPool{
					Times:      row.GetTimes(),
					RandomType: row.GetRandomType(),
					Elements:   make([]*public_protocol_config.DRandomPoolElement, 0),
				}
			}
			poolIdData.Elements = append(poolIdData.Elements, row.GetElement())
		}
		if poolIdData != nil {
			group.GetCustomIndex().RandomPoolIndex[v[0].GetPoolId()] = poolIdData
		}
	}
	return nil
}

func isRandomPool(id int32) bool {
	// 根据类型判断是否为随机池ID
	return id >= int32(public_protocol_common.EnItemTypeRange_EN_ITEM_TYPE_RANGE_RANDOM_POOL_BEGIN) && id < int32(public_protocol_common.EnItemTypeRange_EN_ITEM_TYPE_RANGE_RANDOM_POOL_END)
}

func randomWithPoolInternal(poolID int32, used map[int32]struct{},
	customIndex []int32) (ret int32, result []*public_protocol_common.DItemOffset) {
	row := GetConfigManager().GetCurrentConfigGroup().GetCustomIndex().GetRandomPool(poolID)
	if row == nil {
		ret = int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_NOT_FOUND)
		return
	}

	if row.Times <= 0 {
		return
	}

	if _, exists := used[poolID]; exists {
		ret = int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_RECURSIVE)
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
		ret = int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_TYPE_NOT_SUPPORT)
		return
	}
	delete(used, poolID)
	return
}

// N选1, 执行M次
func handleIndependent(elements []*public_protocol_config.DRandomPoolElement, times int32, used map[int32]struct{}) (ret int32, result []*public_protocol_common.DItemOffset) {
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
				ret, currentResult = addRandomResult(e.GetItemOffset(), used)
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
func handleExclusive(elements []*public_protocol_config.DRandomPoolElement, times int32, used map[int32]struct{}) (ret int32, result []*public_protocol_common.DItemOffset) {
	valid := make([]*public_protocol_config.DRandomPoolElement, 0)
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
			ret, currentResult = addRandomResult(e.GetItemOffset(), used)
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
		ret, currentResult = addRandomResult(opt.GetItemOffset(), used)
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
func handleCustom(elements []*public_protocol_config.DRandomPoolElement, times int32, used map[int32]struct{},
	customIndex []int32) (ret int32, result []*public_protocol_common.DItemOffset) {
	if len(customIndex) != int(times) {
		ret = int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_CUSTOM_TIMES_NOT_MATCH)
		return
	}

	for i := 0; i < int(times); i++ {
		if int(customIndex[i]) >= len(elements) || customIndex[i] < 0 {
			ret = int32(public_protocol_pbdesc.EnErrorCode_EN_ERR_RANDOM_POOL_CUSTOM_INDEX_NOT_MATCH)
			return
		}
		if validElement(elements[customIndex[i]]) {
			var currentResult []*public_protocol_common.DItemOffset
			ret, currentResult = addRandomResult(elements[customIndex[i]].GetItemOffset(), used)
			if ret != 0 {
				return
			}
			result = append(result, currentResult...)
		}
	}
	return
}

// 下发所有道具
func handleAll(elements []*public_protocol_config.DRandomPoolElement, times int32, used map[int32]struct{}) (ret int32, result []*public_protocol_common.DItemOffset) {
	for i := 0; i < int(times); i++ {
		for _, e := range elements {
			if validElement(e) {
				var currentResult []*public_protocol_common.DItemOffset
				ret, currentResult = addRandomResult(e.GetItemOffset(), used)
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

func addRandomResult(item *public_protocol_common.DItemOffset, used map[int32]struct{}) (ret int32, result []*public_protocol_common.DItemOffset) {
	if isRandomPool(item.GetTypeId()) {
		for i := int32(0); i < int32(item.GetCount()); i++ {
			var currentResult []*public_protocol_common.DItemOffset
			ret, currentResult = randomWithPoolInternal(item.GetTypeId(), used, nil)
			if ret != 0 {
				return
			}
			result = append(result, currentResult...)
		}
		return
	}
	result = append(result, item)
	return
}

type itemCountExpire struct {
	count        int64
	expireOffset int64
}

func mergeResult(out *[]*public_protocol_common.DItemOffset) {
	tmp := make(map[int32][]itemCountExpire)
	length := 0
	for i := range *out {
		item := (*out)[i]
		v, ok := tmp[item.GetTypeId()]
		find := false
		if ok {
			for i, t := range v {
				if t.expireOffset == item.GetExpireOffset() {
					v[i].count += item.GetCount()
					find = true
					break
				}
			}
		}
		if !find {
			tmp[item.GetTypeId()] = append(tmp[item.GetTypeId()], itemCountExpire{
				count:        item.GetCount(),
				expireOffset: item.GetExpireOffset(),
			})
			length++
		}
	}
	*out = make([]*public_protocol_common.DItemOffset, 0, length)
	for k, v := range tmp {
		for i := range v {
			*out = append(*out, &public_protocol_common.DItemOffset{
				TypeId:       k,
				Count:        v[i].count,
				ExpireOffset: v[i].expireOffset,
			})
		}
	}
}

func validElement(element *public_protocol_config.DRandomPoolElement) bool {
	return element != nil && element.GetItemOffset().GetTypeId() != 0 && element.GetItemOffset().GetCount() > 0 && element.GetWeight() > 0
}
