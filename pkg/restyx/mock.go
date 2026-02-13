package restyx

import (
	"net/http"
	"net/http/httptest"
)

// MockServer Mock 服务器（用于测试）
type MockServer struct {
	server  *httptest.Server
	handler http.HandlerFunc
}

// NewMockServer 创建 Mock 服务器
func NewMockServer(handler http.HandlerFunc) *MockServer {
	server := httptest.NewServer(handler)
	return &MockServer{
		server:  server,
		handler: handler,
	}
}

// URL 返回 Mock 服务器地址
func (m *MockServer) URL() string {
	return m.server.URL
}

// Close 关闭 Mock 服务器
func (m *MockServer) Close() {
	m.server.Close()
}

// MockResponse Mock 响应构建器
type MockResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

// NewMockResponse 创建 Mock 响应
func NewMockResponse(statusCode int, body string) *MockResponse {
	return &MockResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    make(map[string]string),
	}
}

// WithHeader 添加响应头
func (m *MockResponse) WithHeader(key, value string) *MockResponse {
	m.Headers[key] = value
	return m
}

// Handler 返回 HTTP 处理器
func (m *MockResponse) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for k, v := range m.Headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(m.StatusCode)
		w.Write([]byte(m.Body))
	}
}
