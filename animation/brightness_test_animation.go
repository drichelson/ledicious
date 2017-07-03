package animation

import (
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
	control Control
}

func NewBrightnessTestAnimation(control Control) *BrightnessTestAnimation {
	return &BrightnessTestAnimation{
		control: control,
	}
}

func (a *BrightnessTestAnimation) frame(elapsed time.Duration, frameCount int) {
	//v := a.control.GetVar("A")
	for i, p := range pixels.active {
		v := float64(p.col) / float64(len(cols))
		c := colorful.Color{R: v}
		//fmt.Printf("v: %v\n", v)
		pixels.active[i].color = &c
	}
	//fmt.Printf("v: %v\n", v)

	time.Sleep(500 * time.Millisecond)
}
