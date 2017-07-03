package animation

import (
	"fmt"
	"github.com/golang/geo/s2"
	"github.com/lucasb-eyer/go-colorful"
	"time"
)

var (
	NorthPole          = s2.PointFromLatLng(s2.LatLngFromDegrees(90.0, 0.0))
	SouthPole          = s2.PointFromLatLng(s2.LatLngFromDegrees(-90.0, 0.0))
	EquatorMeridian    = s2.PointFromLatLng(s2.LatLngFromDegrees(0.0, 0.0))
	EquatorNonMeridian = s2.PointFromLatLng(s2.LatLngFromDegrees(0.0, 180.0))
)

type GradientTestAnimation struct {
	control  Control
	gradient GradientTable
	lat      float64
}

func NewGradientTestAnimation(control Control) *GradientTestAnimation {
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

func (a *GradientTestAnimation) syncControl() {
	a.gradient = GradientTable{
		{a.control.GetColor("A"), a.control.GetVar("A")},
		{a.control.GetColor("B"), a.control.GetVar("B")},
		{a.control.GetColor("C"), a.control.GetVar("C")},
		{a.control.GetColor("D"), a.control.GetVar("D")},
	}
}

func (a *GradientTestAnimation) frame(elapsed time.Duration, frameCount int) {
	a.syncControl()
	//if frameCount%180 == 0 {
	//	a.lat = -90.0
	//}

	lat := (a.control.GetVar("A") * 180.0) - 90.0
	for i, p := range pixels.active {
		if s2.LatLngFromPoint(p.Point).Lat.Degrees() < lat {
			pixels.active[i].color = &colorful.Color{R: 1.0}
		}
		//fmt.Printf("lat: %3.2f lon: %3.2f dist from south pole: %3.2f\n",
		//	s2.LatLngFromPoint(p.Point).Lat.Degrees(),
		//	s2.LatLngFromPoint(p.Point).Lng.Degrees(),
		//	p.Point.Distance(SouthPole).Degrees())
		//c := a.gradient.GetInterpolatedColorFor(noiseValNormalized)
		//p.color = &c
		//fmt.Printf("%v\n", noiseVal)
	}

	time.Sleep(100 * time.Millisecond)
	fmt.Println(lat)
	a.lat += 1.0
}
