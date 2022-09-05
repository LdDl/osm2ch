package osm2ch

import (
	"fmt"
	"strings"
)

// PrepareWKTLinestring returns WKT representation of LineString
func PrepareWKTLinestring(pts []GeoPoint) string {
	ptsStr := make([]string, len(pts))
	for i := range pts {
		ptsStr[i] = fmt.Sprintf("%f %f", pts[i].Lon, pts[i].Lat)
	}
	return fmt.Sprintf("LINESTRING(%s)", strings.Join(ptsStr, ","))
}

// PrepareWKTPoint returns WKT representation of Point
func PrepareWKTPoint(pt GeoPoint) string {
	return fmt.Sprintf("POINT(%f %f)", pt.Lon, pt.Lat)
}
