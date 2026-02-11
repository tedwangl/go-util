package uuid

import "github.com/google/uuid"

// NewUUID 生成UUID v4
func NewUUID() string {
	return uuid.New().String()
}


// NewUUIDWithoutDash 生成不带横线的UUID
func NewUUIDWithoutDash() string {
	id := uuid.New()
	return id.String()[:8] + id.String()[9:13] + id.String()[14:18] + id.String()[19:23] + id.String()[24:]
}

// ParseUUID 解析UUID
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// IsValidUUID 验证UUID格式
func IsValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}
