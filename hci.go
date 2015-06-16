package noblechild

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/shirou/gatt"
)

var (
	adapterRegex = regexp.MustCompile("^adapterState (.*)$")
	eventRegex   = regexp.MustCompile("^event (.*)$")
)

// HCI_BLE is a struct to use noble's hci-ble binary.
type HCI_BLE struct {
	path       string
	stdinPipe  io.WriteCloser
	stdoutPipe io.ReadCloser
	command    *exec.Cmd

	device *device

	previousAdapterState string
	currentAdapterState  string

	discoveries map[string]HCIEvent
}

// HCIEvent represents some events from hci.
type HCIEvent struct {
	Address       string
	AddressType   string
	EIR           string
	Advertisement *gatt.Advertisement // JSON
	RSSI          int
	Count         int
}

func NewHCI(d *device, path string) (*HCI_BLE, error) {
	di := make(map[string]HCIEvent)

	hci := HCI_BLE{
		path:        path,
		device:      d,
		discoveries: di,
	}

	return &hci, nil
}

func (hci *HCI_BLE) Init() error {
	cmd := exec.Command(hci.path)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	hci.stdinPipe = stdin
	hci.stdoutPipe = stdout
	hci.command = cmd

	go hci.Out()

	cmd.Start()

	go cmd.Wait()

	return nil
}

func (hci *HCI_BLE) Close() error {
	err := hci.StopScan()
	if err != nil {
		return fmt.Errorf("hci close: stop scan failed:%s", err)
	}
	err = hci.command.Process.Signal(syscall.SIGINT)
	if err != nil {
		return fmt.Errorf("hci close: stop hci failed:%s", err)
	}
	return nil
}

// Out read stdout from hci
func (hci *HCI_BLE) Out() {
	scanner := bufio.NewScanner(hci.stdoutPipe)
	for scanner.Scan() {
		buf := scanner.Text()
		hci.ParseStdout(buf)
	}
}

func (hci *HCI_BLE) StartScan() error {
	return hci.command.Process.Signal(syscall.SIGUSR1)
}
func (hci *HCI_BLE) StartScanFilter() error {
	return hci.command.Process.Signal(syscall.SIGUSR2)
}
func (hci *HCI_BLE) StopScan() error {
	return hci.command.Process.Signal(syscall.SIGHUP)
}

func (hci *HCI_BLE) ParseStdout(buf string) {
	switch {
	case adapterRegex.MatchString(buf):
		tmp := adapterRegex.FindStringSubmatch(buf)
		if len(tmp) != 2 {
			log.Println("invalid adapter state line: %s", buf)
			return
		}
		adapterState := tmp[1]
		if adapterState == "unauthorized" {
		}
		if adapterState == "unsupported" {
		}

		var state gatt.State
		switch adapterState {
		case "unknown":
			state = gatt.StateUnknown
		case "unsupported":
			log.Error("noble warning: adapter does not support Bluetooth Low Energy (BLE, Bluetooth Smart).")
			state = gatt.StateUnsupported
		case "unauthorized":
			log.Error("warning: adapter state unauthorized, please run as root or with sudo")
			state = gatt.StateUnauthorized
		case "poweredOff":
			state = gatt.StatePoweredOff
		case "poweredOn":
			state = gatt.StatePoweredOn
		}

		hci.device.stateChanged(hci.device, state)
	case eventRegex.MatchString(buf):
		tmp := eventRegex.FindStringSubmatch(buf)
		if len(tmp) != 2 {
			log.Errorf("invalid event line: %s", buf)
			return
		}
		event := tmp[1]
		e, err := parseEvent(event)
		if err != nil {
			log.Errorf("parse event failed: %s", err)
			return
		}

		prevE, ok := hci.discoveries[e.Address]
		if ok {
			e.Count = prevE.Count + 1
		}
		hci.discoveries[e.Address] = e

		// only report after an even number of events, so more advertisement data can be collected
		if e.Count%2 == 0 {
			noble, err := FindNobleModule()
			if err != nil {
				log.Errorf("could not find l2cap at ParseStdout: %s", err)
				return
			}
			l2cap, err := NewL2CAP(hci.device, noble.L2CAPPath)
			if err != nil {
				log.Errorf("could not new l2cap at ParseStdout: %s", err)
				return
			}

			p := NewPeripheral(hci.device, l2cap, e.Address)
			hci.device.peripheralDiscovered(&p, e.Advertisement, e.RSSI)
		}

	default:
		log.Errorf("unknown output: %s", buf)
	}
}

func parseEvent(event string) (HCIEvent, error) {
	ret := HCIEvent{}

	splitEvent := strings.Split(event, ",")
	if len(splitEvent) != 4 {
		return ret, fmt.Errorf("invalid event buf len: %s", event)
	}

	tmp := strings.ToLower(splitEvent[0])
	ret.Address = strings.Replace(tmp, ":", "", -1)

	ret.AddressType = splitEvent[1]
	eir, err := hex.DecodeString(splitEvent[2])
	if err != nil {
		return ret, fmt.Errorf("invalid event eir: %s, %s", err, event)
	}
	adv, err := parseEIR(eir)
	if err != nil {
		return ret, fmt.Errorf("parse EIR failed: %s, %s", err, event)
	}
	ret.Advertisement = &adv

	rssi, err := strconv.Atoi(splitEvent[3])
	if err != nil {
		return ret, fmt.Errorf("invalid event rssi: %s, %s", err, event)
	}
	ret.RSSI = rssi

	return ret, nil
}

func parseEIR(eir []byte) (gatt.Advertisement, error) {
	ret := gatt.Advertisement{}

	i := 0
	for {
		if i+1 > len(eir) {
			break
		}
		length := int(eir[i])
		t := eir[i+1]
		data := eir[i+2 : i+2+length-1]
		switch t {
		case 0x02: // Incomplete List of 16-bit Service Class UUID
			fallthrough
		case 0x03: // Complete List of 16-bit Service Class UUIDs
			// TODO

			/*
			   for (j = 0; j < bytes.length; j += 2) {
			     serviceUuid = bytes.readUInt16LE(j).toString(16);
			     if (advertisement.serviceUuids.indexOf(serviceUuid) === -1) {
			       advertisement.serviceUuids.push(serviceUuid);
			     }
			   }*/
			/*
				for j := 0; j < len(data); j += 2 {
					tmp := binary.LittleEndian.Uint32([]byte{data[j]})
					fmt.Printf("%v", data)
					fmt.Printf("%02x", data[j:j+1])
					hex := fmt.Sprintf("%02x", data[j:j+1])
					uuid, err := gatt.ParseUUID(hex)
					if err != nil {
						return ret, err
					}

					if !stringInUUID(uuid, ret.Services) {
						ret.Services = append(ret.Services, uuid)
					}
				}
			*/
		case 0x06: // Incomplete List of 128-bit Service Class UUIDs
			fallthrough
		case 0x07: // Complete List of 128-bit Service Class UUIDs
			for j := 0; j < len(data); j += 16 {
				var hex []string
				buf := data[j : j+16]
				// hex should be reverse
				for i := len(buf) - 1; i >= 0; i-- {
					hex = append(hex, fmt.Sprintf("%02x", buf[i]))
				}
				uuid, err := gatt.ParseUUID(strings.Join(hex, ""))
				if err != nil {
					return ret, err
				}
				if !IncludesUUID(uuid, ret.Services) {
					ret.Services = append(ret.Services, uuid)
				}
			}
		case 0x08: // Shortened Local Name
			fallthrough
		case 0x09: // Complete Local Name»
			ret.LocalName = string(data)
		case 0x0a: // Tx Power Level
			ret.TxPowerLevel = int(data[0])
		case 0x16: // Service Data, there can be multiple occurences
			// TODO
			/*
			   var serviceDataUuid = bytes.slice(0, 2).toString('hex').match(/.{1,2}/g).reverse().join('');
			   var serviceData = bytes.slice(2, bytes.length);

			   advertisement.serviceData.push({
			     uuid: serviceDataUuid,
			     data: serviceData
			   });
			*/
		case 0xff: // Manufacturer Specific Data
			ret.ManufacturerData = data
			break
		}

		i = i + length + 1
	}
	return ret, nil
}

type ByteSlice []byte

func (s ByteSlice) Len() int           { return len(s) }
func (s ByteSlice) Less(i, j int) bool { return s[i] < s[j] }
func (s ByteSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
