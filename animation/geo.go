package animation

import (
	"github.com/StefanSchroeder/Golang-Ellipsoid/ellipsoid"
	"github.com/golang/geo/s2"
)

var (
	// Create Ellipsoid object with WGS84-ellipsoid,
	geo = ellipsoid.Init(
		"WGS84",
		ellipsoid.Degrees,
		ellipsoid.Kilometer,
		ellipsoid.LongitudeIsSymmetric,
		ellipsoid.BearingNotSymmetric)
)

func toPoint(start s2.Point, distance, bearing float64) s2.Point {
	startLatLong := s2.LatLngFromPoint(start)
	lat, lon := geo.At(startLatLong.Lat.Degrees(), startLatLong.Lng.Degrees(), distance, bearing)
	return s2.PointFromLatLng(s2.LatLngFromDegrees(lat, lon))
}
