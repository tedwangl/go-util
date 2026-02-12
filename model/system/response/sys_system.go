package response

import "github.com/tedwangl/go-util/config"

type SysConfigResponse struct {
	Config config.Server `json:"config"`
}
