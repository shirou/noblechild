package noblechild

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
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

const ()

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

	ackChan chan string
}

func NewL2CAP(d *device, path string) (*L2CAP_BLE, error) {
	l2cap := L2CAP_BLE{
		path:    path,
		device:  d,
		ackChan: make(chan string),
	}

	return &l2cap, nil
}

func (l2cap *L2CAP_BLE) Init(address, addressType string) error {
	addr := AddrToCommaAddr(address)
	log.Debugf("l2cap init: %s %s %s", l2cap.path, addr, addressType)
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

	cmd.Start()

	go cmd.Wait()

	return nil
}

func (l2cap *L2CAP_BLE) Close() error {
	var errFinished = errors.New("os: process already finished")

	l2cap.stdinPipe.Close()
	l2cap.stdoutPipe.Close()

	err := l2cap.command.Process.Signal(syscall.SIGINT)
	if err != nil && err != errFinished {
		log.Infof("fail to stop l2cap: %s, %s", l2cap.Address, err)
		return err
	}
	return nil
}

func (l2cap *L2CAP_BLE) Write(buf []byte) (int, error) {
	data := ByteToString(buf)
	data = strings.TrimSpace(data) + "\n"
	log.Debugf("l2cap write:%v,%s", buf, strings.TrimSpace(data))

	n, err := io.WriteString(l2cap.stdinPipe, data)
	if err != nil {
		return -1, fmt.Errorf("l2cap write err: %s", err)
	}
	return n, nil
}
func (l2cap *L2CAP_BLE) Read(b []byte) (int, error) {
	if l2cap == nil || l2cap.ackChan == nil {
		return 0, nil
	}
	lineData, ok := <-l2cap.ackChan
	if !ok {
		return 0, io.EOF
	}

	dd, err := StringToByte(lineData)
	if err != nil {
		return 0, fmt.Errorf("StringToByte failed: %s", err)
	}
	copy(b, dd)

	return len(dd), nil
}
func (l2cap *L2CAP_BLE) Out() {
	scanner := bufio.NewScanner(l2cap.stdoutPipe)
	for scanner.Scan() {
		buf := scanner.Text()
		err := l2cap.ParseStdout(buf)
		if err != nil {
			log.Errorf("l2cap Out failed:%s", err)
		}
	}
}

func (l2cap *L2CAP_BLE) ParseStdout(buf string) error {
	log.Debugf("line = %s", buf)

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

		p := NewPeripheral(l2cap.device, l2cap, l2cap.Address)
		if l2cap.device.peripheralConnected != nil {
			go l2cap.device.peripheralConnected(&p, err)
		}
	case disconnectRegex.MatchString(buf):
		p := NewPeripheral(l2cap.device, l2cap, l2cap.Address)
		if l2cap.device.peripheralDisconnected != nil {
			go l2cap.device.peripheralDisconnected(&p, nil)
		}

		l2cap.Close()
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
		if tmp[1] != "success" {
			log.Printf("write failed:%s", tmp)
			// TODO: re-issue current command
		}
	case dataRegex.MatchString(buf):
		tmp := dataRegex.FindStringSubmatch(buf)
		if len(tmp) != 2 {
			return fmt.Errorf("invalid data line: %s", buf)
		}
		lineData := tmp[1]
		l2cap.ackChan <- lineData
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
