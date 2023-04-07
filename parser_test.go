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
	enablePOI := false

	/* Macroscopic */
	netMacro, err := parser.createNetwork(verbose, enablePOI)
	if err != nil {
		t.Error(err)
		return
	}
	netMacro.genActivityType()
	netMacro.genMovement(verbose)

	/* Mesoscopic */
	netMeso, err := netMacro.genMesoscopicNetwork(verbose)
	if err != nil {
		t.Error(err)
		return
	}

	/* Microscopic */
	netMicro, err := genMicroscopicNetwork(netMacro, netMeso, false, verbose)
	if err != nil {
		t.Error(err)
		return
	}

	outFile := "network"
	err = netMacro.ExportToCSV(outFile)
	if err != nil {
		t.Error(err)
		return
	}
	err = netMeso.ExportToCSV(outFile)
	if err != nil {
		t.Error(err)
		return
	}
	err = netMicro.ExportToCSV(outFile)
	if err != nil {
		t.Error(err)
		return
	}
}
