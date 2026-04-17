// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform_type

type openPlatformRpcErrorImpl struct {
	ErrorCode        int32  // 平台错误码
	ErrorMessage     string // 平台错误信息
	ErrorDescription string // 平台错误描述
}

type OpenPlatformRpcError interface {
	GetErrorCode() int32         // 平台错误码
	GetErrorMessage() string     // 平台错误信息
	GetErrorDescription() string // 平台错误描述
}

func (e *openPlatformRpcErrorImpl) GetErrorCode() int32 {
	if e == nil {
		return 0
	}
	return e.ErrorCode
}

func (e *openPlatformRpcErrorImpl) GetErrorMessage() string {
	if e == nil {
		return ""
	}
	return e.ErrorMessage
}

func (e *openPlatformRpcErrorImpl) GetErrorDescription() string {
	if e == nil {
		return ""
	}
	return e.ErrorDescription
}

func MakeOpenPlatformRpcError(errorCode int32, errorMessage string, errorDescription string) OpenPlatformRpcError {
	return &openPlatformRpcErrorImpl{
		ErrorCode:        errorCode,
		ErrorMessage:     errorMessage,
		ErrorDescription: errorDescription,
	}
}
