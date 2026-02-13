package restyx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

// Logger 日志接口
type (
	Logger interface {
		Debug(msg string, fields ...any)
		Info(msg string, fields ...any)
		Warn(msg string, fields ...any)
		Error(msg string, fields ...any)
	}

	// RequestInterceptor 请求拦截器
	RequestInterceptor func(*resty.Request) error

	// ResponseInterceptor 响应拦截器
	ResponseInterceptor func(*resty.Response) error

	// noopLogger 空日志实现
	noopLogger struct{}

	// Client resty 客户端封装
	Client struct {
		client               *resty.Client
		logger               Logger
		slowRequestThreshold time.Duration
		returnErrorOnNon2xx  bool
		reqInterceptors      []RequestInterceptor
		respInterceptors     []ResponseInterceptor
	}

	// Response 响应封装
	Response struct {
		StatusCode int           // HTTP 状态码
		Body       []byte        // 响应体
		Headers    http.Header   // 响应头
		Time       time.Duration // 请求耗时
	}
	// RequestOption 请求选项
	RequestOption func(*resty.Request)
	// Config 客户端配置
	Config struct {
		BaseURL              string            // 基础 URL
		Timeout              time.Duration     // 请求超时
		RetryCount           int               // 重试次数
		RetryWaitTime        time.Duration     // 重试等待时间
		RetryMaxWaitTime     time.Duration     // 最大重试等待时间
		DefaultHeaders       map[string]string // 默认请求头
		SlowRequestThreshold time.Duration     // 慢请求阈值
		ReturnErrorOnNon2xx  bool              // 非 2xx 是否返回 error
		MaxIdleConns         int               // 最大空闲连接数
		MaxConnsPerHost      int               // 每个 host 最大连接数
		IdleConnTimeout      time.Duration     // 空闲连接超时
		ProxyURL             string            // 代理地址
		TLSClientCert        string            // TLS 客户端证书路径
		TLSClientKey         string            // TLS 客户端密钥路径
		TLSCACert            string            // TLS CA 证书路径
		InsecureSkipVerify   bool              // 跳过 TLS 验证
		EnableCookieJar      bool              // 启用 Cookie 管理
	}
)

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		Timeout:          30 * time.Second,
		RetryCount:       3,
		RetryWaitTime:    100 * time.Millisecond,
		RetryMaxWaitTime: 2 * time.Second,
		DefaultHeaders: map[string]string{
			"Content-Type": "application/json",
			"User-Agent":   "RestyX/1.0",
		},
		SlowRequestThreshold: 1 * time.Second,
		ReturnErrorOnNon2xx:  false,
		MaxIdleConns:         100,
		MaxConnsPerHost:      100,
		IdleConnTimeout:      90 * time.Second,
	}
}

// New 创建客户端
func New(config Config, logger Logger) *Client {
	if logger == nil {
		logger = &noopLogger{}
	}

	client := resty.New()
	client.SetTimeout(config.Timeout)
	client.SetRetryCount(config.RetryCount)
	client.SetRetryWaitTime(config.RetryWaitTime)
	client.SetRetryMaxWaitTime(config.RetryMaxWaitTime)

	// 配置连接池和 TLS
	transport := &http.Transport{
		MaxIdleConns:        config.MaxIdleConns,
		MaxConnsPerHost:     config.MaxConnsPerHost,
		IdleConnTimeout:     config.IdleConnTimeout,
		MaxIdleConnsPerHost: config.MaxConnsPerHost,
	}

	// 配置代理
	if config.ProxyURL != "" {
		proxyURL, err := url.Parse(config.ProxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// 配置 TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	// 加载客户端证书
	if config.TLSClientCert != "" && config.TLSClientKey != "" {
		cert, err := tls.LoadX509KeyPair(config.TLSClientCert, config.TLSClientKey)
		if err == nil {
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}

	// 加载 CA 证书
	if config.TLSCACert != "" {
		caCert, err := os.ReadFile(config.TLSCACert)
		if err == nil {
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = caCertPool
		}
	}

	transport.TLSClientConfig = tlsConfig
	client.SetTransport(transport)

	// 启用 Cookie Jar
	if config.EnableCookieJar {
		client.SetCookieJar(nil) // 使用默认 cookie jar
	}

	if len(config.DefaultHeaders) > 0 {
		for key, value := range config.DefaultHeaders {
			client.SetHeader(key, value)
		}
	}

	if config.BaseURL != "" {
		client.SetBaseURL(config.BaseURL)
	}

	// 重试条件：网络错误或 5xx
	client.AddRetryCondition(func(r *resty.Response, err error) bool {
		if err != nil {
			return true
		}
		return r.StatusCode() >= 500
	})

	return &Client{
		client:               client,
		logger:               logger,
		slowRequestThreshold: config.SlowRequestThreshold,
		returnErrorOnNon2xx:  config.ReturnErrorOnNon2xx,
	}
}

// WithHeader 设置请求头
func WithHeader(key, value string) RequestOption {
	return func(r *resty.Request) {
		r.SetHeader(key, value)
	}
}

// WithHeaders 设置多个请求头
func WithHeaders(headers map[string]string) RequestOption {
	return func(r *resty.Request) {
		for k, v := range headers {
			r.SetHeader(k, v)
		}
	}
}

// WithQueryParam 设置查询参数
func WithQueryParam(key, value string) RequestOption {
	return func(r *resty.Request) {
		r.SetQueryParam(key, value)
	}
}

// WithQueryParams 设置多个查询参数
func WithQueryParams(params map[string]string) RequestOption {
	return func(r *resty.Request) {
		r.SetQueryParams(params)
	}
}

// WithPathParam 设置路径参数
func WithPathParam(key, value string) RequestOption {
	return func(r *resty.Request) {
		r.SetPathParam(key, value)
	}
}

// WithPathParams 设置多个路径参数
func WithPathParams(params map[string]string) RequestOption {
	return func(r *resty.Request) {
		r.SetPathParams(params)
	}
}

// WithBody 设置请求体
func WithBody(body any) RequestOption {
	return func(r *resty.Request) {
		r.SetBody(body)
	}
}

// WithJSON 设置 JSON 请求体
func WithJSON(jsonData any) RequestOption {
	return func(r *resty.Request) {
		r.SetHeader("Content-Type", "application/json")
		r.SetBody(jsonData)
	}
}

// WithForm 设置表单请求体
func WithForm(formData map[string]string) RequestOption {
	return func(r *resty.Request) {
		r.SetHeader("Content-Type", "application/x-www-form-urlencoded")
		r.SetFormData(formData)
	}
}

// WithFile 设置文件上传
func WithFile(paramName, filePath string) RequestOption {
	return func(r *resty.Request) {
		r.SetFile(paramName, filePath)
	}
}

// WithCookie 设置 Cookie
func WithCookie(cookie *http.Cookie) RequestOption {
	return func(r *resty.Request) {
		r.SetCookie(cookie)
	}
}

// WithCookies 设置多个 Cookie
func WithCookies(cookies []*http.Cookie) RequestOption {
	return func(r *resty.Request) {
		r.SetCookies(cookies)
	}
}

// WithBearerToken 设置 Bearer Token
func WithBearerToken(token string) RequestOption {
	return func(r *resty.Request) {
		r.SetAuthToken(token)
	}
}

// WithBasicAuth 设置 Basic Auth
func WithBasicAuth(username, password string) RequestOption {
	return func(r *resty.Request) {
		r.SetBasicAuth(username, password)
	}
}

// WithAuthToken 设置自定义 Token（自定义 header）
func WithAuthToken(headerName, token string) RequestOption {
	return func(r *resty.Request) {
		r.SetHeader(headerName, token)
	}
}

// WithContext 设置请求上下文
func WithContext(ctx context.Context) RequestOption {
	return func(r *resty.Request) {
		r.SetContext(ctx)
	}
}

// IsSuccess 判断响应是否成功 (2xx)
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// UnmarshalJSON 解析 JSON 响应体
func (r *Response) UnmarshalJSON(v any) error {
	if len(r.Body) == 0 {
		return nil
	}
	return json.Unmarshal(r.Body, v)
}

// String 返回响应体字符串
func (r *Response) String() string {
	return string(r.Body)
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(method, url string, options ...RequestOption) (*Response, error) {
	startTime := time.Now()

	req := c.client.R()
	for _, option := range options {
		option(req)
	}

	// 执行请求拦截器
	for _, interceptor := range c.reqInterceptors {
		if err := interceptor(req); err != nil {
			return nil, fmt.Errorf("request interceptor failed: %w", err)
		}
	}

	ctx := req.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	reqID := ctx.Value("request_id")

	var resp *resty.Response
	var err error

	switch strings.ToUpper(method) {
	case http.MethodGet:
		resp, err = req.Get(url)
	case http.MethodPost:
		resp, err = req.Post(url)
	case http.MethodPut:
		resp, err = req.Put(url)
	case http.MethodDelete:
		resp, err = req.Delete(url)
	case http.MethodPatch:
		resp, err = req.Patch(url)
	case http.MethodHead:
		resp, err = req.Head(url)
	case http.MethodOptions:
		resp, err = req.Options(url)
	default:
		return nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}

	duration := time.Since(startTime)

	wrappedResp := &Response{
		StatusCode: resp.StatusCode(),
		Body:       resp.Body(),
		Headers:    resp.Header(),
		Time:       duration,
	}

	// 执行响应拦截器
	for _, interceptor := range c.respInterceptors {
		if err := interceptor(resp); err != nil {
			return wrappedResp, fmt.Errorf("response interceptor failed: %w", err)
		}
	}

	// 日志记录
	c.logRequest(method, url, reqID, wrappedResp, duration)

	if err != nil {
		return wrappedResp, fmt.Errorf("HTTP request failed: %w", err)
	}

	// 根据配置决定是否返回 error
	if c.returnErrorOnNon2xx && !wrappedResp.IsSuccess() {
		return wrappedResp, fmt.Errorf("HTTP request failed with status code: %d", wrappedResp.StatusCode)
	}

	return wrappedResp, nil
}

// Get 发送 GET 请求
func (c *Client) Get(url string, options ...RequestOption) (*Response, error) {
	return c.doRequest(http.MethodGet, url, options...)
}

// Post 发送 POST 请求
func (c *Client) Post(url string, options ...RequestOption) (*Response, error) {
	return c.doRequest(http.MethodPost, url, options...)
}

// Put 发送 PUT 请求
func (c *Client) Put(url string, options ...RequestOption) (*Response, error) {
	return c.doRequest(http.MethodPut, url, options...)
}

// Delete 发送 DELETE 请求
func (c *Client) Delete(url string, options ...RequestOption) (*Response, error) {
	return c.doRequest(http.MethodDelete, url, options...)
}

// Patch 发送 PATCH 请求
func (c *Client) Patch(url string, options ...RequestOption) (*Response, error) {
	return c.doRequest(http.MethodPatch, url, options...)
}

// Head 发送 HEAD 请求
func (c *Client) Head(url string, options ...RequestOption) (*Response, error) {
	return c.doRequest(http.MethodHead, url, options...)
}

// Options 发送 OPTIONS 请求
func (c *Client) Options(url string, options ...RequestOption) (*Response, error) {
	return c.doRequest(http.MethodOptions, url, options...)
}

// DownloadFile 下载文件
func (c *Client) DownloadFile(url, filePath string, options ...RequestOption) error {
	startTime := time.Now()

	req := c.client.R()
	for _, option := range options {
		option(req)
	}

	rsp, err := req.SetOutput(filePath).Get(url)
	duration := time.Since(startTime)

	logFields := []any{
		"method", "GET",
		"url", url,
		"file_path", filePath,
		"status_code", rsp.StatusCode(),
		"duration_ms", duration.Milliseconds(),
	}

	if err != nil {
		c.logger.Error("File download failed", append(logFields, "error", err)...)
		return fmt.Errorf("file download failed: %w", err)
	}

	if rsp.StatusCode() < 200 || rsp.StatusCode() >= 300 {
		c.logger.Error("File download failed with status code", logFields...)
		return fmt.Errorf("file download failed with status code: %d", rsp.StatusCode())
	}

	c.logger.Info("File downloaded successfully", logFields...)
	return nil
}

// UploadFile 上传文件
func (c *Client) UploadFile(url, paramName, filePath string, options ...RequestOption) (*Response, error) {
	options = append(options, WithFile(paramName, filePath))
	return c.Post(url, options...)
}

// Stream 流式处理响应
func (c *Client) Stream(method, url string, callback func(io.Reader) error, options ...RequestOption) error {
	startTime := time.Now()

	req := c.client.R()
	for _, option := range options {
		option(req)
	}

	req.SetDoNotParseResponse(true)

	var resp *resty.Response
	var err error

	switch strings.ToUpper(method) {
	case http.MethodGet:
		resp, err = req.Get(url)
	case http.MethodPost:
		resp, err = req.Post(url)
	default:
		return fmt.Errorf("unsupported streaming HTTP method: %s", method)
	}

	duration := time.Since(startTime)

	logFields := []any{
		"method", method,
		"url", url,
		"status_code", resp.StatusCode(),
		"duration_ms", duration.Milliseconds(),
	}

	if err != nil {
		c.logger.Error("Streaming request failed", append(logFields, "error", err)...)
		return fmt.Errorf("streaming request failed: %w", err)
	}

	defer resp.RawResponse.Body.Close()

	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		err = callback(resp.RawResponse.Body)
	} else {
		body, _ := io.ReadAll(resp.RawResponse.Body)
		c.logger.Error("Streaming request failed with status code",
			append(logFields, "response_body", string(body))...)
		err = fmt.Errorf("streaming request failed with status code: %d", resp.StatusCode())
	}

	return err
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := c.Get(url, WithContext(ctx))
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if !resp.IsSuccess() {
		return fmt.Errorf("health check failed with status code: %d", resp.StatusCode)
	}

	return nil
}

// NewRequest 创建新请求对象
func (c *Client) NewRequest() *resty.Request {
	return c.client.R()
}

// GetRawClient 获取原始 resty 客户端
func (c *Client) GetRawClient() *resty.Client {
	return c.client
}

// SetCookies 设置客户端级别的 Cookie
func (c *Client) SetCookies(cookies []*http.Cookie) {
	c.client.SetCookies(cookies)
}

// GetCookies 获取指定 URL 的 Cookie
func (c *Client) GetCookies(url string) []*http.Cookie {
	u, err := urlParse(url)
	if err != nil {
		return nil
	}
	return c.client.GetClient().Jar.Cookies(u)
}

// ClearCookies 清除所有 Cookie
func (c *Client) ClearCookies() {
	c.client.SetCookieJar(nil)
}

// SetAuthToken 设置客户端级别的 Bearer Token
func (c *Client) SetAuthToken(token string) {
	c.client.SetAuthToken(token)
}

// SetBasicAuth 设置客户端级别的 Basic Auth
func (c *Client) SetBasicAuth(username, password string) {
	c.client.SetBasicAuth(username, password)
}

// urlParse 解析 URL
func urlParse(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}

// AddRequestInterceptor 添加请求拦截器
func (c *Client) AddRequestInterceptor(interceptor RequestInterceptor) {
	c.reqInterceptors = append(c.reqInterceptors, interceptor)
}

// AddResponseInterceptor 添加响应拦截器
func (c *Client) AddResponseInterceptor(interceptor ResponseInterceptor) {
	c.respInterceptors = append(c.respInterceptors, interceptor)
}

// BatchRequest 批量请求
type BatchRequest struct {
	Method  string
	URL     string
	Options []RequestOption
}

// BatchResponse 批量响应
type BatchResponse struct {
	Index    int
	Response *Response
	Error    error
}

// Batch 批量执行请求（带并发控制，流式返回）
func (c *Client) Batch(ctx context.Context, requests []BatchRequest, concurrency int) <-chan BatchResponse {
	if concurrency <= 0 {
		concurrency = 10 // 默认并发数
	}

	resultChan := make(chan BatchResponse, len(requests))
	sem := make(chan struct{}, concurrency) // 信号量控制并发
	var wg sync.WaitGroup

	for i, req := range requests {
		wg.Add(1)
		go func(idx int, r BatchRequest) {
			defer wg.Done()

			// 获取信号量
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				resultChan <- BatchResponse{
					Index: idx,
					Error: ctx.Err(),
				}
				return
			}

			// 执行请求
			resp, err := c.doRequest(r.Method, r.URL, r.Options...)
			resultChan <- BatchResponse{
				Index:    idx,
				Response: resp,
				Error:    err,
			}
		}(i, req)
	}

	// 等待所有请求完成后关闭 channel
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	return resultChan
}

// logRequest 记录请求日志
func (c *Client) logRequest(method, url string, reqID any, resp *Response, duration time.Duration) {
	fields := []any{
		"method", method,
		"url", url,
		"status_code", resp.StatusCode,
		"duration_ms", duration.Milliseconds(),
	}

	if reqID != nil {
		fields = append(fields, "request_id", fmt.Sprintf("%v", reqID))
	}

	if duration > c.slowRequestThreshold {
		c.logger.Warn("Slow HTTP request", fields...)
	} else if resp.StatusCode >= 400 {
		fields = append(fields, "response_body", string(resp.Body))
		c.logger.Error("HTTP request failed", fields...)
	} else {
		c.logger.Debug("HTTP request completed", fields...)
	}
}

func (n *noopLogger) Debug(msg string, fields ...any) {}
func (n *noopLogger) Info(msg string, fields ...any)  {}
func (n *noopLogger) Warn(msg string, fields ...any)  {}
func (n *noopLogger) Error(msg string, fields ...any) {}
