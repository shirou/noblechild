package noblechild

// This files is almost same as paypal/gatt/peripheral_linux.go. But parsing UUID and accessing via Setter/Getter is different.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gatt"
)

type peripheral struct {
	d           *device
	l2cap       *L2CAP_BLE
	Address     string
	AddressType string
	LocalName   string

	svcs []*gatt.Service

	sub *subscriber

	mtu uint16
	l2c io.ReadWriteCloser

	reqc  chan message
	quitc chan struct{}
}

func NewPeripheral(d *device, l2cap *L2CAP_BLE, address string) peripheral {
	p := peripheral{
		d:       d,
		l2cap:   l2cap,
		l2c:     l2cap,
		Address: address,
		reqc:    make(chan message),
		quitc:   make(chan struct{}),
	}
	go p.loop()

	return p
}

func (p *peripheral) Device() gatt.Device       { return p.d }
func (p *peripheral) ID() string                { return strings.ToUpper(p.Address) }
func (p *peripheral) Name() string              { return p.LocalName }
func (p *peripheral) Services() []*gatt.Service { return p.svcs }

func finish(op byte, h uint16, b []byte) bool {
	done := b[0] == attOpError && b[1] == op && b[2] == byte(h) && b[3] == byte(h>>8)
	e := attEcode(b[4])
	if e != attEcodeAttrNotFound {
		// log.Printf("unexpected protocol error: %s", e)
		// FIXME: terminate the connection
	}
	return done
}

func (p *peripheral) DiscoverServices(filter []gatt.UUID) ([]*gatt.Service, error) {
	// p.pd.Conn.Write([]byte{0x02, 0x87, 0x00}) // MTU
	done := false

	start := uint16(0x0001)
	for !done {
		op := byte(attOpReadByGroupReq)
		b := make([]byte, 7)
		b[0] = op
		binary.LittleEndian.PutUint16(b[1:3], start)
		binary.LittleEndian.PutUint16(b[3:5], 0xFFFF)
		binary.LittleEndian.PutUint16(b[5:7], 0x2800)

		b = p.sendReq(op, b)
		if finish(op, start, b) {
			break
		}
		log.Debugf("p read:%v", b)
		b = b[1:]
		l, b := int(b[0]), b[1:]
		switch {
		case l == 6 && (len(b)%6 == 0):
		case l == 20 && (len(b)%20 == 0):
		default:
			return nil, ErrInvalidLength
		}

		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			endh := binary.LittleEndian.Uint16(b[2:4])
			u, err := gatt.ParseUUID(fmt.Sprintf("%2x", b[4:l]))
			if err != nil {
				return nil, fmt.Errorf("DiscoverServices parseUUID failed: %2x, %s", b[4:l], err)
			}
			s := gatt.NewService(u)
			s.SetHandle(h)
			s.SetEndHandle(endh)

			if len(filter) == 0 || IncludesUUID(u, filter) {
				p.svcs = append(p.svcs, s)
			}

			b = b[l:]
			done = endh == 0xFFFF
			start = endh + 1
		}
	}
	return p.svcs, nil
}

func (p *peripheral) DiscoverIncludedServices(ss []gatt.UUID, s *gatt.Service) ([]*gatt.Service, error) {
	// TODO
	return nil, nil
}

func (p *peripheral) DiscoverCharacteristics(cs []gatt.UUID, s *gatt.Service) ([]*gatt.Characteristic, error) {
	done := false
	start := s.Handle()
	var prev *gatt.Characteristic
	for !done {
		op := byte(attOpReadByTypeReq)
		b := make([]byte, 7)
		b[0] = op
		binary.LittleEndian.PutUint16(b[1:3], start)
		binary.LittleEndian.PutUint16(b[3:5], s.EndHandle())
		binary.LittleEndian.PutUint16(b[5:7], 0x2803)

		b = p.sendReq(op, b)
		if finish(op, start, b) {
			break
		}
		b = b[1:]

		l, b := int(b[0]), b[1:]
		switch {
		case l == 7 && (len(b)%7 == 0):
		case l == 21 && (len(b)%21 == 0):
		default:
			return nil, ErrInvalidLength
		}

		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			props := gatt.Property(b[2])
			vh := binary.LittleEndian.Uint16(b[3:5])
			u, err := gatt.ParseUUID(fmt.Sprintf("%2x", b[5:l]))
			if err != nil {
				return nil, fmt.Errorf("DiscoverCharacteristics parseUUID failed: %2x, %s", b[5:l], err)
			}
			s := searchService(p.svcs, h, vh)
			if s == nil {
				log.Printf("Can't find service range that contains 0x%04X - 0x%04X", h, vh)
				return nil, fmt.Errorf("Can't find service range that contains 0x%04X - 0x%04X", h, vh)
			}
			c := gatt.NewCharacteristic(u, s, props, h, vh)
			if len(cs) == 0 || IncludesUUID(u, cs) {
				s.SetCharacteristics(append(s.Characteristics(), c))
			}
			b = b[l:]
			done = vh == s.EndHandle()
			start = vh + 1
			if prev != nil {
				prev.SetEndHandle(c.Handle() - 1)
			}
			prev = c
		}
	}
	sc := s.Characteristics()
	if len(sc) > 1 {
		// s.chars[len(s.chars)-1].endh = s.endh
		sc[len(sc)-1].SetEndHandle(s.EndHandle())
	}
	return s.Characteristics(), nil
}

func (p *peripheral) DiscoverDescriptors(ds []gatt.UUID, c *gatt.Characteristic) ([]*gatt.Descriptor, error) {
	// TODO: implement the UUID filters
	done := false
	start := c.VHandle() + 1
	for !done {
		if c.EndHandle() == 0 {
			c.SetEndHandle(c.Service().EndHandle())
		}
		op := byte(attOpFindInfoReq)
		b := make([]byte, 5)
		b[0] = op
		binary.LittleEndian.PutUint16(b[1:3], start)
		binary.LittleEndian.PutUint16(b[3:5], c.EndHandle())

		b = p.sendReq(op, b)
		if finish(attOpFindInfoReq, start, b) {
			break
		}
		b = b[1:]

		var l int
		f, b := int(b[0]), b[1:]
		switch {
		case f == 1 && (len(b)%4 == 0):
			l = 4
		case f == 2 && (len(b)%18 == 0):
			l = 18
		default:
			return nil, ErrInvalidLength
		}

		for len(b) != 0 {
			h := binary.LittleEndian.Uint16(b[:2])
			u, err := gatt.ParseUUID(fmt.Sprintf("%2x", b[2:l]))
			if err != nil {
				return nil, fmt.Errorf("DiscoverDescriptors parseUUID failed: %2x, %s", b[2:l], err)
			}
			d := gatt.NewDescriptor(u, h, c)
			c.SetDescriptors(append(c.Descriptors(), d))
			if u.String() == attrClientCharacteristicConfigUUID.String() {
				c.SetDescriptor(d)
			}
			b = b[l:]
			done = h == c.EndHandle()
			start = h + 1
		}
	}
	return c.Descriptors(), nil
}

func (p *peripheral) ReadCharacteristic(c *gatt.Characteristic) ([]byte, error) {
	b := make([]byte, 3)
	op := byte(attOpReadReq)
	b[0] = op
	binary.LittleEndian.PutUint16(b[1:3], c.VHandle())

	b = p.sendReq(op, b)
	b = b[1:]
	return b, nil
}

func (p *peripheral) WriteCharacteristic(c *gatt.Characteristic, value []byte, noRsp bool) error {
	b := make([]byte, 3+len(value))
	op := byte(attOpWriteReq)
	b[0] = op
	if noRsp {
		b[0] = attOpWriteCmd
	}
	binary.LittleEndian.PutUint16(b[1:3], c.VHandle())
	copy(b[3:], value)

	if noRsp {
		p.sendCmd(op, b)
		return nil
	}
	b = p.sendReq(op, b)
	// TODO: error handling
	b = b[1:]
	return nil
}

func (p *peripheral) ReadDescriptor(d *gatt.Descriptor) ([]byte, error) {
	b := make([]byte, 3)
	op := byte(attOpReadReq)
	b[0] = op
	binary.LittleEndian.PutUint16(b[1:3], d.Handle())

	b = p.sendReq(op, b)
	b = b[1:]
	// TODO: error handling
	return b, nil
}

func (p *peripheral) WriteDescriptor(d *gatt.Descriptor, value []byte) error {
	b := make([]byte, 3+len(value))
	op := byte(attOpWriteReq)
	b[0] = op
	binary.LittleEndian.PutUint16(b[1:3], d.Handle())
	copy(b[3:], value)

	b = p.sendReq(op, b)
	b = b[1:]
	// TODO: error handling
	return nil
}

func (p *peripheral) setNotifyValue(c *gatt.Characteristic, flag uint16,
	f func(*gatt.Characteristic, []byte, error)) error {
	if c.Descriptor == nil {
		return errors.New("no cccd") // FIXME
	}
	ccc := uint16(0)
	if f != nil {
		ccc = flag
		p.sub.subscribe(c.VHandle(), func(b []byte, err error) { f(c, b, err) })
	}
	b := make([]byte, 5)
	op := byte(attOpWriteReq)
	b[0] = op
	binary.LittleEndian.PutUint16(b[1:3], c.Descriptor().Handle())
	binary.LittleEndian.PutUint16(b[3:5], ccc)

	b = p.sendReq(op, b)
	b = b[1:]
	// TODO: error handling
	if f == nil {
		p.sub.unsubscribe(c.VHandle())
	}
	return nil
}

func (p *peripheral) SetNotifyValue(c *gatt.Characteristic,
	f func(*gatt.Characteristic, []byte, error)) error {
	return p.setNotifyValue(c, gattCCCNotifyFlag, f)
}

func (p *peripheral) SetIndicateValue(c *gatt.Characteristic,
	f func(*gatt.Characteristic, []byte, error)) error {
	return p.setNotifyValue(c, gattCCCIndicateFlag, f)
}

func (p *peripheral) ReadRSSI() int {
	// TODO: implement
	return -1
}

func searchService(ss []*gatt.Service, start, end uint16) *gatt.Service {
	for _, s := range ss {
		if s.Handle() < start && s.EndHandle() >= end {
			return s
		}
	}
	return nil
}

// TODO: unifiy the message with OS X pots and refactor
type message struct {
	op   byte
	b    []byte
	rspc chan []byte
}

func (p *peripheral) sendCmd(op byte, b []byte) {
	p.reqc <- message{op: op, b: b}
}

func (p *peripheral) sendReq(op byte, b []byte) []byte {
	m := message{op: op, b: b, rspc: make(chan []byte)}
	p.reqc <- m
	return <-m.rspc
}

func (p *peripheral) loop() {
	// Serialize the request.
	rspc := make(chan []byte)

	// Dequeue request loop
	go func() {
		for {
			select {
			case req := <-p.reqc:
				log.Debugf("peripheral loop reqc: %v", req.b)
				p.l2c.Write(req.b)
				if req.rspc == nil {
					break
				}
				r := <-rspc
				switch reqOp, rspOp := req.b[0], r[0]; {
				case rspOp == attRspFor[reqOp]:
				case rspOp == attOpError && r[1] == reqOp:
				default:
					log.Printf("Request 0x%02x got a mismatched response: 0x%02x", reqOp, rspOp)
					// FIXME: terminate the connection?
				}
				req.rspc <- r
			case <-p.quitc:
				return
			}
		}
	}()

	// L2CAP implementations shall support a minimum MTU size of 48 bytes.
	// The default value is 672 bytes
	buf := make([]byte, 672)

	// Handling response or notification/indication
	for {
		n, err := p.l2c.Read(buf)
		if n == 0 || err != nil {
			close(p.quitc)
			return
		}

		b := make([]byte, n)
		copy(b, buf)

		if (b[0] != attOpHandleNotify) && (b[0] != attOpHandleInd) {
			rspc <- b
			continue
		}

		h := binary.LittleEndian.Uint16(b[1:3])
		f := p.sub.fn(h)
		if f == nil {
			log.Printf("notified by unsubscribed handle")
			// FIXME: terminate the connection?
		} else {
			go f(b[3:], nil)
		}

		if b[0] == attOpHandleInd {
			// write aknowledgement for indication
			p.l2c.Write([]byte{attOpHandleCnf})
		}

	}
}
