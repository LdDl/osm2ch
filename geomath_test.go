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
	mpt := middlePointSegment(p1, p2)
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

func TestFindCentroid(t *testing.T) {
	line := []GeoPoint{
		GeoPoint{Lon: 37.396747, Lat: 55.8321},
		GeoPoint{Lon: 37.397111, Lat: 55.831987},
		GeoPoint{Lon: 37.397222, Lat: 55.831927},
		GeoPoint{Lon: 37.397322, Lat: 55.831851},
		GeoPoint{Lon: 37.397384, Lat: 55.83177},
		GeoPoint{Lon: 37.397415, Lat: 55.831684},
		GeoPoint{Lon: 37.397407, Lat: 55.831605},
		GeoPoint{Lon: 37.397363, Lat: 55.831525},
		GeoPoint{Lon: 37.397283, Lat: 55.83144},
		GeoPoint{Lon: 37.39717, Lat: 55.831367},
		GeoPoint{Lon: 37.397001, Lat: 55.831313},
		GeoPoint{Lon: 37.39682, Lat: 55.831286},
		GeoPoint{Lon: 37.39662, Lat: 55.83129},
		GeoPoint{Lon: 37.396464, Lat: 55.831311},
		GeoPoint{Lon: 37.396345, Lat: 55.831346},
		GeoPoint{Lon: 37.396202, Lat: 55.83141},
		GeoPoint{Lon: 37.396123, Lat: 55.831459},
		GeoPoint{Lon: 37.396059, Lat: 55.831517},
		GeoPoint{Lon: 37.396013, Lat: 55.831591},
		GeoPoint{Lon: 37.395989, Lat: 55.831674},
	}
	centroid := findCentroid(line)
	correctCentroid := GeoPoint{Lon: 37.39680299905517, Lat: 55.83157265108678}
	if correctCentroid.Lon != centroid.Lon {
		t.Errorf("Correct centroid longitude should be %f, but got %f", correctCentroid.Lon, centroid.Lon)
	}
	if correctCentroid.Lat != centroid.Lat {
		t.Errorf("Correct centroid latitude should be %f, but got %f", correctCentroid.Lat, centroid.Lat)
	}
}

func TestFindMiddlePoint(t *testing.T) {
	line := []GeoPoint{
		GeoPoint{Lon: 37.396747, Lat: 55.8321},
		GeoPoint{Lon: 37.397111, Lat: 55.831987},
		GeoPoint{Lon: 37.397222, Lat: 55.831927},
		GeoPoint{Lon: 37.397322, Lat: 55.831851},
		GeoPoint{Lon: 37.397384, Lat: 55.83177},
		GeoPoint{Lon: 37.397415, Lat: 55.831684},
		GeoPoint{Lon: 37.397407, Lat: 55.831605},
		GeoPoint{Lon: 37.397363, Lat: 55.831525},
		GeoPoint{Lon: 37.397283, Lat: 55.83144},
		GeoPoint{Lon: 37.39717, Lat: 55.831367},
		GeoPoint{Lon: 37.397001, Lat: 55.831313},
		GeoPoint{Lon: 37.39682, Lat: 55.831286},
		GeoPoint{Lon: 37.39662, Lat: 55.83129},
		GeoPoint{Lon: 37.396464, Lat: 55.831311},
		GeoPoint{Lon: 37.396345, Lat: 55.831346},
		GeoPoint{Lon: 37.396202, Lat: 55.83141},
		GeoPoint{Lon: 37.396123, Lat: 55.831459},
		GeoPoint{Lon: 37.396059, Lat: 55.831517},
		GeoPoint{Lon: 37.396013, Lat: 55.831591},
		GeoPoint{Lon: 37.395989, Lat: 55.831674},
	}
	cutStart, middlePoint := findMiddlePoint(line)
	correctCutStart := 9
	correctMiddlePoint := GeoPoint{Lon: 37.39712087557048, Lat: 55.83135130343672}
	if correctMiddlePoint.Lon != middlePoint.Lon {
		t.Errorf("Correct middle point longitude should be %f, but got %f", correctMiddlePoint.Lon, middlePoint.Lon)
	}
	if correctMiddlePoint.Lat != middlePoint.Lat {
		t.Errorf("Correct middle point latitude should be %f, but got %f", correctMiddlePoint.Lat, middlePoint.Lat)
	}
	if cutStart != correctCutStart {
		t.Errorf("Middle point should be after %d-th point, not %d-th", correctCutStart, cutStart)
	}
}
