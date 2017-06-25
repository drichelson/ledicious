package main

import (
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
	"github.com/lucasb-eyer/go-colorful"
	"time"
)

const (
	maxSurfaceArea = 12.566370614359172 // 4 * pi
)

var (
	bubbles = make([]bubble, 0)
)

type GeoAnimation struct {
	pixels []*GlobePixel
}

type bubble struct {
	cap   s2.Cap
	color colorful.Color
}

//http://www.rapidtables.com/web/color/color-picker.htm
func NewGeoAnimation() Animation {
	a := GeoAnimation{
		pixels: []*GlobePixel{pixels.getRandomPixel()},
	}
	for i := 0; i < 7; i++ {
		bubbles = append(bubbles, newBubble(0))
	}
	//fmt.Println(s2.FullCap().Area())
	//s2Cap = s2.CapFromPoint(s2.PointFromLatLng(*a.pixels[0].LatLong))
	//s2Cap = s2.CapFromCenterArea(s2.PointFromLatLng(*(pixels.getRandomPixel()).LatLong), 0.1)
	return &a
}

func newBubble(depth int) bubble {
	newB := bubble{
		cap:   newCap(),
		color: colorful.HappyColor(),
		//color: colorful.Color{R: float64(rand.Intn(100)) / 100.0, G: float64(rand.Intn(100)), B: float64(rand.Intn(100))},
	}
	if depth < 100 {
		for _, b := range bubbles {
			distance := newB.cap.Center().Distance(b.cap.Center()).Degrees()
			if distance <= 3.0*b.cap.Radius().Degrees() {
				//fmt.Printf("too close: %v\n", distance)
				return newBubble(depth + 1)
			}
		}
	}
	return newB
}

func newCap() s2.Cap {
	return s2.CapFromCenterArea(pixels.getRandomPixel().Point, 0.05)
}

func (a *GeoAnimation) frame(elapsed time.Duration, frameCount int) {
	//replaced := make(map[int]bool)
	for i, b := range bubbles {
		//if skip, _ := replaced[i]; skip {
		//	break
		//}
		//fmt.Printf("[%d]Processing bubble %d\n", frameCount, i)
		bubbles[i].cap = bubbles[i].cap.Expanded(s1.Angle(0.005))

		if b.cap.Area() >= maxSurfaceArea/4.0 || b.cap.Area() <= 0.0 {
			bubbles[i] = newBubble(0)
		}

		for otherI, otherB := range bubbles {
			if otherI != i {
				if b.cap.Intersects(otherB.cap) {
					if b.cap.Area() > otherB.cap.Area() {
						bubbles[otherI] = newBubble(0)
						//replaced[otherI] = true
					} else {
						bubbles[i] = newBubble(0)
						break
					}
					//fmt.Printf("\tpopping bubble %d because it hit bubble %d\n", i, otherI)
				}
			}
		}

	}
	for i, _ := range bubbles {
		capRadius := bubbles[i].cap.Radius().Degrees()
		//fmt.Printf("capRadius: %v\n", capRadius)
		for _, p := range pixels {
			if !p.disabled {
				if bubbles[i].cap.ContainsPoint(p.Point) {
					distanceFromCenter := p.Point.Distance(bubbles[i].cap.Center())
					//fmt.Printf("distanceFromCenter: %v\n", distanceFromCenter)

					h, s, _ := bubbles[i].color.Hsv()
					c := colorful.Hsv(h, s, 1.0-(distanceFromCenter.Degrees()/capRadius))
					p.color = &c
				}
			}
		}
	}
	time.Sleep(40 * time.Millisecond)
}
