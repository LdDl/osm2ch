package osm2ch

import (
	"testing"
)

func TestMiddlePoint(t *testing.T) {
	p1 := GeoPoint{
		Lon: 37.6417350769043,
		Lat: 55.751849391735284,
	}
	p2 := GeoPoint{
		Lon: 37.668514251708984,
		Lat: 55.73261980350401,
	}
	res := GeoPoint{
		Lon: 37.65512796336629,
		Lat: 55.742235325526806,
	}
	mpt := middlePoint(p1, p2)
	if mpt != res {
		t.Errorf("Middle point must be %v, but got %v", res, mpt)
	}
}

func TestGreatCircleDistance(t *testing.T) {
	p1 := GeoPoint{
		Lon: 37.6417350769043,
		Lat: 55.751849391735284,
	}
	p2 := GeoPoint{
		Lon: 37.668514251708984,
		Lat: 55.73261980350401,
	}
	res := 2.71693096539 // kilometers
	gcd := greatCircleDistance(p1, p2)
	if Round(gcd, 0.0005) != Round(res, 0.0005) {
		t.Errorf("Great circle dist must be %f, but got %f", res, gcd)
	}
}

func Round(x, unit float64) float64 {
	if x > 0 {
		return float64(int64(x/unit+0.5)) * unit
	}
	return float64(int64(x/unit-0.5)) * unit
}
