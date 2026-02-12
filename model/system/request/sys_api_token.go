package request

import (
	"github.com/tedwangl/go-util/model/common/request"
	"github.com/tedwangl/go-util/model/system"
)

type SysApiTokenSearch struct {
	system.SysApiToken
	request.PageInfo
	Status *bool `json:"status" form:"status"`
}
