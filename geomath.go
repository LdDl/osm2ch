package osm2ch

import (
	"fmt"
	"math"

	"github.com/paulmach/orb"
)

const (
	earthRadius = 6370.986884258304
	pi180       = math.Pi / 180.0
	pi180Rev    = 180.0 / math.Pi
)

// GeoPoint representation of point on Earth
type GeoPoint struct {
	Lat float64
	Lon float64
}

// String returns pretty printed value for for GeoPoint
func (gp GeoPoint) String() string {
	return fmt.Sprintf("Lon: %f | Lat: %f", gp.Lon, gp.Lat)
}

// calcRadiusCurvature returns radius of curvature for given line (in meters)
func calcRadiusCurvature(line []GeoPoint) float64 {
	var rs float64
	for i := 1; i < len(line)-1; i++ {
		a := greatCircleDistance(line[i-1], line[i])
		b := greatCircleDistance(line[i], line[i+1])
		c := greatCircleDistance(line[i-1], line[i+1])
		p := (a + b + c) / 2
		s := math.Sqrt(p * (p - a) * (p - b) * (p - c))
		r := (a * b * c) / (4 * s)
		rs += r
	}
	rs = 1000 * rs / float64(len(line)-2)
	return rs
}

// degreesToRadians deg = r * pi / 180
func degreesToRadians(d float64) float64 {
	return d * pi180
}

// radiansTodegrees r = deg  * 180 / pi
func radiansTodegrees(d float64) float64 {
	return d * pi180Rev
}

// greatCircleDistance returns distance between two geo-points (kilometers)
func greatCircleDistance(p, q GeoPoint) float64 {
	lat1 := degreesToRadians(p.Lat)
	lon1 := degreesToRadians(p.Lon)
	lat2 := degreesToRadians(q.Lat)
	lon2 := degreesToRadians(q.Lon)
	diffLat := lat2 - lat1
	diffLon := lon2 - lon1
	a := math.Pow(math.Sin(diffLat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(diffLon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	ans := c * earthRadius
	return ans
}

// getSphericalLength returns length for given line (kilometers)
func getSphericalLength(line []GeoPoint) float64 {
	totalLength := 0.0
	if len(line) < 2 {
		return totalLength
	}
	for i := 1; i < len(line); i++ {
		totalLength += greatCircleDistance(line[i-1], line[i])
	}
	return totalLength
}

// middlePointSegment return middle point for given segment
func middlePointSegment(p, q GeoPoint) GeoPoint {
	lat1 := degreesToRadians(p.Lat)
	lon1 := degreesToRadians(p.Lon)
	lat2 := degreesToRadians(q.Lat)
	lon2 := degreesToRadians(q.Lon)

	Bx := math.Cos(lat2) * math.Cos(lon2-lon1)
	By := math.Cos(lat2) * math.Sin(lon2-lon1)

	latMid := math.Atan2(math.Sin(lat1)+math.Sin(lat2), math.Sqrt((math.Cos(lat1)+Bx)*(math.Cos(lat1)+Bx)+By*By))
	lonMid := lon1 + math.Atan2(By, math.Cos(lat1)+Bx)
	return GeoPoint{Lat: radiansTodegrees(latMid), Lon: radiansTodegrees(lonMid)}
}

// findCentroid returns center point for given line (not middle point)
func findCentroid(line []GeoPoint) GeoPoint {
	totalPoints := len(line)
	if totalPoints == 1 {
		return line[0]
	}
	x, y, z := 0.0, 0.0, 0.0
	for i := 0; i < totalPoints; i++ {
		longitude := degreesToRadians(line[i].Lon)
		latitude := degreesToRadians(line[i].Lat)
		c1 := math.Cos(latitude)
		x += c1 * math.Cos(longitude)
		y += c1 * math.Sin(longitude)
		z += math.Sin(latitude)
	}

	x /= float64(totalPoints)
	y /= float64(totalPoints)
	z /= float64(totalPoints)

	centralLongitude := math.Atan2(y, x)
	centralSquareRoot := math.Sqrt(x*x + y*y)
	centralLatitude := math.Atan2(z, centralSquareRoot)

	return GeoPoint{
		Lon: radiansTodegrees(centralLongitude),
		Lat: radiansTodegrees(centralLatitude),
	}
}

// findDistance returns distance between two points (assuming they are Euclidean: Lon == X, Lat == Y)
func findDistance(p, q GeoPoint) float64 {
	xdistance := p.Lon - q.Lon
	ydistance := p.Lat - q.Lat
	return math.Sqrt(xdistance*xdistance + ydistance*ydistance)
}

// Check if two segments intersects and returns intersections Point
// p1, p2 - first segment
// p3, p4 - second segment
// Note: Euclidean space
func intersect(p1, p2, p3, p4 orb.Point) (orb.Point, error) {
	// Calculate the coefficients of the linear equations
	a1 := p2[1] - p1[1]
	b1 := p1[0] - p2[0]
	c1 := a1*p1[0] + b1*p1[1]
	a2 := p4[1] - p3[1]
	b2 := p3[0] - p4[0]
	c2 := a2*p3[0] + b2*p3[1]

	// Calculate the determinant
	det := a1*b2 - a2*b1
	if det == 0 {
		return orb.Point{}, fmt.Errorf("The lines are parallel")
	}

	// Calculate the intersection point
	x := (b2*c1 - b1*c2) / det
	y := (a1*c2 - a2*c1) / det
	return orb.Point{x, y}, nil
}

func offsetCurve(line orb.LineString, distance float64) orb.LineString {
	// Initialize result list and segment list
	var result orb.LineString
	var segments [][2]orb.Point

	// Iterate over line segments and calculate offset segments
	for i := 1; i < len(line); i++ {
		// Get current and previous points
		p1 := line[i-1]
		p2 := line[i]

		// Calculate the vector between the points
		vec := [2]float64{p2[0] - p1[0], p2[1] - p1[1]}

		// Normalize the vector
		vecLen := math.Sqrt(vec[0]*vec[0] + vec[1]*vec[1])
		vec = [2]float64{vec[0] / vecLen, vec[1] / vecLen}

		// Rotate the vector by 90 degrees
		rotated := [2]float64{-vec[1], vec[0]}

		// Scale the rotated vector by the distance
		offset := [2]float64{rotated[0] * distance, rotated[1] * distance}

		// Calculate the offset points
		op1 := [2]float64{p1[0] + offset[0], p1[1] + offset[1]}
		op2 := [2]float64{p2[0] + offset[0], p2[1] + offset[1]}

		// Add the offset segment to the list of segments
		segments = append(segments, [2]orb.Point{op1, op2})
	}

	result = append(result, segments[0][0])
	// Iterate over the segments and calculate the intersections
	for i := 1; i < len(segments); i++ {
		// Get the current and previous segments
		seg1 := segments[i-1]
		seg2 := segments[i]
		// Calculate the intersection point
		intersection, err := intersect(seg1[0], seg1[1], seg2[0], seg2[1])
		if err != nil {
			continue
		}
		// If there is an intersection, add the intersection and the current segment to the result
		result = append(result, intersection)
	}
	result = append(result, segments[len(segments)-1][1])
	return result
}

// getLength returns length for given line  (assuming points of the line are Euclidean: Lon == X, Lat == Y)
func getLength(line []GeoPoint) float64 {
	totalLength := 0.0
	if len(line) < 2 {
		return totalLength
	}
	for i := 1; i < len(line); i++ {
		totalLength += findDistance(line[i-1], line[i])
	}
	return totalLength
}

// findMiddlePoint returns middle point for give line (not center point) and index of point in line right before middle one
// Purpose of returning index of point in line right before middle point is to give the ability to split line in a half
func findMiddlePoint(line []GeoPoint) (int, GeoPoint) {
	euclideanLength := getLength(line)
	halfDistance := euclideanLength / 2.0
	cl := 0.0
	ol := 0.0
	var result GeoPoint
	var idx int
	for i := 1; i < len(line); i++ {
		ol = cl
		tmpDist := findDistance(line[i-1], line[i])
		cl += tmpDist
		if halfDistance <= cl && halfDistance > ol {
			halfSub := halfDistance - ol
			result = pointOnSegmentByFraction(line[i-1], line[i], halfSub/tmpDist, halfSub)
			idx = i - 1
		}
	}
	return idx, result
}

// pointOnSegment returns a point on given segment using distance
func pointOnSegment(p, q GeoPoint, distance float64) GeoPoint {
	fraction := distance / findDistance(p, q)
	return GeoPoint{
		Lon: (1-fraction)*p.Lon + (fraction * q.Lon),
		Lat: (1-fraction)*p.Lat + (fraction * q.Lat),
	}
}

// pointOnSegmentByFraction returns a point on given segment using distance assuming knowledge about fraction
func pointOnSegmentByFraction(p, q GeoPoint, fraction, distance float64) GeoPoint {
	return GeoPoint{
		Lon: (1-fraction)*p.Lon + (fraction * q.Lon),
		Lat: (1-fraction)*p.Lat + (fraction * q.Lat),
	}
}

// reverseLine reverses order of points in given line. Returns new slice
func reverseLine(pts []GeoPoint) []GeoPoint {
	inputLen := len(pts)
	output := make([]GeoPoint, inputLen)
	for i, n := range pts {
		j := inputLen - i - 1
		output[j] = n
	}
	return output
}

// copyLine reverses order of points in given line. Returns new slice
func copyLine(pts []GeoPoint) []GeoPoint {
	inputLen := len(pts)
	output := make([]GeoPoint, inputLen)
	for i, n := range pts {
		output[i] = n
	}
	return output
}

// reverseLine reverses order of points in given line
func reverseLineInPlace(pts []GeoPoint) {
	inputLen := len(pts)
	inputMid := inputLen / 2
	for i := 0; i < inputMid; i++ {
		j := inputLen - i - 1
		pts[i], pts[j] = pts[j], pts[i]
	}
}
