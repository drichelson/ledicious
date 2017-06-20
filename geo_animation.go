package main

import (
	"fmt"
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
	"github.com/lucasb-eyer/go-colorful"
	"math/rand"
	"time"
)

const (
	maxSurfaceArea = 12.566370614359172 // 4 * pi
	areaPerPixel   = 0.012566371        //very approximate
	bubbleCount    = 15
)

var (
	caps    = make([]bubble, 0)
	bubbleI = 0
)

type GeoAnimation struct {
	pixels []*BallPixel
}

type bubble struct {
	cap      s2.Cap
	color    colorful.Color
	lifeSpan time.Duration
	birth    time.Time
}

func newBubble() bubble {
	return bubble{
		cap:      s2.CapFromCenterArea(s2.PointFromLatLng(*(pixels.getRandomPixel()).LatLong), areaPerPixel),
		color:    colorful.HappyColor(),
		lifeSpan: time.Duration(5+rand.Intn(10)) * time.Second,
		birth:    time.Now(),
	}
}

//http://www.rapidtables.com/web/color/color-picker.htm
func NewGeoAnimation() Animation {
	bubbleI = 0
	a := GeoAnimation{
		pixels: []*BallPixel{pixels.getRandomPixel()},
	}
	for i := 0; i <= bubbleCount; i++ {
		caps = append(caps, newBubble())
	}
	return &a
}

func (a *GeoAnimation) frame(elapsedTime time.Duration, frameCount int) {
	//for i, b := range caps {
	b := caps[bubbleI]

	//if s2Cap.Area() >= maxSurfaceArea*float64(rand.Intn(9000))/1000.0 {
	//	caps[i] = s2.CapFromCenterArea(s2.PointFromLatLng(*(pixels.getRandomPixel()).LatLong), 0.1)
	//}
	shouldExpand := true
	for otherI, otherB := range caps {
		if otherI != bubbleI {
			if caps[bubbleI].cap.Intersects(otherB.cap) {
				//caps[i] = bubble{s2.CapFromCenterArea(s2.PointFromLatLng(*(pixels.getRandomPixel()).LatLong), areaPerPixel), colorful.HappyColor()}
				//} else {
				caps[bubbleI].cap = s2.CapFromCenterArea(b.cap.Center(), b.cap.Area()*0.8)
				shouldExpand = false
				//Expanded(s1.Angle(0.005))
			}
		}
	}
	if shouldExpand {
		caps[bubbleI].cap = b.cap.Expanded(s1.Angle(0.005))
		//break
	}
	//}
	for _, b := range caps {
		for _, p := range pixels {
			if !p.disabled {
				if b.cap.ContainsPoint(s2.PointFromLatLng(*p.LatLong)) {
					color := b.color
					p.color = &color
				}
			}
		}
	}
	if time.Since(b.birth) > b.lifeSpan {
		caps[bubbleI] = newBubble()
	}

	bubbleI++
	if bubbleI >= bubbleCount {
		bubbleI = 0
	}
	//time.Sleep(50 * time.Millisecond)
	//fmt.Printf("cap: %s\n", s2Cap.String())
}

func (a *GeoAnimation) testCells() {
	for _, p := range pixels {
		reset()
		if !p.disabled {
			cell := *p.cell
			//fmt.Printf("CellId: %s is leaf? %v avg area: %v\n", cell.ID().String(), cell.IsLeaf(), cell.AverageArea())
			intersects := make([]*BallPixel, 0)
			for _, otherpixel := range pixels {
				if !otherpixel.disabled {
					otherCell := *otherpixel.cell
					if p != otherpixel && cell.IntersectsCell(otherCell) {
						//fmt.Printf("%s intersects %s\n", cell.ID().String(), otherCell.ID().String())
						intersects = append(intersects, otherpixel)
					}
				}
			}
			fmt.Printf("Row: %d Col: %d Cell %s intesects %d other cells: \n",
				p.row, p.col, cell.ID().String(), len(intersects))
			intersectsString := ""

			p.color = &colorful.Color{R: 1.0, G: 1.0, B: 1.0}
			for _, p := range intersects {
				p.color = &colorful.Color{R: 1.0}
				intersectsString += fmt.Sprintf("[row: %d col: %d]", p.row, p.col)
			}
			fmt.Printf("\t%s\n", intersectsString)

			render()
			time.Sleep(500 * time.Millisecond)
			//fmt.Printf("Parent 2 cell id: %s\n", cell.ID().Parent(3).String())
			//fmt.Printf("Parent 2 cell id: %s\n", cell.ID().Parent(2).String())
			//fmt.Printf("Parent 1 cell id: %s\n", cell.ID().Parent(1).String())
			//fmt.Printf("Parent 0 cell id: %s\n", cell.ID().Parent(0).String())
		}
	}
}
