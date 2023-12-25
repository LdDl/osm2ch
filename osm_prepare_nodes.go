package osm2ch

import (
	"fmt"
	"time"
)

func (data *OSMDataRaw) prepareNodes(verbose bool) error {
	if verbose {
		fmt.Printf("Prepare nodes...")
	}
	st := time.Now()
	for nodeID, nodeMedium := range data.nodes {
		if nodeMedium.useCount >= 2 || nodeMedium.controlType == IS_SIGNAL {
			data.nodes[nodeID].isCrossing = true
		}
	}
	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}
	return nil
}
