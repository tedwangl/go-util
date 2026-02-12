package request

import (
	"github.com/tedwangl/go-util/model/common/request"
	"github.com/tedwangl/go-util/model/system"
)

type SysOperationRecordSearch struct {
	system.SysOperationRecord
	request.PageInfo
}
