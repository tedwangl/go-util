package restyx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/go-resty/resty/v2"
)

// Signer 请求签名器接口
type Signer interface {
	Sign(req *resty.Request) error
}

// HMACSignerConfig HMAC 签名配置
type HMACSignerConfig struct {
	SecretKey     string   // 密钥
	SignHeader    string   // 签名头名称，默认 "X-Signature"
	TimestampKey  string   // 时间戳参数名，默认 "timestamp"
	IncludeBody   bool     // 是否包含请求体
	IncludeParams []string // 包含的参数（为空则包含所有）
}

// HMACSigner HMAC-SHA256 签名器
type HMACSigner struct {
	config HMACSignerConfig
}

// NewHMACSigner 创建 HMAC 签名器
func NewHMACSigner(config HMACSignerConfig) *HMACSigner {
	if config.SignHeader == "" {
		config.SignHeader = "X-Signature"
	}
	if config.TimestampKey == "" {
		config.TimestampKey = "timestamp"
	}
	return &HMACSigner{config: config}
}

// Sign 签名请求
func (s *HMACSigner) Sign(req *resty.Request) error {
	// 构建签名字符串
	var parts []string

	// 添加查询参数
	if req.QueryParam != nil {
		keys := make([]string, 0, len(req.QueryParam))
		for k := range req.QueryParam {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			if len(s.config.IncludeParams) == 0 || contains(s.config.IncludeParams, k) {
				parts = append(parts, fmt.Sprintf("%s=%s", k, req.QueryParam.Get(k)))
			}
		}
	}

	// 添加请求体
	if s.config.IncludeBody && req.Body != nil {
		parts = append(parts, fmt.Sprintf("body=%v", req.Body))
	}

	// 生成签名
	signStr := strings.Join(parts, "&")
	h := hmac.New(sha256.New, []byte(s.config.SecretKey))
	h.Write([]byte(signStr))
	signature := hex.EncodeToString(h.Sum(nil))

	// 设置签名头
	req.SetHeader(s.config.SignHeader, signature)

	return nil
}

// contains 检查字符串是否在切片中
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// WithSigner 设置签名器
func WithSigner(signer Signer) RequestOption {
	return func(r *resty.Request) {
		signer.Sign(r)
	}
}
