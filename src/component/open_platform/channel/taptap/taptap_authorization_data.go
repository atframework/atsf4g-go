// Copyright 2026 atframework
// @brief 开放平台管理器

package atframework_component_open_platform

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
)

type TapTapTokenData struct {
	Kid          string `json:"kid"`           // 令牌ID
	TokenType    string `json:"token_type"`    // 令牌类型
	MacKey       string `json:"mac_key"`       // MAC密钥
	MacAlgorithm string `json:"mac_algorithm"` // MAC算法
	Scope        string `json:"scope"`         // 令牌权限范围
}

type TapTapClientTokenData struct {
	Origin string
	Token  TapTapTokenData
}

// buildSigningString 构造待签名字符串
//
// 参数说明：
//   - ts: 时间戳
//   - nonce: 随机数
//   - method: HTTP 方法
//   - uri: 请求路径（含 query string）
//   - host: 请求域名
//   - port: 端口号
func buildSigningString(ts, nonce, method, uri, host, port string) string {
	return ts + "\n" + nonce + "\n" + method + "\n" + uri + "\n" + host + "\n" + port + "\n\n"
}

// hmacSha1 使用 HMAC-SHA1 生成签名，返回 Base64 编码的签名值
//
// 参数说明：
//   - signingString: 待签名字符串
//   - key: MAC 密钥
func hmacSha1(signingString, key string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(signingString))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// randomString 生成随机字符串
//
// 参数说明：
//   - length: 随机字符串的长度
func randomString(length int) string {
	const chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = chars[b[i]%byte(len(chars))]
	}
	return string(b)
}
