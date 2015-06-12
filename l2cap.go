package noblechild

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/paypal/gatt"
)

var (
	infoRegex       = regexp.MustCompile("^info (.*)$")
	connectRegex    = regexp.MustCompile("^connect (.*)$")
	disconnectRegex = regexp.MustCompile("^disconnect$")
	rssiRegex       = regexp.MustCompile("^rssi = (.*)$")
	securityRegex   = regexp.MustCompile("^security = (.*)$")
	writeRegex      = regexp.MustCompile("^write = (.*)$")
	dataRegex       = regexp.MustCompile("^data (.*)$")
)

const (
	ATT_OP_ERROR              = 0x01
	ATT_OP_MTU_REQ            = 0x02
	ATT_OP_MTU_RESP           = 0x03
	ATT_OP_FIND_INFO_REQ      = 0x04
	ATT_OP_FIND_INFO_RESP     = 0x05
	ATT_OP_READ_BY_TYPE_REQ   = 0x08
	ATT_OP_READ_BY_TYPE_RESP  = 0x09
	ATT_OP_READ_REQ           = 0x0a
	ATT_OP_READ_RESP          = 0x0b
	ATT_OP_READ_BLOB_REQ      = 0x0c
	ATT_OP_READ_BLOB_RESP     = 0x0d
	ATT_OP_READ_BY_GROUP_REQ  = 0x10
	ATT_OP_READ_BY_GROUP_RESP = 0x11
	ATT_OP_WRITE_REQ          = 0x12
	ATT_OP_WRITE_RESP         = 0x13
	ATT_OP_HANDLE_NOTIFY      = 0x1b
	ATT_OP_HANDLE_IND         = 0x1d
	ATT_OP_HANDLE_CNF         = 0x1e
	ATT_OP_WRITE_CMD          = 0x52

	ATT_ECODE_SUCCESS              = 0x00
	ATT_ECODE_INVALID_HANDLE       = 0x01
	ATT_ECODE_READ_NOT_PERM        = 0x02
	ATT_ECODE_WRITE_NOT_PERM       = 0x03
	ATT_ECODE_INVALID_PDU          = 0x04
	ATT_ECODE_AUTHENTICATION       = 0x05
	ATT_ECODE_REQ_NOT_SUPP         = 0x06
	ATT_ECODE_INVALID_OFFSET       = 0x07
	ATT_ECODE_AUTHORIZATION        = 0x08
	ATT_ECODE_PREP_QUEUE_FULL      = 0x09
	ATT_ECODE_ATTR_NOT_FOUND       = 0x0a
	ATT_ECODE_ATTR_NOT_LONG        = 0x0b
	ATT_ECODE_INSUFF_ENCR_KEY_SIZE = 0x0c
	ATT_ECODE_INVAL_ATTR_VALUE_LEN = 0x0d
	ATT_ECODE_UNLIKELY             = 0x0e
	ATT_ECODE_INSUFF_ENC           = 0x0f
	ATT_ECODE_UNSUPP_GRP_TYPE      = 0x10
	ATT_ECODE_INSUFF_RESOURCES     = 0x11
)

var (
	GATT_PRIM_SVC_UUID = []byte{0x00, 0x28}
	GATT_INCLUDE_UUID  = []byte{0x02, 0x28}
	GATT_CHARAC_UUID   = []byte{0x03, 0x28}

	GATT_CLIENT_CHARAC_CFG_UUID = []byte{0x02, 0x29}
	GATT_SERVER_CHARAC_CFG_UUID = []byte{0x03, 0x29}
)

type L2CAP_BLE struct {
	path       string
	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	command    *exec.Cmd

	device *device

	Address string

	cmdQueue       chan L2Cmd // TODO: use channel to serialize
	currentCommand *L2Cmd
}

// L2Cmd is an callback
type L2Cmd struct {
	Opcode        byte
	Buffer        []byte
	Callback      func(data []byte) error
	WriteCallback func(data []byte) error
}

func NewL2CAP(d *device, path string) (*L2CAP_BLE, error) {
	l2cap := L2CAP_BLE{
		path:     path,
		device:   d,
		cmdQueue: make(chan L2Cmd),
	}

	return &l2cap, nil
}

func (l2cap *L2CAP_BLE) Init(address, addressType string) error {
	addr := AddrToCommaAddr(address)
	log.Printf("l2cap init: %s %s %s", l2cap.path, addr, addressType)
	cmd := exec.Command(l2cap.path, addr, addressType)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	l2cap.stdinPipe = stdin
	l2cap.stdoutPipe = stdout
	l2cap.command = cmd
	l2cap.Address = address

	go l2cap.Out()
	go l2cap.doCommand()

	cmd.Start()

	go cmd.Wait()

	return nil
}

func (l2cap *L2CAP_BLE) Stop() {
	l2cap.stdinPipe.Close()
	l2cap.stdoutPipe.Close()

	err := l2cap.command.Process.Signal(syscall.SIGINT)
	if err != nil {
		log.Printf("fail to stop l2cap: %s", l2cap.Address)
	}
}

func (l2cap *L2CAP_BLE) doCommand() {
	for {
		select {
		case cmd, ok := <-l2cap.cmdQueue:
			if !ok {
				log.Errorf("cmdQueue channel closed")
				return
			}
			if l2cap.currentCommand == nil {
				l2cap.currentCommand = &cmd
				err := l2cap.write(cmd.Buffer)
				if err != nil {
					log.Errorf("command err:%s", err)
					l2cap.currentCommand = nil
					continue
				}
				if cmd.WriteCallback != nil {
					cmd.WriteCallback(cmd.Buffer)
				}
			} else {

			}
		}
	}
}

func (l2cap *L2CAP_BLE) Out() {
	scanner := bufio.NewScanner(l2cap.stdoutPipe)
	for scanner.Scan() {
		buf := scanner.Text()
		err := l2cap.ParseStdout(buf)
		if err != nil {
			log.Println(err)
		}
	}
}

func (l2cap *L2CAP_BLE) ParseStdout(buf string) error {
	log.Printf("line = %s", buf)

	switch {
	case infoRegex.MatchString(buf):
		// do nothing
		log.Printf(buf)
	case connectRegex.MatchString(buf):
		tmp := connectRegex.FindStringSubmatch(buf)
		if len(tmp) != 2 {
			return fmt.Errorf("invalid connect line: %s", buf)
		}
		var err error
		if tmp[1] == "success" {
			err = nil
		} else {
			err = fmt.Errorf(tmp[1])
		}

		p := NewPerfipheral(l2cap.device, l2cap, l2cap.Address)
		if l2cap.device.peripheralConnected != nil {
			go l2cap.device.peripheralConnected(&p, err)
		}
	case disconnectRegex.MatchString(buf):
		p := NewPerfipheral(l2cap.device, l2cap, l2cap.Address)
		if l2cap.device.peripheralDisconnected != nil {
			go l2cap.device.peripheralDisconnected(&p, nil)
		}

		l2cap.Stop()
	case rssiRegex.MatchString(buf):
		tmp := rssiRegex.FindStringSubmatch(buf)
		if len(tmp) != 2 {
			return fmt.Errorf("invalid rssi line: %s", buf)
		}
		fmt.Printf("rssi: %d", tmp[1])
	case securityRegex.MatchString(buf):
		tmp := securityRegex.FindStringSubmatch(buf)
		if len(tmp) != 2 {
			return fmt.Errorf("invalid security line: %s", buf)
		}
		fmt.Printf("security: %d", tmp[1])
	case writeRegex.MatchString(buf):
		tmp := writeRegex.FindStringSubmatch(buf)
		if len(tmp) != 2 {
			return fmt.Errorf("invalid write line: %s", buf)
		}
		log.Printf("Write:%s", tmp)
		if tmp[1] != "success" {
			// TODO: re-issue current command
		}
	case dataRegex.MatchString(buf):
		tmp := dataRegex.FindStringSubmatch(buf)
		if len(tmp) != 2 {
			return fmt.Errorf("invalid data line: %s", buf)
		}
		lineData := tmp[1]
		if lineData[0] == ATT_OP_HANDLE_NOTIFY || lineData[0] == ATT_OP_HANDLE_IND {
			log.Debugf("handleNotify")
		} else {
			log.Debugf("data: %v", lineData)
		}
		if l2cap.currentCommand != nil {
			if l2cap.currentCommand.Callback != nil {
				dd, err := StringToByte(lineData)
				if err != nil {
					return fmt.Errorf("callback StringToByte failed: %s", err)
				}

				err = l2cap.currentCommand.Callback(dd)
				l2cap.currentCommand = nil
				if err != nil {
					return fmt.Errorf("callback failed: %s", err)
				}
			}
			l2cap.currentCommand = nil
		}
	default:
		return fmt.Errorf("unknown stdout: %s", buf)
	}
	return nil
}

func (l2cap *L2CAP_BLE) Disconnect() error {
	return l2cap.command.Process.Signal(syscall.SIGHUP)
}
func (l2cap *L2CAP_BLE) UpdateRssi() error {
	return l2cap.command.Process.Signal(syscall.SIGUSR1)
}
func (l2cap *L2CAP_BLE) UpgradeSecurity() error {
	return l2cap.command.Process.Signal(syscall.SIGUSR2)
}

func (l2cap *L2CAP_BLE) DiscoverServices(uuids []gatt.UUID) ([]*gatt.Service, error) {
	var ret []*gatt.Service

	l2cap.readByGroupRequest([]byte{0x00, 0x01},
		[]byte{0xff, 0xff}, GATT_PRIM_SVC_UUID)

	time.Sleep(2 * time.Second)

	return ret, NotImplementedError
}

func (l2cap *L2CAP_BLE) write(buf []byte) error {
	data := ByteToString(buf)
	data = strings.TrimSpace(data) + "\n"
	log.Printf("write:%v,%s", buf, data)

	n, err := io.WriteString(l2cap.stdinPipe, data)
	if err != nil {
		return fmt.Errorf("write err: %s", err)
	}
	if n != len(data) {
		return fmt.Errorf("write err: command not write: %d/%d", n, len(data))
	}
	return nil

}
func (l2cap *L2CAP_BLE) queueCommand(cmd byte, buf []byte,
	callback func(data []byte) error,
	writeCallback func(data []byte) error) {
	l2cmd := L2Cmd{
		Opcode:        cmd,
		Buffer:        buf,
		Callback:      callback,
		WriteCallback: writeCallback,
	}

	l2cap.cmdQueue <- l2cmd
}

func (l2cap *L2CAP_BLE) readByGroupRequest(startHandle, endHandle, groupUuid []byte) {
	b := new(bytes.Buffer)
	// var buf = new Buffer(7);
	// <Buffer 10 01 00 ff ff 00 28>
	// <Buffer 10 23 00 ff ff 00 28>
	// <Buffer 10 5a 00 ff ff 00 28>
	// <Buffer 10 5e 00 ff ff 00 28>
	// <Buffer 10 77 00 ff ff 00 28>
	// <Buffer 10 86 00 ff ff 00 28>
	// <Buffer 10 8a 00 ff ff 00 28>
	// <Buffer 10 97 00 ff ff 00 28>

	binary.Write(b, binary.BigEndian, []byte{0x10})
	binary.Write(b, binary.BigEndian, startHandle)
	binary.Write(b, binary.BigEndian, endHandle)
	binary.Write(b, binary.LittleEndian, groupUuid)

	var services []string

	callback := func(data []byte) error {
		log.Infof("read callback %v", data)
		log.Infof("read callback %v", string(data))
		opcode := data[0]
		log.Infof("read opcode %v", opcode)
		if opcode == ATT_OP_READ_BY_GROUP_RESP {
			t := data[1]
			num := (len(data) - 2) / int(t)
			log.Infof("resp: %v, %d", t, num)
			for i := 0; i < num; i++ {
				services = append(services, fmt.Sprintf("%d", i))
			}
			log.Infof("callback: %v", services)
			//			111400ff07ff363e879a0580f7992e444cd42dbc6aa8
		}

		return nil
	}

	l2cap.queueCommand(ATT_OP_READ_BY_GROUP_REQ, b.Bytes(), callback, nil)

	/*	buf.writeUInt8(ATT_OP_READ_BY_GROUP_REQ, 0);
		buf.writeUInt16LE(startHandle, 1);
		buf.writeUInt16LE(endHandle, 3);
		buf.writeUInt16LE(groupUuid, 5);
	*/

	//l2cap.Command(b.Bytes())
}

func ByteToString(b []byte) string {
	return fmt.Sprintf("%02x", b)
}
func StringToByte(arg string) ([]byte, error) {
	if len(arg)%2 != 0 { // something wrong
		return []byte{}, fmt.Errorf("invalid binary is specified")
	}
	ret := make([]byte, 0, len(arg)/2)
	for i := 0; i < len(arg); i += 2 {
		tmp := arg[i : i+2]
		b, err := strconv.ParseInt(tmp, 16, 64)
		if err != nil { // could not parse
			return ret, fmt.Errorf("could not parse binary payload")
		}
		ret = append(ret, byte(b))
	}
	return ret, nil

	return []byte(arg), nil
}
