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

	osmData, err := readOSM("./sample.osm", verbose)
	if err != nil {
		t.Error(err)
	}

	edgesMacro, err := osmData.prepareEdgesMacro(DEFAULT_FIRST_VERTEX, DEFAULT_FIRST_EDGE, verbose)
	if err != nil {
		t.Error(err)
	}
	for _, edge := range edgesMacro {
		fmt.Println(edge.id, edge.osmID, edge.linkClass, edge.linkType, edge.linkConnectionType, edge.wasOneWay, edge.lanesNum, edge.freeSpeed, edge.maxSpeed, edge.capacity)
	}
}
