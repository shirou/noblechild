package noblechild

import (
	"testing"

	"github.com/paypal/gatt"
	"github.com/stretchr/testify/assert"
)

func Test_parseEvent(t *testing.T) {
	assert := assert.New(t)

	{
		e, err := parseEvent("A0:14:3D:47:25:02,public,02010611061bc5d5a50200baafe211a88400fae13902ff01,-90")
		assert.Nil(err)
		assert.Equal("a0143d472502", e.Address)
		assert.Equal("public", e.AddressType)
		assert.Equal(-90, e.RSSI)
		assert.Equal(1, len(e.Advertisement.Services))
		assert.Equal(gatt.MustParseUUID("39e1fa0084a811e2afba0002a5d5c51b"), e.Advertisement.Services[0])
	}

	{
		e, err := parseEvent("A0:14:3D:47:25:02,public,1209466c6f77657220706f776572203235303205120a006400020a00,-83")
		assert.Nil(err)
		assert.Equal("public", e.AddressType)
		assert.Equal(-83, e.RSSI)
		assert.Equal(0, len(e.Advertisement.Services))
	}

	{
		e, err := parseEvent("20:73:77:65:43:21,public,02010509ff0f000202f202203a100957494345442053656e7365204b6974,-77")
		assert.Nil(err)
		assert.Equal("public", e.AddressType)
		assert.Equal(-77, e.RSSI)
		assert.Equal(0, len(e.Advertisement.Services))
	}
	{
		e, err := parseEvent("event 20:73:77:65:43:21,public,020a04,-81")
		assert.Nil(err)
		assert.Equal("public", e.AddressType)
		assert.Equal(-81, e.RSSI)
		assert.Equal(0, len(e.Advertisement.Services))
	}
	{
		e, err := parseEvent("event B4:99:4C:64:A6:E0,public,0201060302e0ff09ff5946010004390020,-53")
		assert.Nil(err)
		assert.Equal("public", e.AddressType)
		assert.Equal(-53, e.RSSI)
		assert.Equal(0, len(e.Advertisement.Services))
	}
	{
		e, err := parseEvent("event B4:99:4C:64:A6:E0,public,08094e65787475726e051210009001,-53")
		assert.Nil(err)
		assert.Equal("public", e.AddressType)
		assert.Equal(-53, e.RSSI)
		assert.Equal(0, len(e.Advertisement.Services))
	}

}
