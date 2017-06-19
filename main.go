package main

import (
	"fmt"
	"github.com/drichelson/usb-test/usb"
	"github.com/lucasb-eyer/go-colorful"
	"log"
	"time"
)

const (
	GlobalBrightness   = 0.3
	ColumnCount        = 64
	RowCount           = 20
	expectedPixelCount = 1200
)

var (
	pixels   []*BallPixel
	rows     = make([][]*BallPixel, RowCount)
	cols     = make([][]*BallPixel, ColumnCount)
	renderCh = make(chan []colorful.Color, 1)
)

type Animation interface {
	frame(time float64, frameCount int)
}

type BallPixel struct {
	col      int
	row      int
	x        float64
	y        float64
	z        float64
	lat      float64
	lon      float64
	color    *colorful.Color
	disabled bool
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
	usb.Initialize()

	go func() {
		for {
			usb.Render(<-renderCh, GlobalBrightness)
		}
	}()

	//test()
	var a Animation

	a = NewOpenSimplexAnimation()
	startTime := time.Now()
	checkPointTime := startTime
	frameCount := 0

	for {
		timeSinceStartSeconds := time.Since(startTime).Seconds()
		a.frame(timeSinceStartSeconds, frameCount)
		//w := float64(time.Since(startTime).Nanoseconds())
		//fmt.Printf("%f\n", w)
		//w += 0.005

		render()
		reset()
		//time.Sleep(100 * time.Millisecond)
		frameCount++
		if frameCount%1000 == 0 {
			newCheckPointTime := time.Now()
			fmt.Printf("Avg FPS for past 1000 frames: %v\n", 1000.0/time.Since(checkPointTime).Seconds())
			checkPointTime = newCheckPointTime
		}
		//fmt.Printf("histo: min: %v median: %v, max: %v\n", histo.Min(), histo.Percentile(0.5), histo.Max())
	}
}

//Teensy:
// descriptor: &{Length:18 DescriptorType:Device descriptor. USBSpecification:0x0200 (2.00) DeviceClass:Communications class. DeviceSubClass:0 DeviceProtocol:0 MaxPacketSize0:64 VendorID:5824 ProductID:1155 DeviceReleaseNumber:0x0100 (1.00) ManufacturerIndex:1 ProductIndex:2 SerialNumberIndex:3 NumConfigurations:1}

func reset() {
	for i := range pixels {
		pixels[i].color = &colorful.Color{}
	}
}

func render() {
	colors := make([]colorful.Color, len(pixels))
	for i, p := range pixels {
		colors[i] = *p.color
	}
	renderCh <- colors
}

func test() {
	for _, c := range []colorful.Color{{R: 1.0}, {G: 1.0}, {B: 1.0}} {
		for i, row := range rows {
			fmt.Printf("row: %d\n", i)
			reset()
			for _, pixel := range row {
				pixel.color = &c
			}
			render()
			time.Sleep(50 * time.Millisecond)
		}
	}
	for _, c := range []colorful.Color{{R: 1.0}, {G: 1.0}, {B: 1.0}} {
		for i, col := range cols {
			fmt.Printf("column: %d\n", i)
			reset()
			for _, pixel := range col {
				pixel.color = &c
			}
			render()
			time.Sleep(50 * time.Millisecond)
		}
	}
	reset()
	render()
	time.Sleep(50 * time.Millisecond)
}
