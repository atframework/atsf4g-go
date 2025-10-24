package lobbysvr_logic_user_impl

import (
	"fmt"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-private/pbdesc/protocol/pbdesc"

	public_protocol_common "github.com/atframework/atsf4g-go/component-protocol-public/common/protocol/common"
	public_protocol_config "github.com/atframework/atsf4g-go/component-protocol-public/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component-protocol-public/pbdesc/protocol/pbdesc"

	config "github.com/atframework/atsf4g-go/component-config"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	cd "github.com/atframework/atsf4g-go/component-dispatcher"

	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

func init() {
	data.RegisterUserModuleManagerCreator[logic_user.UserBasicManager](func(_ctx *cd.RpcContext,
		owner *data.User,
	) data.UserModuleManagerImpl {
		return CreateUserBasicManager(owner)
	})
}

type UserBasicManager struct {
	data.UserModuleManagerBase

	dirtyExp      bool
	dirtyUserInfo bool
}

func CreateUserBasicManager(owner *data.User) *UserBasicManager {
	ret := &UserBasicManager{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
	}

	return ret
}

func (m *UserBasicManager) InitFromDB(_ctx *cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) DumpToDB(_ctx *cd.RpcContext, dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) RefreshLimitSecond(_ctx *cd.RpcContext) {
}

func (m *UserBasicManager) DumpUserInfo(to *public_protocol_pbdesc.DUserInfo) {
	if to == nil {
		return
	}

	loginInfo := m.GetOwner().GetLoginInfo()
	to.UserLevel = m.GetOwner().GetUserData().GetUserLevel()
	to.MutableUserStat().RegisterTime = loginInfo.GetBusinessRegisterTime()
	to.MutableUserStat().LastLoginTime = loginInfo.GetBusinessLoginTime()
}

func (m *UserBasicManager) insertDirtyHandle() {
	m.GetOwner().InsertDirtyHandleIfNotExists(m,
		func(ctx *cd.RpcContext, dirty *data.UserDirtyData) bool {
			ret := false
			dirtyData := dirty.MutableNormalDirtyChangeMessage()
			if m.dirtyExp {
				dirtyData.MutableDirtyInventory().AppendItem(&public_protocol_common.DItemInstance{
					ItemBasic: &public_protocol_common.DItemBasic{
						TypeId: int32(public_protocol_common.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_USER_EXP),
						Count:  m.GetUserExp(),
					},
				})
				ret = true
			}

			if m.dirtyUserInfo {
				m.DumpUserInfo(dirtyData.MutableUserInfo())
				ret = true
			}

			return ret
		},
		func(_ctx *cd.RpcContext) {
			m.dirtyExp = false
			m.dirtyUserInfo = false
		},
	)
}

func (m *UserBasicManager) CheckAddUserExp(_ctx *cd.RpcContext, v int64) data.Result {
	if v < 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}
	// 奖励经验永远为true
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) CheckSubUserExp(ctx *cd.RpcContext, v int64) data.Result {
	if v < 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if m.GetUserExp() < v {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_ITEM_NOT_ENOUGH)
	}

	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) addLevelUpReward(ctx *cd.RpcContext, cfg *public_protocol_config.ExcelUserLevel) {
	if cfg == nil {
		return
	}

	if len(cfg.LevelUpReward) == 0 {
		return
	}

	rewardInstances := make([]*public_protocol_common.DItemInstance, 0, len(cfg.LevelUpReward))
	for _, itemOffset := range cfg.LevelUpReward {
		itemInst, result := m.GetOwner().GenerateItemInstanceFromOffset(ctx, itemOffset)
		if result.IsError() {
			result.LogError(ctx, "generate level up reward item instance failed",
				"zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(),
				"type_id", itemOffset.GetTypeId(), "item_count", itemOffset.GetCount())
			continue
		}

		rewardInstances = append(rewardInstances, itemInst)
	}

	if len(rewardInstances) == 0 {
		return
	}

	batchAddGuard, result := m.GetOwner().CheckAddItem(ctx, rewardInstances)
	if result.IsOK() {
		m.GetOwner().AddItem(ctx, batchAddGuard, &data.ItemFlowReason{
			// TODO: 道具流水原因
		})
		return
	}

	result.LogError(ctx, "batch check add item failed",
		"zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId())

	for _, itemInst := range rewardInstances {
		singleAddGuard, result := m.GetOwner().CheckAddItem(ctx, []*public_protocol_common.DItemInstance{itemInst})
		if result.IsOK() {
			m.GetOwner().AddItem(ctx, singleAddGuard, &data.ItemFlowReason{
				// TODO: 道具流水原因
			})
			continue
		}

		result.LogError(ctx, "single check add item failed",
			"zone_id", m.GetOwner().GetZoneId(), "user_id", m.GetOwner().GetUserId(),
			"type_id", itemInst.GetItemBasic().GetTypeId(), "item_count", itemInst.GetItemBasic().GetCount())
	}
}

func (m *UserBasicManager) AddUserExp(ctx *cd.RpcContext, v int64) data.Result {
	if v < 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if v == 0 {
		return cd.CreateRpcResultOk()
	}

	configGroup := config.GetConfigManager().GetCurrentConfigGroup()
	userExpConfigIndex := configGroup.GetCustomIndex().GetUserExpLevelConfigIndex()
	if userExpConfigIndex == nil {
		return cd.CreateRpcResultError(fmt.Errorf("Can not find UserExpLevelConfigIndex"), public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	oldExp := m.GetOwner().GetUserData().GetUserExp()
	newExp := oldExp
	hasDirty := false
	if oldExp < userExpConfigIndex.MaxExp {
		if oldExp+v > userExpConfigIndex.MaxExp {
			newExp = userExpConfigIndex.MaxExp
		} else {
			newExp = oldExp + v
		}

		hasDirty = true
		m.dirtyExp = true
		m.GetOwner().MutableUserData().UserExp = newExp
	}

	checkLevelUp := m.GetUserLevel()
	for checkLevelUp < userExpConfigIndex.MaxLevel {
		checkLevelUp++

		levelConfig := configGroup.GetExcelUserLevelByLevel(int32(checkLevelUp))
		if levelConfig == nil {
			ctx.LogWarn("user level configure is missing", "level", checkLevelUp)
			continue
		}

		if newExp < levelConfig.Exp {
			break
		}

		// 升级
		m.GetOwner().MutableUserData().UserLevel = uint32(checkLevelUp)
		hasDirty = true
		m.dirtyUserInfo = true

		// 升级奖励
		m.addLevelUpReward(ctx, levelConfig)
	}

	if hasDirty {
		m.insertDirtyHandle()
	}

	// TODO: OSS经营分析统计
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) SubUserExp(_ctx *cd.RpcContext, v int64) data.Result {
	if v < 0 {
		return cd.CreateRpcResultError(nil, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if v == 0 {
		return cd.CreateRpcResultOk()
	}

	if m.GetUserExp() < v {
		m.GetOwner().MutableUserData().UserExp = 0
	} else {
		m.GetOwner().MutableUserData().UserExp -= v
	}

	m.insertDirtyHandle()

	// TODO: OSS经营分析统计
	return cd.CreateRpcResultOk()
}

func (m *UserBasicManager) GetUserExp() int64 {
	return m.GetOwner().GetUserData().GetUserExp()
}

func (m *UserBasicManager) GetUserLevel() uint32 {
	return m.GetOwner().GetUserData().GetUserLevel()
}

func (m *UserBasicManager) ForeachItem(fn func(item *public_protocol_common.DItemInstance) bool) bool {
	if fn == nil {
		return true
	}

	inst := &public_protocol_common.DItemInstance{
		ItemBasic: &public_protocol_common.DItemBasic{
			TypeId: int32(public_protocol_common.EnItemVirtualItemType_EN_ITEM_VIRTUAL_ITEM_TYPE_USER_EXP),
			Count:  m.GetUserExp(),
		},
	}

	return fn(inst)
}
