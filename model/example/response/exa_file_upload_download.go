package response

import "github.com/tedwangl/go-util/pkg/model/example"


type ExaFileResponse struct {
	File example.ExaFileUploadAndDownload `json:"file"`
}
