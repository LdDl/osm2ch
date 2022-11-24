package osm2ch

import (
	"testing"
)

func TestParser(t *testing.T) {
	parser := NewParser(
		"./sample.osm",
		WithPreparePOI(false),
		WithStrictMode(false),
	)
	t.Log(parser)
	verbose := true
	err := parser.createNetwork(verbose)
	if err != nil {
		t.Error(err)
		return
	}
}
