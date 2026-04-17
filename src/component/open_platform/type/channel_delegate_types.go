// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform_type

import (
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"

	cd "github.com/atframework/atsf4g-go/component/dispatcher"
)

type OpenPlatformChannelDelegate interface {
	IsError(r OpenPlatformRpcError) bool
	IsOk(r OpenPlatformRpcError) bool

	IsErrorInvalidAccessToken(r OpenPlatformRpcError) bool
	IsErrorNotSupported(r OpenPlatformRpcError) bool

	MakeUserAuthData(ctx cd.RpcContext,
		userKey OpenPlatformUserKey,
		accountData *public_protocol_pbdesc.DAccountData,
		accessData *public_protocol_pbdesc.DClientAuthAccessData,
	) UserAuthData

	// RPCs
	ValidateAuthorization(ctx cd.AwaitableContext,
		manager OpenPlatformManager,
		userKey OpenPlatformUserKey,
		accountData *public_protocol_pbdesc.DAccountData,
		accessData *public_protocol_pbdesc.DClientAuthAccessData,
	) (UserAuthData, OpenPlatformRpcError, cd.RpcResult)

	PullUserBasicProfile(ctx cd.AwaitableContext,
		manager OpenPlatformManager,
		userAuth UserAuthData,
	) (UserBasicProfile, OpenPlatformRpcError, cd.RpcResult)
}
