package logx

import (
	"github.com/tedwangl/go-util/pkg/base/color"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithColor(t *testing.T) {
	old := atomic.SwapUint32(&encoding, plainEncodingType)
	defer atomic.StoreUint32(&encoding, old)

	output := WithColor("hello", color.BgBlue)
	assert.Equal(t, "hello", output)

	atomic.StoreUint32(&encoding, jsonEncodingType)
	output = WithColor("hello", color.BgBlue)
	assert.Equal(t, "hello", output)
}

func TestWithColorPadding(t *testing.T) {
	old := atomic.SwapUint32(&encoding, plainEncodingType)
	defer atomic.StoreUint32(&encoding, old)

	output := WithColorPadding("hello", color.BgBlue)
	assert.Equal(t, " hello ", output)

	atomic.StoreUint32(&encoding, jsonEncodingType)
	output = WithColorPadding("hello", color.BgBlue)
	assert.Equal(t, "hello", output)
}
