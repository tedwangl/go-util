package request

import "github.com/tedwangl/go-util/pkg/model/common/request"

type ExaAttachmentCategorySearch struct {
	ClassId int `json:"classId" form:"classId"`
	request.PageInfo
}
