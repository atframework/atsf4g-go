package lobbysvr_logic_open_platform_impl

import (
	cd "github.com/atframework/atsf4g-go/component/dispatcher"
	"github.com/atframework/libatapp-go"

	private_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/private/pbdesc/protocol/pbdesc"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"

	component_open_platform "github.com/atframework/atsf4g-go/component/open_platform"

	data "github.com/atframework/atsf4g-go/service-lobbysvr/data"

	logic_open_platform "github.com/atframework/atsf4g-go/service-lobbysvr/logic/open_platform"
	logic_user "github.com/atframework/atsf4g-go/service-lobbysvr/logic/user"
)

func init() {
	var _ logic_open_platform.UserOpenPlatformManager = (*UserOpenPlatformManagerInstance)(nil)
	data.RegisterUserModuleManagerCreator[logic_open_platform.UserOpenPlatformManager](func(_ctx cd.RpcContext,
		owner *data.User,
	) data.UserModuleManagerImpl {
		return CreateUserOpenPlatformManager(owner)
	})
}

type UserOpenPlatformManagerInstance struct {
	data.UserModuleManagerBase

	ioTask cd.TaskActionImpl

	platformUserKey         component_open_platform.OpenPlatformUserKey
	platformChannelDelegate component_open_platform.OpenPlatformChannelDelegate
	platformAuthData        component_open_platform.UserAuthData
}

func CreateUserOpenPlatformManager(owner *data.User) *UserOpenPlatformManagerInstance {
	ret := &UserOpenPlatformManagerInstance{
		UserModuleManagerBase: *data.CreateUserModuleManagerBase(owner),
	}

	return ret
}

func (m *UserOpenPlatformManagerInstance) InitFromDB(_ctx cd.RpcContext, _dbUser *private_protocol_pbdesc.DatabaseTableUser) cd.RpcResult {
	m.platformUserKey = component_open_platform.MakeOpenPlatformUserKey(m.GetOwner().GetOpenId())
	return cd.CreateRpcResultOk()
}

func (m *UserOpenPlatformManagerInstance) LoginInit(ctx cd.RpcContext) {
	if m == nil {
		return
	}

	m.UserModuleManagerBase.LoginInit(ctx)

	// Update platform nick name and avatar
	if m.platformChannelDelegate != nil && m.platformAuthData != nil {
		m.updateProfileFromPlatform(ctx)
	}
}

func (m *UserOpenPlatformManagerInstance) AwaitIoTask(ctx cd.AwaitableContext) cd.RpcResult {
	if m == nil {
		return cd.CreateRpcResultOk()
	}

	if m.ioTask == nil {
		return cd.CreateRpcResultOk()
	}

	if m.ioTask.IsExiting() {
		m.ioTask = nil
		return cd.CreateRpcResultOk()
	}

	ret := cd.AwaitTask(ctx, m.ioTask)

	if m.ioTask.IsExiting() {
		m.ioTask = nil
	}

	return ret
}

func (m *UserOpenPlatformManagerInstance) UpdateAccessToken(ctx cd.RpcContext,
	accessToken string, authData *public_protocol_pbdesc.DClientAuthAccessData,
) {
	if m == nil {
		return
	}

	m.GetOwner().MutableAccountInfo().Access = accessToken
	if authData != nil && m.platformChannelDelegate != nil {
		m.platformAuthData = m.platformChannelDelegate.MakeUserAuthData(ctx, m.platformUserKey,
			&public_protocol_pbdesc.DAccountData{
				AccountType: m.GetOwner().GetAccountInfo().GetAccountType(),
				Access:      accessToken,
				ChannelId:   m.GetOwner().GetAccountInfo().GetChannelId(),
			}, authData)
	}
}

func (m *UserOpenPlatformManagerInstance) UpdateAuthData(ctx cd.RpcContext,
	channelDelegate component_open_platform.OpenPlatformChannelDelegate, authData component_open_platform.UserAuthData,
) {
	if m == nil {
		return
	}

	if channelDelegate != nil {
		m.platformChannelDelegate = channelDelegate
	}

	if authData != nil {
		m.platformAuthData = authData
	}
}

func (m *UserOpenPlatformManagerInstance) updateProfileFromPlatform(ctx cd.RpcContext) {
	if m == nil {
		return
	}

	if m.ioTask != nil && !m.ioTask.IsExiting() {
		return
	}

	m.ioTask = cd.AsyncInvoke(ctx, "UserOpenPlatformManagerInstance.updateProfileFromPlatform", m.GetOwner().GetActorExecutor(),
		func(childCtx cd.AwaitableContext) cd.RpcResult {
			if m.platformChannelDelegate == nil || m.platformAuthData == nil {
				childCtx.LogDebug("user may already logout and skip to update profile from open platform",
					"open_id", m.GetOwner().GetOpenId(),
					"user_id", m.GetOwner().GetUserId(),
					"zone_id", m.GetOwner().GetZoneId(),
				)
				return cd.CreateRpcResultOk()
			}

			if !m.GetOwner().IsWriteable() {
				childCtx.LogDebug("user may already logout and skip to update profile from open platform",
					"open_id", m.GetOwner().GetOpenId(),
					"user_id", m.GetOwner().GetUserId(),
					"zone_id", m.GetOwner().GetZoneId(),
				)
				return cd.CreateRpcResultOk()
			}

			opMgr := libatapp.AtappGetModule[component_open_platform.OpenPlatformManager](childCtx.GetApp())
			if opMgr == nil {
				childCtx.LogError("OpenPlatformManager is not setup",
					"open_id", m.GetOwner().GetOpenId(),
					"user_id", m.GetOwner().GetUserId(),
					"zone_id", m.GetOwner().GetZoneId(),
				)
				return cd.CreateRpcResultOk()
			}

			profile, rpcErr, rpcResult := m.platformChannelDelegate.PullUserBasicProfile(childCtx, opMgr, m.platformAuthData)
			if rpcResult.IsError() {
				logArgs := []any{
					"open_id", m.GetOwner().GetOpenId(),
					"user_id", m.GetOwner().GetUserId(),
					"zone_id", m.GetOwner().GetZoneId(),
					"response_code", rpcResult.GetResponseCode(),
					"response_message", rpcResult.GetResponseMessage(),
				}
				if rpcErr != nil {
					logArgs = append(logArgs,
						"error_code", rpcErr.GetErrorCode(),
						"error_message", rpcErr.GetErrorMessage(),
					)

					// 如果是访问令牌无效导致的错误，说明用户可能已经在平台侧解绑或者访问令牌过期，需要踢下线用户
					if m.platformChannelDelegate.IsErrorInvalidAccessToken(rpcErr) {
						session := m.GetOwner().GetUserSession()
						if session != nil {
							session.Close(childCtx,
								int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_SESSION_KICKOFF_BY_SERVER),
								"invalid platform access token")
						}
					}
				}

				childCtx.LogError("failed to pull user basic profile from open platform", logArgs...)
				return rpcResult
			}

			if m.platformChannelDelegate.IsError(rpcErr) {
				childCtx.LogWarn("open platform channel delegate return error when pull user basic profile",
					"open_id", m.GetOwner().GetOpenId(),
					"user_id", m.GetOwner().GetUserId(),
					"zone_id", m.GetOwner().GetZoneId(),
					"error_code", rpcErr.GetErrorCode(),
					"error_message", rpcErr.GetErrorMessage(),
					"error_description", rpcErr.GetErrorDescription(),
				)

				// 如果是访问令牌无效导致的错误，说明用户可能已经在平台侧解绑或者访问令牌过期，需要踢下线用户
				if m.platformChannelDelegate.IsErrorInvalidAccessToken(rpcErr) {
					session := m.GetOwner().GetUserSession()
					if session != nil {
						session.Close(childCtx,
							int32(public_protocol_pbdesc.EnCloseReasonType_EN_CRT_SESSION_KICKOFF_BY_SERVER),
							"invalid platform access token")
					}
				}
				return cd.CreateRpcResultOk()
			}

			changed := false
			if profile.GetNickName() != "" && profile.GetNickName() != m.GetOwner().GetAccountInfo().GetProfile().GetPlatformNickName() {
				m.GetOwner().MutableAccountInfo().MutableProfile().PlatformNickName = profile.GetNickName()
				changed = true
			}
			if profile.GetAvatar() != "" && profile.GetAvatar() != m.GetOwner().GetAccountInfo().GetProfile().GetPlatformAvatar() {
				m.GetOwner().MutableAccountInfo().MutableProfile().PlatformAvatar = profile.GetAvatar()
				changed = true
			}
			if changed {
				ubMgr := data.UserGetModuleManager[logic_user.UserBasicManager](m.GetOwner())
				if ubMgr != nil {
					ubMgr.MarkUserProfileDirty()
				}
			}
			return cd.CreateRpcResultOk()
		})

	if m.ioTask != nil {
		m.ioTask.AddFinishCallback(func(rc cd.RpcContext) {
			m.ioTask = nil
		})
	} else {
		ctx.LogError("create async io task to pull open platform profile failed",
			"open_id", m.GetOwner().GetOpenId(),
			"user_id", m.GetOwner().GetUserId(),
			"zone_id", m.GetOwner().GetZoneId(),
		)
	}
}
