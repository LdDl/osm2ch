package osm2ch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/meso"
	"github.com/LdDl/osm2gmns/osmmacro"
	"github.com/LdDl/osm2gmns/osmmeso"
	"github.com/LdDl/osm2gmns/osmmicro"
	"github.com/LdDl/osm2gmns/osmmovement"
)

// GMNSLevel enumerates the GMNS network levels that can be exported.
type GMNSLevel string

const (
	GMNSMacro    GMNSLevel = "macro"
	GMNSMovement GMNSLevel = "movement"
	GMNSMeso     GMNSLevel = "meso"
	GMNSMicro    GMNSLevel = "micro"
)

// ParseGMNSLevels parses a comma-separated list of GMNS network levels (e.g.
// "macro,movement,meso,micro"). Unknown tokens are reported and skipped. The
// returned slice is empty when nothing valid was requested.
func ParseGMNSLevels(s string) []GMNSLevel {
	parts := strings.Split(s, ",")
	levels := make([]GMNSLevel, 0, len(parts))
	for _, p := range parts {
		switch GMNSLevel(strings.TrimSpace(strings.ToLower(p))) {
		case GMNSMacro:
			levels = append(levels, GMNSMacro)
		case GMNSMovement:
			levels = append(levels, GMNSMovement)
		case GMNSMeso:
			levels = append(levels, GMNSMeso)
		case GMNSMicro:
			levels = append(levels, GMNSMicro)
		case "":
			// ignore empty tokens (e.g. trailing comma or empty flag)
		default:
			fmt.Printf("Warning: unknown GMNS network level '%s', skipping\n", strings.TrimSpace(p))
		}
	}
	return levels
}

// ExportGMNS writes the requested GMNS network levels as CSV files into outputDir.
//
// The macro network and movements come straight from nets. The meso network is
// generated on demand (it is required by both the "meso" and "micro" levels), and
// the micro network is generated from the macro + meso networks and movements.
//
// Produced files (osm2gmns naming convention):
//   - macro:    node.csv, link.csv
//   - movement: movement.csv
//   - meso:     mesonode.csv, mesolink.csv
//   - micro:    micronode.csv, microlink.csv
func ExportGMNS(nets *GMNSNetworks, levels []GMNSLevel, outputDir string) error {
	if len(levels) == 0 {
		return nil
	}
	if nets == nil || nets.Macro == nil {
		return fmt.Errorf("no GMNS networks available to export")
	}

	want := make(map[GMNSLevel]bool, len(levels))
	for _, l := range levels {
		want[l] = true
	}

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return fmt.Errorf("can't create output directory %q: %w", outputDir, err)
		}
	}

	if want[GMNSMacro] {
		fmt.Println("Exporting GMNS macro network...")
		if err := osmmacro.ExportToCSV(nets.Macro, outputDir); err != nil {
			return fmt.Errorf("export macro network: %w", err)
		}
	}

	if want[GMNSMovement] {
		fmt.Println("Exporting GMNS movements...")
		if err := osmmovement.ExportToCSV(nets.Movements, filepath.Join(outputDir, "movement.csv")); err != nil {
			return fmt.Errorf("export movements: %w", err)
		}
	}

	// meso is needed for both the meso and micro levels
	var mesoNet *meso.Net
	if want[GMNSMeso] || want[GMNSMicro] {
		fmt.Println("Generating GMNS mesoscopic network...")
		var err error
		mesoNet, err = generators.GenerateMesoscopic(nets.Macro, nets.Movements)
		if err != nil {
			return fmt.Errorf("generate meso network: %w", err)
		}
	}

	if want[GMNSMeso] {
		fmt.Println("Exporting GMNS meso network...")
		if err := osmmeso.ExportToCSV(mesoNet, outputDir); err != nil {
			return fmt.Errorf("export meso network: %w", err)
		}
	}

	if want[GMNSMicro] {
		fmt.Println("Generating GMNS microscopic network...")
		microNet, err := generators.GenerateMicroscopic(nets.Macro, mesoNet, nets.Movements)
		if err != nil {
			return fmt.Errorf("generate micro network: %w", err)
		}
		fmt.Println("Exporting GMNS micro network...")
		if err := osmmicro.ExportToCSV(microNet, outputDir); err != nil {
			return fmt.Errorf("export micro network: %w", err)
		}
	}

	return nil
}
