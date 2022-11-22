package osm2ch

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
)

var (
	defaultLanesByLinkType = map[LinkType]int{
		LINK_MOTORWAY:     4,
		LINK_TRUNK:        3,
		LINK_PRIMARY:      3,
		LINK_SECONDARY:    2,
		LINK_TERTIARY:     2,
		LINK_RESIDENTIAL:  1,
		LINK_SERVICE:      1,
		LINK_CYCLEWAY:     1,
		LINK_FOOTWAY:      1,
		LINK_TRACK:        1,
		LINK_UNCLASSIFIED: 1,
		LINK_CONNECTOR:    2,
	}
	defaultSpeedByLinkType = map[LinkType]float64{
		LINK_MOTORWAY:     120,
		LINK_TRUNK:        100,
		LINK_PRIMARY:      80,
		LINK_SECONDARY:    60,
		LINK_TERTIARY:     40,
		LINK_RESIDENTIAL:  30,
		LINK_SERVICE:      30,
		LINK_CYCLEWAY:     5,
		LINK_FOOTWAY:      5,
		LINK_TRACK:        30,
		LINK_UNCLASSIFIED: 30,
		LINK_CONNECTOR:    120,
	}
	defaultCapacityByLinkType = map[LinkType]int{
		LINK_MOTORWAY:     2300,
		LINK_TRUNK:        2200,
		LINK_PRIMARY:      1800,
		LINK_SECONDARY:    1600,
		LINK_TERTIARY:     1200,
		LINK_RESIDENTIAL:  1000,
		LINK_SERVICE:      800,
		LINK_CYCLEWAY:     800,
		LINK_FOOTWAY:      800,
		LINK_TRACK:        800,
		LINK_UNCLASSIFIED: 800,
		LINK_CONNECTOR:    9999,
	}
)

func (data *OSMDataRaw) prepareWellDone(verbose bool) error {
	err := data.prepareWaysWellDone(verbose)
	if err != nil {
		return errors.Wrap(err, "Can't preprocess ways")
	}
	return nil
}

func (data *OSMDataRaw) prepareWaysWellDone(verbose bool) error {
	if verbose {
		fmt.Printf("Cook well-done ways...")
	}
	st := time.Now()
	for _, way := range data.waysMedium {
		if way.capacity < 0 {
			if defaultCap, ok := defaultCapacityByLinkType[way.linkType]; ok {
				way.capacity = defaultCap
			}
		}
		if way.freeSpeed < 0 {
			if way.maxSpeed >= 0 {
				way.freeSpeed = way.maxSpeed
			} else {
				if defaultSpeed, ok := defaultSpeedByLinkType[way.linkType]; ok {
					way.freeSpeed = defaultSpeed
					way.maxSpeed = defaultSpeed
				}
			}
		}
		// Find and mark pure cycles
		if way.isCycle {
			way.isPureCycle = true
			for _, nodeID := range way.Nodes {
				if _, ok := data.nodes[nodeID]; !ok {
					return fmt.Errorf("No such node '%d'. Way ID: '%d'", nodeID, way.ID)
				}
				if data.nodes[nodeID].isCrossing {
					way.isPureCycle = false
				}
			}
		}
	}
	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}
	return nil
}
