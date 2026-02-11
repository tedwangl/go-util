package uuid

import "github.com/google/uuid"

// NewUuid returns an uuid string.
func NewUuid() string {
	return uuid.New().String()
}
