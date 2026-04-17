// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type tapTapBodyMessage struct {
	Success *bool           `json:"success"`
	Data    json.RawMessage `json:"data"`
	Now     uint64          `json:"now"`
}

type TapTapErrorMessage struct {
	Code             int32  `json:"code"`
	ErrorMessage     string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// Response 公共结构
// @see https://developer.taptap.cn/docs/sdk/common/api/#response-%E9%80%9A%E7%94%A8%E7%BB%93%E6%9E%84
//
//	{
//	  "success": true，
//	  "data": {},
//	  "now": 1767065075
//	}
func ParseTapTapBodyMessage[T any](jsonData []byte) (*T, *TapTapErrorMessage, error) {
	retBody := &tapTapBodyMessage{}
	if err := json.Unmarshal(jsonData, retBody); err != nil {
		return nil, nil, err
	}

	if retBody.Success == nil {
		return nil, nil, fmt.Errorf("invalid taptap response: missing success field")
	}

	if len(retBody.Data) == 0 {
		return nil, nil, fmt.Errorf("invalid taptap response: missing data field")
	}

	if bytes.Equal(bytes.TrimSpace(retBody.Data), []byte("null")) {
		return nil, nil, fmt.Errorf("invalid taptap response: data is null")
	}

	if !*retBody.Success {
		retErr := &TapTapErrorMessage{}
		if err := json.Unmarshal(retBody.Data, retErr); err != nil {
			return nil, nil, err
		}

		return nil, retErr, nil
	}

	ret := new(T)
	if err := json.Unmarshal(retBody.Data, ret); err != nil {
		return nil, nil, err
	}

	return ret, nil, nil
}
