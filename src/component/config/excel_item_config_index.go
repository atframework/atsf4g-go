package atframework_component_config

import (
	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
)

func MapToArrayItemOffset(
	from map[int32]*public_protocol_common.DItemOffset,
) []*public_protocol_common.DItemOffset {
	if len(from) <= 0 {
		return nil
	}

	ret := make([]*public_protocol_common.DItemOffset, 0, len(from))
	for _, item := range from {
		if item != nil && item.GetCount() > 0 {
			ret = append(ret, item)
		}
	}

	return ret
}

func MergeItemOffset(
	to map[int32]*public_protocol_common.DItemOffset,
	from ...*public_protocol_common.DItemOffset,
) {
	if to == nil || len(from) == 0 {
		return
	}

	for _, f := range from {
		if f == nil || f.GetCount() <= 0 {
			continue
		}

		item, exists := to[f.GetTypeId()]
		if exists && item != nil {
			item.Count += f.GetCount()
		} else {
			item = &public_protocol_common.DItemOffset{
				TypeId: f.GetTypeId(),
				Count:  f.GetCount(),
			}
			to[f.GetTypeId()] = item
		}
	}
}

func MergeItemOffsetCfg(
	to map[int32]*public_protocol_common.DItemOffset,
	from ...*public_protocol_common.Readonly_DItemOffset,
) {
	if to == nil || len(from) == 0 {
		return
	}

	for _, f := range from {
		if f == nil || f.GetCount() <= 0 {
			continue
		}

		item, exists := to[f.GetTypeId()]
		if exists && item != nil {
			item.Count += f.GetCount()
		} else {
			item = &public_protocol_common.DItemOffset{
				TypeId: f.GetTypeId(),
				Count:  f.GetCount(),
			}
			to[f.GetTypeId()] = item
		}
	}
}

func MergeItemOffsetFromBasic(
	to map[int32]*public_protocol_common.DItemOffset,
	from ...*public_protocol_common.DItemBasic,
) {
	if to == nil || len(from) == 0 {
		return
	}

	for _, f := range from {
		if f == nil || f.GetCount() <= 0 {
			continue
		}

		item, exists := to[f.GetTypeId()]
		if exists && item != nil {
			item.Count += f.GetCount()
		} else {
			item = &public_protocol_common.DItemOffset{
				TypeId: f.GetTypeId(),
				Count:  f.GetCount(),
			}
			to[f.GetTypeId()] = item
		}
	}
}

func MergeItemOffsetFromInstance(
	to map[int32]*public_protocol_common.DItemOffset,
	from ...*public_protocol_common.DItemInstance,
) {
	if to == nil || len(from) == 0 {
		return
	}

	for _, f := range from {
		MergeItemOffsetFromBasic(to, f.GetItemBasic())
	}
}

func MapToArrayItemBasic(
	from map[int32]map[int64]*public_protocol_common.DItemBasic,
) []*public_protocol_common.DItemBasic {
	if len(from) <= 0 {
		return nil
	}

	count := 0
	for _, itemType := range from {
		if len(itemType) <= 0 {
			continue
		}

		count += len(itemType)
	}

	ret := make([]*public_protocol_common.DItemBasic, 0, count)
	for _, itemType := range from {
		if len(itemType) <= 0 {
			continue
		}

		for _, item := range itemType {
			if item != nil && item.GetCount() > 0 {
				ret = append(ret, item)
			}
		}
	}

	return ret
}

func MergeItemBasic(
	to map[int32]map[int64]*public_protocol_common.DItemBasic,
	from ...*public_protocol_common.DItemBasic,
) {
	if to == nil || len(from) == 0 {
		return
	}

	for _, f := range from {
		if f == nil || f.GetCount() <= 0 {
			continue
		}

		itemType, exists := to[f.GetTypeId()]
		if !exists || itemType == nil {
			itemType = make(map[int64]*public_protocol_common.DItemBasic)
			to[f.GetTypeId()] = itemType
		}

		item, exists := itemType[f.GetGuid()]
		if exists && item != nil {
			item.Count += f.GetCount()
		} else {
			item = &public_protocol_common.DItemBasic{
				TypeId: f.GetTypeId(),
				Guid:   f.GetGuid(),
				Count:  f.GetCount(),
			}
			itemType[f.GetGuid()] = item
		}
	}
}

func MergeItemBasicCfg(
	to map[int32]map[int64]*public_protocol_common.DItemBasic,
	from ...*public_protocol_common.Readonly_DItemBasic,
) {
	if to == nil || len(from) == 0 {
		return
	}

	for _, f := range from {
		if f == nil || f.GetCount() <= 0 {
			continue
		}

		itemType, exists := to[f.GetTypeId()]
		if !exists || itemType == nil {
			itemType = make(map[int64]*public_protocol_common.DItemBasic)
			to[f.GetTypeId()] = itemType
		}

		item, exists := itemType[f.GetGuid()]
		if exists && item != nil {
			item.Count += f.GetCount()
		} else {
			item = &public_protocol_common.DItemBasic{
				TypeId: f.GetTypeId(),
				Guid:   f.GetGuid(),
				Count:  f.GetCount(),
			}
			itemType[f.GetGuid()] = item
		}
	}
}
