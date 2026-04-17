// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform

import (
	"encoding/json"
	"fmt"
	"strconv"

	libatapp "github.com/atframework/libatapp-go"

	private_protocol_config "github.com/atframework/atsf4g-go/component/protocol/private/config/protocol/config"
	public_protocol_pbdesc "github.com/atframework/atsf4g-go/component/protocol/public/pbdesc/protocol/pbdesc"

	lu "github.com/atframework/atframe-utils-go/lang_utility"
	cd "github.com/atframework/atsf4g-go/component/dispatcher"

	op_types "github.com/atframework/atsf4g-go/component/open_platform/type"
)

type TapTapChannelDelegate struct{}

var (
	_                   op_types.OpenPlatformChannelDelegate = (*TapTapChannelDelegate)(nil)
	platformNonceLength                                      = 16

	taptapApiPathGetBasicInfo = "/account/basic-info/v1"
	taptapApiPathGetProfile   = "/account/profile/v1"
)

func getTapTapConfigure(manager op_types.OpenPlatformManager) *private_protocol_config.Readonly_OpenPlatformChannelTaptapCfg {
	if lu.IsNil(manager) {
		return nil
	}

	configure := manager.GetConfigure()
	if configure == nil {
		return nil
	}

	channels := configure.GetChannel()
	if channels == nil {
		return nil
	}

	return channels.GetTaptap()
}

func (d *TapTapChannelDelegate) IsError(r op_types.OpenPlatformRpcError) bool {
	if r == nil {
		return false
	}

	return r.GetErrorMessage() != ""
}

func (d *TapTapChannelDelegate) IsOk(r op_types.OpenPlatformRpcError) bool {
	if r == nil {
		return true
	}

	return r.GetErrorMessage() == ""
}

func (d *TapTapChannelDelegate) IsErrorInvalidAccessToken(r op_types.OpenPlatformRpcError) bool {
	if r == nil {
		return false
	}

	return r.GetErrorMessage() == "access_denied"
}

func (d *TapTapChannelDelegate) IsErrorNotSupported(r op_types.OpenPlatformRpcError) bool {
	if r == nil {
		return false
	}

	return false
}

func (d *TapTapChannelDelegate) MakeUserAuthData(ctx cd.RpcContext,
	userKey op_types.OpenPlatformUserKey,
	accountData *public_protocol_pbdesc.DAccountData,
	accessData *public_protocol_pbdesc.DClientAuthAccessData,
) op_types.UserAuthData {
	if lu.IsNil(userKey) || accessData == nil || accountData == nil {
		return nil
	}

	platPrivateData := &TapTapClientTokenData{
		Origin: accountData.GetAccess(),
	}
	if platPrivateData.Origin != "" {
		if err := json.Unmarshal([]byte(platPrivateData.Origin), &platPrivateData.Token); err != nil {
			ctx.LogWarn("access token invalid",
				"open_id", userKey.GetOpenId(),
				"error", err,
				"access_token_length", len(platPrivateData.Origin))
		}
	}

	ret := op_types.MakeUserAuthData(userKey, accountData, accessData.GetOpenPlatform().GetPlatformParameter())
	op_types.SetUserAuthDataPlatformPrivate(ret, platPrivateData)

	return ret
}

// RPCs
func (d *TapTapChannelDelegate) ValidateAuthorization(ctx cd.AwaitableContext,
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

	platformPrivateData := &TapTapClientTokenData{
		Origin: accountData.GetAccess(),
	}
	if platformPrivateData.Origin != "" {
		if err := json.Unmarshal([]byte(platformPrivateData.Origin), &platformPrivateData.Token); err != nil {
			ctx.LogWarn("access token invalid",
				"open_id", userKey.GetOpenId(),
				"error", err,
				"access_token_length", len(platformPrivateData.Origin))
			return nil, nil, cd.CreateRpcResultError(err, public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
		}
	}

	authData := op_types.MakeUserAuthData(userKey, accountData, accessData.GetOpenPlatform().GetPlatformParameter())

	taptapConf := getTapTapConfigure(manager)
	if taptapConf.GetHost() == "" || taptapConf.GetClientId() == "" {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("taptap channel not configured properly"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	// 先尝试拉取所有数据的接口
	httpClientManager := libatapp.AtappGetModule[*cd.HttpClientDispatcher](ctx.GetApp())
	if httpClientManager == nil {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("http client manager is not setup"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	nowSec := ctx.GetSysNow().Unix()
	port := taptapConf.GetPort()
	if port <= 0 {
		port = 443
	}

	uri := fmt.Sprintf("%s?client_id=%s", taptapApiPathGetBasicInfo, taptapConf.GetClientId())
	nonce := randomString(platformNonceLength)
	macToken := hmacSha1(buildSigningString(strconv.FormatInt(nowSec, 10), nonce,
		"GET", uri, taptapConf.GetHost(), strconv.FormatInt(int64(port), 10)), platformPrivateData.Token.MacKey)
	authorization := fmt.Sprintf("MAC id=\"%s\",ts=\"%d\",nonce=\"%s\",mac=\"%s\"",
		platformPrivateData.Token.Kid, nowSec, nonce, macToken)

	var fullUrl string
	if port == int32(443) {
		fullUrl = fmt.Sprintf("https://%s%s", taptapConf.GetHost(), uri)
	} else if port == int32(80) {
		fullUrl = fmt.Sprintf("http://%s%s", taptapConf.GetHost(), uri)
	} else {
		fullUrl = fmt.Sprintf("https://%s:%d%s", taptapConf.GetHost(), port, uri)
	}

	query, err := httpClientManager.CreateQuery(ctx, "GET", fullUrl, nil)
	if err != nil {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("failed to create http query: %w", err),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	query.GetHttpRequest().Header.Add("Authorization", authorization)
	rpcResult := httpClientManager.StartQuery(ctx, query)
	if rpcResult.IsError() {
		return nil, nil, rpcResult
	}

	userProfile, rpcErr, err := ParseTapTapBodyMessage[TapTapUserProfile](query.GetHttpResponseBody())
	if err != nil {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("failed to parse taptap response: %w", err),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	if rpcErr != nil {
		return nil,
			op_types.MakeOpenPlatformRpcError(rpcErr.Code, rpcErr.ErrorMessage, rpcErr.ErrorDescription),
			cd.CreateRpcResultOk()
	}

	if userProfile == nil || userProfile.OpenID != userKey.GetOpenId() {
		gotOpenID := ""
		if userProfile != nil {
			gotOpenID = userProfile.OpenID
		}

		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("expect openid: %s, got: %s", userKey.GetOpenId(), gotOpenID),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_LOGIN_AUTHORIZE)
	}

	op_types.SetUserAuthDataPlatformPrivate(authData, platformPrivateData)
	return authData, nil, cd.CreateRpcResultOk()
}

func (d *TapTapChannelDelegate) PullUserBasicProfile(ctx cd.AwaitableContext,
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

	taptapConf := getTapTapConfigure(manager)
	if taptapConf.GetHost() == "" || taptapConf.GetClientId() == "" {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("taptap channel not configured properly"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	platformPrivateData := op_types.GetUserAuthDataPlatformPrivate[*TapTapClientTokenData](userAuth)
	if platformPrivateData == nil || platformPrivateData.Origin == "" ||
		platformPrivateData.Token.Kid == "" || platformPrivateData.Token.MacKey == "" {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("platform private data invalid, maybe token is missing or invalid"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_INVALID_PARAM)
	}

	// 先尝试拉取所有数据的接口
	httpClientManager := libatapp.AtappGetModule[*cd.HttpClientDispatcher](ctx.GetApp())
	if httpClientManager == nil {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("http client manager is not setup"),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	nowSec := ctx.GetSysNow().Unix()
	port := taptapConf.GetPort()
	if port <= 0 {
		port = 443
	}

	uri := fmt.Sprintf("%s?client_id=%s", taptapApiPathGetProfile, taptapConf.GetClientId())
	nonce := randomString(platformNonceLength)
	macToken := hmacSha1(buildSigningString(strconv.FormatInt(nowSec, 10), nonce,
		"GET", uri, taptapConf.GetHost(), strconv.FormatInt(int64(port), 10)), platformPrivateData.Token.MacKey)
	authorization := fmt.Sprintf("MAC id=\"%s\",ts=\"%d\",nonce=\"%s\",mac=\"%s\"",
		platformPrivateData.Token.Kid, nowSec, nonce, macToken)

	var fullUrl string
	if port == int32(443) {
		fullUrl = fmt.Sprintf("https://%s%s", taptapConf.GetHost(), uri)
	} else if port == int32(80) {
		fullUrl = fmt.Sprintf("http://%s%s", taptapConf.GetHost(), uri)
	} else {
		fullUrl = fmt.Sprintf("https://%s:%d%s", taptapConf.GetHost(), port, uri)
	}

	query, err := httpClientManager.CreateQuery(ctx, "GET", fullUrl, nil)
	if err != nil {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("failed to create http query: %w", err),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	query.GetHttpRequest().Header.Add("Authorization", authorization)
	rpcResult := httpClientManager.StartQuery(ctx, query)
	if rpcResult.IsError() {
		return nil, nil, rpcResult
	}

	userProfile, rpcErr, err := ParseTapTapBodyMessage[TapTapUserProfile](query.GetHttpResponseBody())
	if err != nil {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("failed to parse taptap response: %w", err),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	if rpcErr == nil {
		return op_types.MakeUserBasicProfile(userProfile.Name, userProfile.Avatar, userProfile.OpenID, userProfile.UnionID), nil, cd.CreateRpcResultOk()
	}

	if rpcErr.ErrorMessage != "insufficient_scope" {
		return nil,
			op_types.MakeOpenPlatformRpcError(rpcErr.Code, rpcErr.ErrorMessage, rpcErr.ErrorDescription),
			cd.CreateRpcResultOk()
	}

	// Fallback拉取基础信息
	uri = fmt.Sprintf("%s?client_id=%s", taptapApiPathGetBasicInfo, taptapConf.GetClientId())
	nonce = randomString(platformNonceLength)
	macToken = hmacSha1(buildSigningString(strconv.FormatInt(nowSec, 10), nonce,
		"GET", uri, taptapConf.GetHost(), strconv.FormatInt(int64(port), 10)), platformPrivateData.Token.MacKey)
	authorization = fmt.Sprintf("MAC id=\"%s\",ts=\"%d\",nonce=\"%s\",mac=\"%s\"",
		platformPrivateData.Token.Kid, nowSec, nonce, macToken)

	if port == int32(443) {
		fullUrl = fmt.Sprintf("https://%s%s", taptapConf.GetHost(), uri)
	} else if port == int32(80) {
		fullUrl = fmt.Sprintf("http://%s%s", taptapConf.GetHost(), uri)
	} else {
		fullUrl = fmt.Sprintf("https://%s:%d%s", taptapConf.GetHost(), port, uri)
	}

	query, err = httpClientManager.CreateQuery(ctx, "GET", fullUrl, nil)
	if err != nil {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("failed to create http query: %w", err),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	query.GetHttpRequest().Header.Add("Authorization", authorization)
	rpcResult = httpClientManager.StartQuery(ctx, query)
	if rpcResult.IsError() {
		return nil, nil, rpcResult
	}

	userProfile, rpcErr, err = ParseTapTapBodyMessage[TapTapUserProfile](query.GetHttpResponseBody())
	if err != nil {
		return nil, nil, cd.CreateRpcResultError(
			fmt.Errorf("failed to parse taptap response: %w", err),
			public_protocol_pbdesc.EnErrorCode_EN_ERR_SYSTEM)
	}

	if rpcErr != nil {
		return nil,
			op_types.MakeOpenPlatformRpcError(rpcErr.Code, rpcErr.ErrorMessage, rpcErr.ErrorDescription),
			cd.CreateRpcResultOk()
	}

	return op_types.MakeUserBasicProfile(userProfile.Name, userProfile.Avatar, userProfile.OpenID, userProfile.UnionID), nil, cd.CreateRpcResultOk()
}
