package response

import (
	"github.com/tedwangl/go-util/model/example"
)

type FilePathResponse struct {
	FilePath string `json:"filePath"`
}

type FileResponse struct {
	File example.ExaFile `json:"file"`
}
