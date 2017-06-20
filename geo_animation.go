package main

import (
	"fmt"
	"github.com/golang/geo/s1"
	"github.com/golang/geo/s2"
	"github.com/lucasb-eyer/go-colorful"
	"time"
)

const (
	maxSurfaceArea = 12.566370614359172 // 4 * pi
)

var (
	s2Cap s2.Cap
)

type GeoAnimation struct {
	pixels []*BallPixel
}

//http://www.rapidtables.com/web/color/color-picker.htm
func NewGeoAnimation() Animation {
	a := GeoAnimation{
		pixels: []*BallPixel{pixels.getRandomPixel()},
	}
	fmt.Println(s2.FullCap().Area())
	//s2Cap = s2.CapFromPoint(s2.PointFromLatLng(*a.pixels[0].LatLong))
	//s2Cap = s2.CapFromCenterArea(s2.PointFromLatLng(*(pixels.getRandomPixel()).LatLong), 0.1)
	return &a
}

func (a *GeoAnimation) frame(elapsedTime float64, frameCount int) {
	if s2Cap.Area() >= maxSurfaceArea || s2Cap.Area() <= 0.0 {
		s2Cap = s2.CapFromCenterArea(s2.PointFromLatLng(*(pixels.getRandomPixel()).LatLong), 0.1)
	}
	area := s2Cap.Area()
	for _, p := range pixels {
		if !p.disabled {
			if s2Cap.ContainsPoint(s2.PointFromLatLng(*p.LatLong)) {
				color := colorful.Hsv(180.0*area/maxSurfaceArea, 1.0, 0.6)
				p.color = &color
			}
		}
	}
	time.Sleep(70 * time.Millisecond)
	//fmt.Printf("cap: %s\n", s2Cap.String())
	s2Cap = s2Cap.Expanded(s1.Angle(0.1))
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
