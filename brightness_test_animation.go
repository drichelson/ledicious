package main

import (
	"fmt"
	"github.com/lucasb-eyer/go-colorful"
	"time"
)

/*
Notes:
in RGB world:
we end up sending each value as a byte, and the minimum visible value is not 1, but is 2:
2/255 = .007843137
Minimum visible R: 0.00784314
Minimum visible G: 0.00784314
Minimum visible B: 0.00784314
*/

type BrightnessTestAnimation struct {
}

func (a *BrightnessTestAnimation) frame(elapsed time.Duration, frameCount int) {
	//v := (float64(frameCount%100) / 100.0) / 4.0
	v := float64(webVar1) / 1000.0
	for i, _ := range pixels.active {
		//v := float64(p.col) / float64(len(cols))
		c := colorful.Color{R: v}
		pixels.active[i].color = &c
	}
	//fmt.Printf("v: %v\n", v)
	fmt.Printf("v: %v\n", v)

	time.Sleep(100 * time.Millisecond)
}
