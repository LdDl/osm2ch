package osm2ch

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkt"
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

func TestRadiusСurvatureLine(t *testing.T) {
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
	r := calcRadiusCurvature(line)
	correctR := 47.22026299802612925305
	if (correctR - r) > 10e-9 {
		t.Errorf("Correct radius of curve should be %f, but got %f", correctR, r)
	}
}

func lineAsString(l orb.LineString) string {
	agg := []string{}
	for _, pt := range l {
		agg = append(agg, fmt.Sprintf("[%f, %f]", pt.X(), pt.Y()))
	}
	return "[" + strings.Join(agg, ",") + "]"
}

func TestOffset(t *testing.T) {
	line := orb.LineString{{10.0, 10.0}, {15.0, 10.0}, {18.0, 15.0}, {18.0, 20.0}, {15.0, 24.0}, {12.0, 24.0}, {10.0, 18.0}, {10.0, 15.0}, {13.0, 12.0}, {15.0, 16.0}}
	distance := 1.0

	leftL := lineAsString(offsetCurve(line, distance))
	rightL := lineAsString(offsetCurve(line, -distance))

	correctLeft := "[[10.000000, 11.000000],[14.433810, 11.000000],[17.000000, 15.276984],[17.000000, 19.666667],[14.500000, 23.000000],[12.720759, 23.000000],[11.000000, 17.837722],[11.000000, 15.414214],[12.726049, 13.688165],[14.105573, 16.447214]]"
	if leftL != correctLeft {
		t.Errorf("Left offset line should be '%s' but got '%s'", correctLeft, leftL)
	}
	correctRight := "[[10.000000, 9.000000],[15.566190, 9.000000],[19.000000, 14.723016],[19.000000, 20.333333],[15.500000, 25.000000],[11.279241, 25.000000],[9.000000, 18.162278],[9.000000, 14.585786],[13.273951, 10.311835],[15.894427, 15.552786]]"
	if rightL != correctRight {
		t.Errorf("Right offset line should be '%s' but got '%s'", correctRight, rightL)
	}
}

func findDist(p1, p2 orb.Point) float64 {
	return math.Sqrt(math.Pow(p2.X()-p1.X(), 2) + math.Pow(p2.Y()-p1.Y(), 2))
}

func rotateVector(vec orb.Point, angle float64) orb.Point {
	rad := deg2rad(angle)
	return orb.Point{
		vec[0]*math.Cos(rad) - vec[1]*math.Sin(rad),
		vec[0]*math.Sin(rad) + vec[1]*math.Cos(rad),
	}
}

const (
	d2r = math.Pi / 180.0
)

func deg2rad(deg float64) float64 {
	return deg * d2r
}

func TestLineSubstring(t *testing.T) {
	lineWKT := "LINESTRING (37.56319128200903 55.78357465483572, 37.565235359279626 55.78497472894253, 37.565822487858156 55.785421030200496, 37.567355545810614 55.784711836767826)"
	line, err := wkt.UnmarshalLineString(lineWKT)
	if err != nil {
		t.Error(err)
		return
	}
	newline := SubstringHaversine(line, 215, 278)
	newLineWKT := wkt.MarshalString(newline)
	correctLine := "LINESTRING(37.56536219999623 55.78507114703719,37.565822487858156 55.785421030200496,37.56600203415945 55.785337974305975)"
	if correctLine != newLineWKT {
		t.Errorf("Correct line should be '%s', but got '%s'", correctLine, newLineWKT)
	}
}
