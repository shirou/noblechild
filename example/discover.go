package main

import (
	"fmt"
	"log"
	"time"

	gatt "github.com/shirou/gatt"

	noblechild "github.com/shirou/noblechild"
)

func onStateChanged(d gatt.Device, s gatt.State) {
	fmt.Println("State:", s)
	switch s {
	case gatt.StatePoweredOn:
		fmt.Println("scanning...")
		d.Scan([]gatt.UUID{}, false)
		return
	default:
		d.StopScanning()
	}
}

func onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
	fmt.Printf("\nPeripheral ID:%s, NAME:(%s)\n", p.ID(), p.Name())
	fmt.Println("  Local Name        =", a.LocalName)
	fmt.Println("  TX Power Level    =", a.TxPowerLevel)
	fmt.Println("  Manufacturer Data =", a.ManufacturerData)
	fmt.Println("  Service Data      =", a.ServiceData)
}

var DefaultClientOptions = []gatt.Option{}

func main() {
	d, err := noblechild.NewDevice(DefaultClientOptions...)
	if err != nil {
		log.Fatalf("Failed to open device, err: %s\n", err)
		return
	}

	// Register handlers.
	d.Handle(noblechild.PeripheralDiscovered(onPeriphDiscovered))

	d.Init(onStateChanged)

	for {
		time.Sleep(10 * time.Minute)
	}
}
