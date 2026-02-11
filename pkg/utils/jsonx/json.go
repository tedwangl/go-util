package jsonx

import (
	"github.com/bytedance/sonic"
)

var (
	Marshal       = sonic.Marshal
	Unmarshal     = sonic.Unmarshal
	MarshalString = sonic.MarshalString
)

func MarshalToString(v any) (string, error) {
	return sonic.MarshalString(v)
}

func UnmarshalFromString(str string, v any) error {
	return sonic.UnmarshalString(str, v)
}

func UnmarshalFromBytes(data []byte, v any) error {
	return sonic.Unmarshal(data, v)
}

func MarshalToBytes(v any) ([]byte, error) {
	return sonic.Marshal(v)
}
