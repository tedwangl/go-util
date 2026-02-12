package response

import (
	"github.com/tedwangl/go-util/model/system/request"
)

type PolicyPathResponse struct {
	Paths []request.CasbinInfo `json:"paths"`
}
