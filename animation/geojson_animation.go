package animation

import (
	"github.com/lucasb-eyer/go-colorful"
	"time"
	"github.com/paulmach/orb/geojson"
	"io/ioutil"
	"github.com/paulmach/orb"
	"github.com/golang/geo/s2"
	"fmt"
)

var (
	c = colorful.WarmColor()
)

type GeojsonAnimation struct {
	control           Control
	featureCollection *geojson.FeatureCollection
	countries         map[string]*s2.Polygon
}

func NewGeojsonAnimation(control Control) *GeojsonAnimation {
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

	bytes, err := ioutil.ReadFile("animation/custom.geo.json")
	if err != nil {
		panic(err)
	}
	fc, e := geojson.UnmarshalFeatureCollection(bytes)
	if e != nil {
		panic(e)
	}
	polygonMap := make(map[string]*s2.Polygon)

	for _, feature := range fc.Features {
		polygonType := feature.Geometry.GeoJSONType()
		country := feature.Properties.MustString("name_sort", "no name")
		//fmt.Println(polygonType)
		if polygonType == "Polygon" {
			polygon := feature.Geometry.(orb.Polygon)
			loops := make([]*s2.Loop, len(polygon))
			for r, ring := range polygon { // Ring is an array of LineStrings
				//fmt.Printf("\t%v\n", ring)
				points := make([]s2.Point, len(ring))

				for l, lineString := range ring { //LineString is an array of Points
					//fmt.Printf("\t\t%v\n", lineString)
					points[l] = s2.PointFromLatLng(s2.LatLngFromDegrees(lineString.Lat(), lineString.Lon()))
				}
				loops[r] = s2.LoopFromPoints(points)
			}
			polygonMap[country] = s2.PolygonFromLoops(loops)
		}
	}
	for k, v := range polygonMap {
		fmt.Printf("%s, %+v\n", k, v)
	}

	return &GeojsonAnimation{
		control:           control,
		featureCollection: fc,
		countries:         polygonMap,
	}
}




func (a *GeojsonAnimation) syncControl() {

}

func (a *GeojsonAnimation) frame(elapsed time.Duration, frameCount int) {
	a.syncControl()

	for _, p := range pixels.active {
		for name, country := range a.countries {
			if name == "Canada" && country.ContainsPoint(p.Point) {
				p.color = &c
			}
		}
	}
	//if frameCount%180 == 0 {
	//	a.lat = -90.0
	//}

	//aVar := a.control.GetVar("A")

	//if a.currentRow

}
