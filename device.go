package noblechild

import (
	"errors"

	log "github.com/Sirupsen/logrus"
	gatt "github.com/shirou/gatt"
)

var (
	NotImplementedError = errors.New("not implemented")
)

type device struct {
	stateChanged           func(d gatt.Device, s gatt.State)
	centralConnected       func(c gatt.Central)
	centralDisconnected    func(c gatt.Central)
	peripheralDiscovered   func(p gatt.Peripheral, a *gatt.Advertisement, rssi int)
	peripheralConnected    func(p gatt.Peripheral, err error)
	peripheralDisconnected func(p gatt.Peripheral, err error)

	state  gatt.State
	hci    *HCI_BLE
	l2caps map[string]*L2CAP_BLE // peripheralUuid -> L2CAP_BLE

	nobleModules NobleModule
}

func NewDevice(opts ...gatt.Option) (gatt.Device, error) {
	d := device{
		l2caps: map[string]*L2CAP_BLE{},
	}

	noble, err := FindNobleModule()
	if err != nil {
		return &d, err
	}
	d.nobleModules = noble

	hci, err := NewHCI(&d, noble.HCIPath)
	if err != nil {
		return &d, err
	}
	d.hci = hci

	return &d, nil
}

func (d *device) Init(f func(gatt.Device, gatt.State)) error {
	err := d.hci.Init()
	if err != nil {
		return err
	}

	d.state = gatt.StatePoweredOn
	d.stateChanged = f

	return nil
}

func (d *device) Stop() error {
	return nil
}

func (d *device) AddService(s *gatt.Service) error {
	return NotImplementedError
}

func (d *device) RemoveAllServices() error {
	return NotImplementedError
}

func (d *device) SetServices(s []*gatt.Service) error {
	return NotImplementedError
}

func (d *device) AdvertiseNameAndServices(name string, uu []gatt.UUID) error {
	return NotImplementedError
}

func (d *device) AdvertiseIBeaconData(b []byte) error {
	return NotImplementedError
}

func (d *device) AdvertiseIBeacon(u gatt.UUID, major, minor uint16, pwr int8) error {
	return NotImplementedError
}

func (d *device) StopAdvertising() error {
	return NotImplementedError
}

func (d *device) Scan(ss []gatt.UUID, dup bool) {
	err := d.hci.StartScan()
	if err != nil {
		log.Printf("start scan failed: %s", err)
	}
}

func (d *device) StopScanning() {
	err := d.hci.StopScan()
	if err != nil {
		log.Printf("stop scan failed: %s", err)
	}
}

func (d *device) Connect(p gatt.Peripheral) {
	address := p.ID()
	l2cap, ok := d.l2caps[address]
	if ok {
		log.Printf("already connected perfipheral: %s", address)
		return
	}
	l2cap, err := NewL2CAP(d, d.nobleModules.L2CAPPath)
	if err != nil {
		log.Printf("l2cap start failed: %s", address)
		return
	}
	addressType := "public"
	err = l2cap.Init(address, addressType)
	if err != nil {
		log.Printf("l2cap init failed: %s", address)
		return
	}
	d.l2caps[address] = l2cap
}

func (d *device) CancelConnection(p gatt.Peripheral) {
	address := p.ID()
	l2cap, ok := d.l2caps[address]
	if !ok {
		log.Printf("no such perfipheral id connected: %s", address)
		return
	}
	l2cap.Close()
}

/*
func (d *device) SendHCIRawCommand(c cmd.CmdParam) ([]byte, error) {
	return []byte{}, NotImplementedError
}
*/

// Handle registers the specified handlers.
func (d *device) Handle(hh ...gatt.Handler) {
	for _, h := range hh {
		h(d)
	}
}
func (d *device) Option(opts ...gatt.Option) error {
	var err error
	for _, opt := range opts {
		err = opt(d)
	}
	return err
}

func CentralConnected(f func(gatt.Central)) gatt.Handler {
	return func(d gatt.Device) { d.(*device).centralConnected = f }
}
func CentralDisconnected(f func(gatt.Central)) gatt.Handler {
	return func(d gatt.Device) { d.(*device).centralDisconnected = f }
}
func PeripheralDiscovered(f func(gatt.Peripheral, *gatt.Advertisement, int)) gatt.Handler {
	return func(d gatt.Device) { d.(*device).peripheralDiscovered = f }
}
func PeripheralConnected(f func(gatt.Peripheral, error)) gatt.Handler {
	return func(d gatt.Device) { d.(*device).peripheralConnected = f }
}
func PeripheralDisconnected(f func(gatt.Peripheral, error)) gatt.Handler {
	return func(d gatt.Device) { d.(*device).peripheralDisconnected = f }
}
