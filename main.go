package main

import (
	"github.com/drichelson/usb-test/usb"
	"github.com/lucasb-eyer/go-colorful"
	"log"
	"math/rand"
	"time"
)



func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
	Init()
}

//Teensy:
// descriptor: &{Length:18 DescriptorType:Device descriptor. USBSpecification:0x0200 (2.00) DeviceClass:Communications class. DeviceSubClass:0 DeviceProtocol:0 MaxPacketSize0:64 VendorID:5824 ProductID:1155 DeviceReleaseNumber:0x0100 (1.00) ManufacturerIndex:1 ProductIndex:2 SerialNumberIndex:3 NumConfigurations:1}

func Init() {
	usb.Initialize()
	//temp
	start := time.Now()
	pixels := make([]colorful.Color, 1200)
	for i := 0; i < 100; i++ {
		for i := range pixels {
			pixels[i] = colorful.Color{R: rand.Float64(), G: rand.Float64(), B: rand.Float64()}
		}
		usb.Render(pixels, 0.6)
		//time.Sleep(10 * time.Millisecond)
	}
	log.Printf("avg time per frame: %v", time.Since(start) / 100)
}

