package system

import (
	"github.com/tedwangl/go-util/pkg/base"
)

type JwtBlacklist struct {
	global.GVA_MODEL
	Jwt string `gorm:"type:text;comment:jwt"`
}
