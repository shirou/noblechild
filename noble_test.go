package noblechild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_AddrToCommaAddr(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("", AddrToCommaAddr(""))
	assert.Equal("aa:bb:cc", AddrToCommaAddr("aa:bb:cc"))
	assert.Equal("aa:bb:cc", AddrToCommaAddr("aabbcc"))
}

func Test_ByteToString(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("020001", ByteToString([]byte{0x02, 0x00, 0x01}))
	b := []byte{0x10, 0x01, 0x00, 0xff, 0xff, 0x00, 0x28}
	data := ByteToString(b)
	assert.Equal("100100ffff0028", data)
}

func Test_StringToBytes(t *testing.T) {
	assert := assert.New(t)

	b1, err := StringToByte("020001")
	assert.Nil(err)
	assert.Equal([]byte{0x02, 0x00, 0x01}, b1)

	b2, err := StringToByte("100100ffff0028")
	assert.Nil(err)
	assert.Equal([]byte{0x10, 0x01, 0x00, 0xff, 0xff, 0x00, 0x28}, b2)
}
