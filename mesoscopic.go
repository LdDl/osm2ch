package osm2ch

import (
	"fmt"
	"sort"
	"time"

	"github.com/paulmach/orb"
)

type NetworkMesoscopic struct {
}

const (
	resolution = 5.0
)

func (net *NetworkMacroscopic) genMesoscopicNetwork(verbose bool) (*NetworkMesoscopic, error) {
	if verbose {
		fmt.Print("Preparing mesocopic...")
	}
	st := time.Now()
	mesoscopic := NetworkMesoscopic{}

	/* Prepare segments */
	for _, link := range net.links {
		breakpoints := []float64{0, link.lengthMeters}
		if link.lengthMeters <= resolution {
			link.breakpoints = make([]float64, len(breakpoints))
		} else {
			for len(breakpoints) != 0 {
				target := breakpoints[0]
				remove := make(map[int]struct{})
				for idx, point := range breakpoints {
					if target-resolution <= point && point <= target+resolution {
						remove[idx] = struct{}{}
					}
				}
				link.breakpoints = append(link.breakpoints, target)
				for idx := range remove {
					breakpoints = append(breakpoints[:idx], breakpoints[idx+1:]...)
				}
			}
			sort.Float64s(link.breakpoints)
		}
		lanes := link.GetLanes()
		link.lanesList = make([]int, 0, len(link.breakpoints)-1)
		for i := 0; i < len(link.breakpoints)-1; i++ {
			link.lanesList = append(link.lanesList, lanes)
			link.lanesChange = append(link.lanesChange, []int{0.0, 0.0})
		}
	}
	/* */

	/* Offset geometies */
	observed := make(map[NetworkLinkID]bool)
	links := linksToSlice(net.links)
	for i, linkID := range links {
		link, ok := net.links[linkID]
		if !ok {
			return nil, fmt.Errorf("Link %d not found. Should not happen", linkID)
		}
		if _, ok := observed[linkID]; ok {
			continue
		}
		reversedGeom := link.geomEuclidean.Clone()
		reversedGeom.Reverse()
		reversedLinkExists := false
		for _, linkCompareID := range links[i+1:] {
			linkCompare, ok := net.links[linkCompareID]
			if !ok {
				return nil, fmt.Errorf("Link %d not found. Should not happen", linkID)
			}
			if orb.Equal(reversedGeom, linkCompare.geomEuclidean) {
				reversedLinkExists = true
				observed[linkID] = true
				observed[linkCompareID] = true
				break
			}
		}
		if !reversedLinkExists {
			observed[linkID] = false
		}
	}
	/* */

	/* Process movements */
	// @todo
	/* */

	/* Process macro links */
	// @todo
	/* */

	/* Gen movement (if needed) */
	// @todo
	/* */

	/* Build meso/micro */
	// @todo
	/* */

	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}
	return &mesoscopic, nil
}

func linksToSlice(links map[NetworkLinkID]*NetworkLink) []NetworkLinkID {
	ans := make([]NetworkLinkID, 0, len(links))
	for k := range links {
		ans = append(ans, k)
	}
	return ans
}
