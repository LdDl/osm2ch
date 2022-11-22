package osm2ch

import (
	"fmt"
	"testing"
)

func TestParser(t *testing.T) {
	parser := NewParser(
		WithFilename("sample.osm"),
		WithPreparePOI(true),
		WithStrictMode(true),
		WithDefaultLanes(nil),
		WithDefaultSpeed(nil),
	)

	t.Log(parser)

	verbose := true

	osmDataRaw, err := readOSM("./sample.osm", verbose)
	if err != nil {
		t.Error(err)
	}

	osmDataMedium, err := osmDataRaw.prepareWaysMedium(DEFAULT_FIRST_VERTEX, DEFAULT_FIRST_EDGE, verbose)
	if err != nil {
		t.Error(err)
	}
	for _, way := range osmDataMedium.ways {
		fmt.Println(way.id, way.osmID, way.linkClass, way.linkType, way.linkConnectionType, way.wasOneWay, way.lanesNum, way.freeSpeed, way.maxSpeed, way.capacity)
	}
}
