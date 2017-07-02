package animation

import (
	"fmt"
	"github.com/drichelson/ledicious/usb"
	"github.com/golang/geo/s2"
	"github.com/lucasb-eyer/go-colorful"
	"math/rand"
	"time"
)

const (
	GlobalBrightness   = 1.0
	ColumnCount        = 64
	RowCount           = 20
	expectedPixelCount = 1200
)

var (
	pixels   Pixels
	renderCh = make(chan []colorful.Color, 1)
	rows     = make([][]*Pixel, RowCount)
	cols     = make([][]*Pixel, ColumnCount)
)

type Animation interface {
	frame(elapsed time.Duration, frameCount int)
}
type Pixels struct {
	all    []*Pixel
	active []*Pixel
}

type Pixel struct {
	col      int
	row      int
	x        float64
	y        float64
	z        float64
	Point    s2.Point
	color    *colorful.Color
	disabled bool
}

func Start(control Control) {
	usb.Initialize()
	go func() {
		for {
			usb.Render(<-renderCh, GlobalBrightness)
		}
	}()

	var a Animation

	a = NewOpenSimplexAnimation(control)
	//a = NewGeoAnimation(control)
	//a = NewBrightnessTestAnimation(control)
	startTime := time.Now()
	checkPointTime := startTime
	frameCount := 0

	for {
		a.frame(time.Since(startTime), frameCount)
		pixels.render()
		pixels.reset()
		frameCount++
		if frameCount%1000 == 0 {
			newCheckPointTime := time.Now()
			fmt.Printf("Avg FPS for past 1000 frames: %v\n", 1000.0/time.Since(checkPointTime).Seconds())
			checkPointTime = newCheckPointTime
		}
	}
}

func (p Pixels) getRandomPixel() *Pixel {
	return p.active[rand.Int31n(int32(len(pixels.active)))]
}

func (p *Pixels) reset() {
	for i := range pixels.active {
		pixels.active[i].color = &colorful.Color{}
	}
}

func (p *Pixels) render() {
	colors := make([]colorful.Color, len(pixels.all))
	for i, p := range pixels.all {
		colors[i] = *p.color
	}
	renderCh <- colors
}

func test() {
	for _, c := range []colorful.Color{{R: 1.0}, {G: 1.0}, {B: 1.0}} {
		for i, row := range rows {
			fmt.Printf("row: %d\n", i)
			pixels.reset()
			for _, pixel := range row {
				pixel.color = &c
			}
			pixels.render()
			time.Sleep(50 * time.Millisecond)
		}
	}
	for _, c := range []colorful.Color{{R: 1.0}, {G: 1.0}, {B: 1.0}} {
		for i, col := range cols {
			fmt.Printf("column: %d\n", i)
			pixels.reset()
			for _, pixel := range col {
				pixel.color = &c
			}
			pixels.render()
			time.Sleep(50 * time.Millisecond)
		}
	}
	pixels.reset()
	pixels.render()
	time.Sleep(50 * time.Millisecond)
}
