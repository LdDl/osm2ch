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

	netMacro, err := parser.createNetwork(verbose)
	if err != nil {
		t.Error(err)
		return
	}
	netMacro.genActivityType()
	netMacro.genMovement(verbose)
	netMeso, err := netMacro.genMesoscopicNetwork(verbose)
	if err != nil {
		t.Error(err)
		return
	}
	err = netMacro.ExportToCSV("network")
	if err != nil {
		t.Error(err)
		return
	}
	err = netMeso.ExportToCSV("network")
	if err != nil {
		t.Error(err)
		return
	}
}
