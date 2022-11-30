package osm2ch

import (
	"math"

	"github.com/paulmach/orb"
)

const (
	earthR = 20037508.34
)

func epsg3857To4326(lat, lng float64) (float64, float64) {
	newLat := lat * 180 / earthR
	newLong := math.Atan(math.Exp(lng*math.Pi/earthR))*360/math.Pi - 90
	return newLat, newLong
}

func epsg4326To3857(lon, lat float64) (float64, float64) {
	x := lon * earthR / 180
	y := math.Log(math.Tan((90+lat)*math.Pi/360)) / (math.Pi / 180)
	y = y * earthR / 180
	return x, y
}

func pointToEuclidean(pt orb.Point) orb.Point {
	euclideanX, euclideanY := epsg4326To3857(pt.Lon(), pt.Lat())
	return orb.Point{euclideanX, euclideanY}
}

func lineToEuclidean(line orb.LineString) orb.LineString {
	newLine := make(orb.LineString, len(line))
	for i, pt := range line {
		newLine[i] = pointToEuclidean(pt)
	}
	return newLine
}

// angleBetweenLines returs angle between two lines
//
// Note: panics if number of points in any line is less than 2
//
func angleBetweenLines(l1 orb.LineString, l2 orb.LineString) float64 {
	angle1 := math.Atan2(l1[len(l1)-1].Y()-l1[0].Y(), l1[len(l1)-1].X()-l1[0].X())
	angle2 := math.Atan2(l2[len(l2)-1].Y()-l2[0].Y(), l2[len(l2)-1].X()-l2[0].X())
	angle := angle2 - angle1
	if angle < -1*math.Pi {
		angle += 2 * math.Pi
	}
	if angle > math.Pi {
		angle -= 2 * math.Pi
	}
	return angle
}
