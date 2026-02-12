package request

import (
	"time"

	"github.com/tedwangl/go-util/model/common/request"
)

type SysErrorSearch struct{
    CreatedAtRange []time.Time `json:"createdAtRange" form:"createdAtRange[]"`
      Form  *string `json:"form" form:"form"`
      Info  *string `json:"info" form:"info"`
    request.PageInfo
}
