package osm2ch

import (
	"fmt"

	geojson "github.com/paulmach/go.geojson"
)

// PrepareGeoJSONLinestring returns GeoJSON representation of LineString
func PrepareGeoJSONLinestring(pts []GeoPoint) string {
	pts2d := make([][]float64, len(pts))
	for i := range pts {
		pts2d[i] = []float64{pts[i].Lon, pts[i].Lat}
	}
	b, err := geojson.NewLineStringGeometry(pts2d).MarshalJSON()
	if err != nil {
		fmt.Printf("Warning. Can not convert geometry to geojson format: %s", err.Error())
		return ""
	}
	return string(b)
}

// PrepareGeoJSONPoint returns GeoJSON representation of Point
func PrepareGeoJSONPoint(pt GeoPoint) string {
	b, err := geojson.NewPointGeometry([]float64{pt.Lon, pt.Lat}).MarshalJSON()
	if err != nil {
		fmt.Printf("Warning. Can not convert geometry to geojson format: %s", err.Error())
		return ""
	}
	return string(b)
}
