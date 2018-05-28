package animation

import (
	"github.com/lucasb-eyer/go-colorful"
	"time"
)

type TestPatternAnimation struct {
	control  Control
	currentRow int
	currentColumn int
}

func NewTestPatternAnimation(control Control) *GradientTestAnimation {
	colorA := colorful.Hsv(0.0, 1.0, 0.3) // red
	colorB := colorful.Hsv(0.0, 1.0, 0.0)
	colorC := colorful.Hsv(234.0, 0.0, 0.0)
	colorD := colorful.Hsv(234.0, 1.0, 0.3) // purple

	control.SetColor("A", colorA)
	control.SetColor("B", colorB)
	control.SetColor("C", colorC)
	control.SetColor("D", colorD)

	control.SetVar("A", 0.0)
	control.SetVar("B", 0.1)
	control.SetVar("C", 0.9)
	control.SetVar("D", 1.0)

	return &GradientTestAnimation{
		control: control,
		lat:     -90.0,
	}
}

func (a *TestPatternAnimation) syncControl() {

}

func (a *TestPatternAnimation) frame(elapsed time.Duration, frameCount int) {
	a.syncControl()
	//if frameCount%180 == 0 {
	//	a.lat = -90.0
	//}

	//aVar := a.control.GetVar("A")



	//if a.currentRow

}
