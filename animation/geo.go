package animation

import (
	"github.com/StefanSchroeder/Golang-Ellipsoid/ellipsoid"
	"github.com/golang/geo/s2"
	"math"
)

const (
	minVisibleLatitude = -48.75 //all points south of here don't have any pixels associated with them.
	latitudeRange      = 90.0 + 48.75
	epsilon            = 0.00001
)

var (
	// Create Ellipsoid object with WGS84-ellipsoid,
	geo = ellipsoid.Init(
		"WGS84",
		ellipsoid.Degrees,
		ellipsoid.Kilometer,
		ellipsoid.LongitudeIsSymmetric,
		ellipsoid.BearingNotSymmetric)

	NorthPole          = s2.PointFromLatLng(s2.LatLngFromDegrees(90.0, 0.0))
	SouthPole          = s2.PointFromLatLng(s2.LatLngFromDegrees(-90.0, 0.0))
	EquatorMeridian    = s2.PointFromLatLng(s2.LatLngFromDegrees(0.0, 0.0))
	EquatorNonMeridian = s2.PointFromLatLng(s2.LatLngFromDegrees(0.0, 180.0))
)

func toPoint(start s2.Point, distance, bearing float64) s2.Point {
	startLatLong := s2.LatLngFromPoint(start)
	lat, lon := geo.At(startLatLong.Lat.Degrees(), startLatLong.Lng.Degrees(), distance, bearing)
	return s2.PointFromLatLng(s2.LatLngFromDegrees(lat, lon))
}

func float64Equal(a, b float64) bool {
	if a == b {
		return true
	}
	diff := math.Abs(a-b) / math.Abs(a)
	return diff < epsilon
}

func reverseBearing(bearing float64) float64 {
	if bearing >= 180.0 {
		return bearing - 180.0
	}
	newBearing := bearing + 180.0
	if bearing > 360.0 {
		return bearing - 180.0
	}
	return newBearing
}
