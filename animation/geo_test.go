package animation

import (
	"testing"

	"fmt"
	"github.com/golang/geo/s2"
	"github.com/stretchr/testify/assert"
)

//earth circum: 40,075 km

func TestGeo(t *testing.T) {
	//lat1, lon1 := 0.0, 0.0               //equator
	//lat2, lon2 := 33.942536, -118.408074 //LAX

	// Create Ellipsoid object with WGS84-ellipsoid,
	// angle units are degrees, distance units are meter.
	//geo1 := ellipsoid.Init("WGS84", ellipsoid.Degrees, ellipsoid.Kilometer, ellipsoid.LongitudeIsSymmetric, ellipsoid.BearingNotSymmetric)

	// Calculate the distance and bearing from SFO to LAX.
	//distance, bearing := geo1.To(lat1, lon1, lat2, lon2)
	//fmt.Printf("Distance = %v Bearing = %v\n", distance, bearing)

	// Calculate where you are when going from SFO in
	// direction 45.0 deg. for 20000 meters.
	//lat3, lon3 := geo1.At(lat1, lon1, 20000.0, 45.0)
	//fmt.Printf("lat3 = %v lon3 = %v\n", lat3, lon3)

	startPoint := s2.PointFromLatLng(s2.LatLngFromDegrees(0.0, 0.0))
	bearing := 360.0
	for distance := 5000.0; distance <= 40000.0; distance += 1000.0 {
		end := toPoint(startPoint, distance, bearing)
		endLatLon := s2.LatLngFromPoint(end)
		//lat2, lon2 = geo1.At(lat1, lon1, distance, bearing)
		fmt.Printf("distance: %3.2f: %3.2f, %3.2f\n", distance, endLatLon.Lat.Degrees(), endLatLon.Lng.Degrees())

	}
}

func TestFloat64Equal(t *testing.T) {
	assert.True(t, float64Equal(0.0, 0.0))
	assert.True(t, float64Equal(180.0, 180.0))
	assert.True(t, float64Equal(180.0, 180.0000001))

	assert.False(t, float64Equal(-180.0, 180.0))
	assert.False(t, float64Equal(-180.0, 180.001))
}

func TestReverseBearing(t *testing.T) {
	assert.Equal(t, 0.0, reverseBearing(180.0))
	assert.Equal(t, 180.0, reverseBearing(0.0))
	assert.Equal(t, 180.0, reverseBearing(360.0))
	assert.Equal(t, 270.0, reverseBearing(90.0))

}
