package animation

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/golang/geo/s2"
	"github.com/lucasb-eyer/go-colorful"
)

const (
	moverCount = 5
)

var (
	black = colorful.Color{}
)

type GeoAnimation2 struct {
	control Control
	movers  []mover
}

type mover struct {
	cap           s2.Cap
	startPoint    s2.Point
	speed         float64
	color         colorful.Color
	bearing       float64
	totalDistance float64
	distanceSoFar float64
	tailLength    int
	tail          []s2.Cap
	done          bool
}

func (m *mover) String() string {
	ll := s2.LatLngFromPoint(m.cap.Center())
	return fmt.Sprintf("%s bearing: %3.2f", ll.String(), m.bearing)
}

//http://www.rapidtables.com/web/color/color-picker.htm
func NewGeoAnimation2(control Control) Animation {
	movers := make([]mover, moverCount)
	for i, _ := range movers {
		movers[i] = newMover()
	}
	a := GeoAnimation2{
		control: control,
		movers:  movers,
	}
	return &a
}

func newMover() mover {
	cap := newRandomCap()

	tailLength := 50
	tail := make([]s2.Cap, tailLength)
	for i, _ := range tail {
		tail[i] = cap
	}
	return mover{
		cap:        cap,
		startPoint: cap.Center(),
		speed:      float64(rand.Intn(40) + 50),
		color:      colorful.WarmColor(),
		bearing:    float64(rand.Intn(3600)) / 10.0,
		//totalDistance: float64(rand.Intn(100000) + int64(20000000000)),
		tailLength: tailLength,
		tail:       tail,
		done:       false,
	}
}

func (m *mover) move() {
	if m.distanceSoFar > m.totalDistance || float64Equal(m.distanceSoFar, m.totalDistance) {
		m.done = true
		//fmt.Println("Done!")
		return
	}
	//fmt.Printf("start: %s %s\n", m.String(), m.cap.Center().Vector.String())
	oldBearing := m.bearing
	//oldLL := s2.LatLngFromPoint(m.cap.Center())
	//travel 10% each move
	distanceToTravel := m.distanceSoFar + m.speed
	if distanceToTravel >= m.totalDistance {
		distanceToTravel = m.totalDistance
	}

	//fmt.Printf("distance so far: %2.3f to travel now: %2.3f\n", m.distanceSoFar, distanceToTravel)
	m.distanceSoFar = distanceToTravel
	//distanceSoFarAsPercent := math.Min(1.0, m.distanceSoFar/m.totalDistance)
	//h, s, v := m.color.Hsv()
	//newV := v*(1.0-distanceSoFarAsPercent) + 0.01
	//m.color = colorful.Hsv(h, s, newV)

	newCenterPoint := toPoint(m.startPoint, distanceToTravel, oldBearing)
	//newLL := s2.LatLngFromPoint(newCenter)
	//fmt.Printf("Old point: %s new point: %s\n", oldLL.String(), newLL.String())
	newCenter := newCap(newCenterPoint)
	for i := m.tailLength - 1; i > 0; i-- {
		m.tail[i] = m.tail[i-1]
	}
	m.tail[0] = newCenter
	m.cap = newCenter
	//fmt.Println(m.tail[4].Center().Distance(m.tail[0].Center()).String())
	//detect north/south pole crossing:

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
	wg := sync.WaitGroup{}
	for i, m := range a.movers {
		if m.done {
			a.movers[i] = newMover()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			a.movers[i].move()
		}()
		wg.Wait()
		//newCenter := toPoint(b.cap.Center(), 100.0, -45.0)
		//fmt.Printf("%d: %s\n", i, s2.LatLngFromPoint(newCenter).String())
		//a.movers[i].cap = s2.CapFromCenterArea(newCenter, 0.01)
	}

	for _, b := range a.movers {
		mover := b
		color := b.color
		for _, p := range pixels.active {
			if b.cap.ContainsPoint(p.Point) {
				p.color = &color
			}
			for i, t := range mover.tail {
				blendAmount := float64(i) / float64(mover.tailLength)
				if t.ContainsPoint(p.Point) {
					//fmt.Println("tail")
					newColor := color.BlendLab(black, blendAmount)
					p.color = &newColor
				}
			}

		}

	}
	//time.Sleep(50 * time.Millisecond)
}
