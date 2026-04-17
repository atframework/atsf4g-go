// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform

import (
	"fmt"

	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"

	op_types "github.com/atframework/atsf4g-go/component/open_platform/type"
)

type NoneChannelDelegate struct{}

var _ op_types.OpenPlatformChannelDelegate = (*NoneChannelDelegate)(nil)

func (d *NoneChannelDelegate) IsError(r op_types.OpenPlatformRpcError) bool {
	if r == nil {
		return false
	}

	return r.GetErrorCode() != 0 || r.GetErrorMessage() != ""
}

func (d *NoneChannelDelegate) IsOk(r op_types.OpenPlatformRpcError) bool {
	if r == nil {
		return true
	}

	return r.GetErrorCode() == 0 && r.GetErrorMessage() == ""
}

func (d *NoneChannelDelegate) IsErrorInvalidAccessToken(_r op_types.OpenPlatformRpcError) bool {
	return false
}

func (d *NoneChannelDelegate) IsErrorNotSupported(_r op_types.OpenPlatformRpcError) bool {
	return false
}

func (d *NoneChannelDelegate) MakeUserAuthData(ctx cd.RpcContext,
	userKey op_types.OpenPlatformUserKey,
	accountData *public_protocol_pbdesc.DAccountData,
	accessData *public_protocol_pbdesc.DClientAuthAccessData,
) op_types.UserAuthData {
	if lu.IsNil(userKey) || accessData == nil || accountData == nil {
		return nil
	}

	return op_types.MakeUserAuthData(userKey, accountData, nil)
}

// RPCs
func (d *NoneChannelDelegate) ValidateAuthorization(ctx cd.AwaitableContext,
	manager op_types.OpenPlatformManager,
	userKey op_types.OpenPlatformUserKey,
	accountData *public_protocol_pbdesc.DAccountData,
	accessData *public_protocol_pbdesc.DClientAuthAccessData,
) (op_types.UserAuthData, op_types.OpenPlatformRpcError, cd.RpcResult) {
	if lu.IsNil(userKey) || accessData == nil || accountData == nil {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("user key/account data/access data should not be nil"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if lu.IsNil(manager) {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("manager should not be nil"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	return op_types.MakeUserAuthData(userKey, accountData, nil), nil, cd.CreateRpcResultOk()
}

func (d *NoneChannelDelegate) PullUserBasicProfile(ctx cd.AwaitableContext,
	manager op_types.OpenPlatformManager,
	userAuth op_types.UserAuthData,
) (op_types.UserBasicProfile, op_types.OpenPlatformRpcError, cd.RpcResult) {
	if lu.IsNil(userAuth) {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("user auth should not be nil"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	if lu.IsNil(manager) {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("manager should not be nil"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	return op_types.MakeUserBasicProfile("", "", userAuth.GetOpenId(), userAuth.GetOpenId()), nil, cd.CreateRpcResultOk()
}
