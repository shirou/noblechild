package noblechild

import "github.com/paypal/gatt"

type peripheral struct {
	NameChanged func(*peripheral)

	// ServicedModified is called when one or more service of a peripheral have changed.
	// A list of invalid service is provided in the parameter.
	ServicesModified func(*peripheral, []*gatt.Service)

	svcs map[string]*gatt.Service

	d           *device
	l2cap       *L2CAP_BLE
	Address     string
	AddressType string
	LocalName   string
}

func NewPerfipheral(d *device, l2cap *L2CAP_BLE, address string) peripheral {
	p := peripheral{
		d:       d,
		l2cap:   l2cap,
		Address: address,
	}

	return p
}

func (p *peripheral) Device() gatt.Device { return p.d }
func (p *peripheral) ID() string          { return p.Address }
func (p *peripheral) Name() string        { return p.LocalName }
func (p *peripheral) Services() []*gatt.Service {
	ret := make([]*gatt.Service, len(p.svcs))
	for _, svc := range p.svcs {
		ret = append(ret, svc)
	}

	return ret
}

func finish(op byte, h uint16, b []byte) bool {
	return false
}

func (p *peripheral) DiscoverServices(s []gatt.UUID) ([]*gatt.Service, error) {

	return p.l2cap.DiscoverServices(s)
}

func (p *peripheral) DiscoverIncludedServices(ss []gatt.UUID, s *gatt.Service) ([]*gatt.Service, error) {
	return nil, NotImplementedError
}

func (p *peripheral) DiscoverCharacteristics(cs []gatt.UUID, s *gatt.Service) ([]*gatt.Characteristic, error) {

	return nil, nil
}

func (p *peripheral) DiscoverDescriptors(ds []gatt.UUID, c *gatt.Characteristic) ([]*gatt.Descriptor, error) {
	return nil, NotImplementedError
}

func (p *peripheral) ReadCharacteristic(c *gatt.Characteristic) ([]byte, error) {
	return []byte{}, NotImplementedError
}

func (p *peripheral) WriteCharacteristic(c *gatt.Characteristic, value []byte, noRsp bool) error {
	return NotImplementedError
}

func (p *peripheral) ReadDescriptor(d *gatt.Descriptor) ([]byte, error) {
	return []byte{}, NotImplementedError
}

func (p *peripheral) WriteDescriptor(d *gatt.Descriptor, value []byte) error {
	return NotImplementedError
}

func (p *peripheral) setNotifyValue(c *gatt.Characteristic, flag uint16,
	f func(*gatt.Characteristic, []byte, error)) error {
	return NotImplementedError
}

func (p *peripheral) SetNotifyValue(c *gatt.Characteristic,
	f func(*gatt.Characteristic, []byte, error)) error {
	return NotImplementedError
}

func (p *peripheral) SetIndicateValue(c *gatt.Characteristic,
	f func(*gatt.Characteristic, []byte, error)) error {
	return NotImplementedError
}

func (p *peripheral) ReadRSSI() int {
	return -1
}

func searchService(ss []*gatt.Service, start, end uint16) *gatt.Service {
	return nil
}
func (p *peripheral) loop() {
}
