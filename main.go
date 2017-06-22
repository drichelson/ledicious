package main

import (
	"fmt"
	"github.com/drichelson/usb-test/usb"
	"github.com/golang/geo/s2"
	"github.com/lucasb-eyer/go-colorful"
	"gopkg.in/macaron.v1"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	GlobalBrightness   = 0.2
	ColumnCount        = 64
	RowCount           = 20
	expectedPixelCount = 1200
)

var (
	pixels   BallPixels
	cells    = make(map[s2.Cell]*BallPixel)
	rows     = make([][]*BallPixel, RowCount)
	cols     = make([][]*BallPixel, ColumnCount)
	renderCh = make(chan []colorful.Color, 1)
	webVar1  = 0
	webVar2  = 0
	webLock  = &sync.RWMutex{}
)

type Animation interface {
	frame(elapsed time.Duration, frameCount int)
}
type BallPixels []*BallPixel

func (bp BallPixels) getRandomPixel() *BallPixel {
	p := bp[rand.Int31n(expectedPixelCount)]
	if p.disabled {
		return bp.getRandomPixel()
	}
	return p
}

type BallPixel struct {
	col      int
	row      int
	x        float64
	y        float64
	z        float64
	LatLong  *s2.LatLng
	cell     *s2.Cell
	color    *colorful.Color
	disabled bool
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)

	m := macaron.Classic()
	m.Use(macaron.Static("assets",
		macaron.StaticOptions{
			// Prefix is the optional prefix used to serve the static directory content. Default is empty string.
			Prefix: "",
			// SkipLogging will disable [Static] log messages when a static file is served. Default is false.
			SkipLogging: true,
			// IndexFile defines which file to serve as index if it exists. Default is "index.html".
			IndexFile: "index.html",
		}))

	m.Get("/var", func(ctx *macaron.Context) string {
		webLock.RLock()
		defer webLock.RUnlock()
		return strconv.Itoa(webVar1)
	})
	m.Put("/var", func(ctx *macaron.Context) string {
		webLock.Lock()
		defer webLock.Unlock()
		newValString := ctx.Query("newVal")
		newVal, err := strconv.Atoi(newValString)
		if err != nil {
			ctx.Resp.WriteHeader(http.StatusBadRequest)
			return "not a number!"
		}
		webVar1 = newVal
		return newValString
	})

	//m.Run()

	usb.Initialize()
	go func() {
		for {
			usb.Render(<-renderCh, GlobalBrightness)
		}
	}()

	//test()
	var a Animation

	//a = NewOpenSimplexAnimation()
	a = NewGeoAnimation()
	startTime := time.Now()
	checkPointTime := startTime
	frameCount := 0

	for {
		a.frame(time.Since(startTime), frameCount)
		render()
		reset()
		frameCount++
		if frameCount%1000 == 0 {
			newCheckPointTime := time.Now()
			fmt.Printf("Avg FPS for past 1000 frames: %v\n", 1000.0/time.Since(checkPointTime).Seconds())
			checkPointTime = newCheckPointTime
		}
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
