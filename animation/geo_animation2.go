package animation

import (
	"fmt"
	"github.com/golang/geo/s2"
	"github.com/lucasb-eyer/go-colorful"
	"time"
)

const ()

var ()

type GeoAnimation2 struct {
	control Control
	movers  []mover
}

type mover struct {
	cap     s2.Cap
	color   colorful.Color
	bearing float64
}

func (m *mover) String() string {
	ll := s2.LatLngFromPoint(m.cap.Center())
	return fmt.Sprintf("%s bearing: %3.2f", ll.String(), m.bearing)
}

//http://www.rapidtables.com/web/color/color-picker.htm
func NewGeoAnimation2(control Control) Animation {
	a := GeoAnimation2{
		control: control,
		movers:  []mover{newMover()},
	}
	return &a
}

func newMover() mover {
	return mover{cap: newCap(), color: colorful.WarmColor(), bearing: 360.0} //float64(rand.Intn(3600))/10.0 - 180.0}
}

func (m *mover) move(distance float64) {
	fmt.Printf("start: %s %s\n", m.String(), m.cap.Center().Vector.String())
	oldBearing := m.bearing
	oldLL := s2.LatLngFromPoint(m.cap.Center())
	newCenter := toPoint(m.cap.Center(), distance, oldBearing)
	newLL := s2.LatLngFromPoint(newCenter)
	_, reverseB := geo.To(newLL.Lat.Degrees(), newLL.Lng.Degrees(), oldLL.Lat.Degrees(), oldLL.Lng.Degrees())

	fmt.Printf("reverse bearing: %2.3f\n", reverseB)

	if float64Equal(reverseBearing(oldBearing), reverseB) {
		fmt.Printf("not changing bearing\n")
		// time to change bearings!
	} else {
		fmt.Printf("changing bearing to: %3.2f\n", reverseBearing(oldBearing))
		m.bearing = reverseBearing(oldBearing)
	}

	m.cap = s2.CapFromCenterArea(newCenter, 0.05)
	//newLat := s2.LatLngFromPoint(newCenter).Lat.Degrees()
	//newLon := s2.LatLngFromPoint(newCenter).Lng.Degrees()
	//if oldLat >= 0.0 && newLat < oldLat { //we crossed the north pole
	//	if oldBearing <= 180.0 {
	//		m.bearing = oldBearing + 180.0
	//	}
	//	m.bearing = m.bearing - 180.0
	//	//fmt.Printf("oldbearing: %3.2f newBearing: %3.2f newLat: %3.2f newLon: %3.2f\n",
	//	//	oldBearing, m.bearing, newLat, newLon)
	//}
	//fmt.Printf("end: %s\n", m.String())

}

func (a *GeoAnimation2) frame(elapsed time.Duration, frameCount int) {
	for i, _ := range a.movers {
		a.movers[i].move(1000.0)
		//newCenter := toPoint(b.cap.Center(), 100.0, -45.0)
		//fmt.Printf("%d: %s\n", i, s2.LatLngFromPoint(newCenter).String())
		//a.movers[i].cap = s2.CapFromCenterArea(newCenter, 0.01)
	}

	for _, b := range a.movers {
		for _, p := range pixels.active {
			if b.cap.ContainsPoint(p.Point) {
				p.color = &b.color
			}
		}
	}
	time.Sleep(100 * time.Millisecond)
}
