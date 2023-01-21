package osm2ch

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
)

type NetworkMesoscopic struct {
}

const (
	resolution  = 5.0
	laneWidth   = 3.5
	shortcutLen = 0.1
	cutLenMin   = 2.0
)

var (
	cutLen = [100]float64{2.0, 8.0, 12.0, 14.0, 16.0, 18.0, 20, 22, 24, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25}
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
			return nil, fmt.Errorf("Link %d not found. Should not happen [Loop over all links]", linkID)
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
				return nil, fmt.Errorf("Link %d not found. Should not happen [Loop over remaining links]", linkID)
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
	for linkID, needOffset := range observed {
		link, ok := net.links[linkID]
		if !ok {
			return nil, fmt.Errorf("Link %d not found. Should not happen [Loop over observed links]", linkID)
		}
		if needOffset {
			offsetDistance := 2 * (float64(link.MaxLanes())/2 + 0.5) * laneWidth
			geomEuclidean := link.geomEuclidean.Clone()
			offsetGeom := offsetCurve(geomEuclidean, offsetDistance) // Use "-" sign to make offset to the right side
			link.geomEuclideanOffset = offsetGeom.Clone()
			link.geomOffset = lineToSpherical(link.geomEuclideanOffset)
			continue
		}
		link.geomOffset = link.geom.Clone()
		link.geomEuclideanOffset = link.geomEuclidean.Clone()
	}

	// Update breakpoints since geometry has changed
	for _, link := range net.links {
		// Re-calcuate length for offset geometry and round to 2 decimal places
		link.lengthMetersOffset = math.Round(geo.LengthHaversign(link.geomOffset)*100.0) / 100.0
		for i, item := range link.breakpoints {
			link.breakpoints[i] = (item / link.lengthMeters) * link.lengthMetersOffset
		}
	}
	/* */

	/* Process movements */
	for _, node := range net.nodes {
		if node.controlType == IS_SIGNAL {
			continue
		}
		if len(node.incomingLinks) == 1 && len(node.outcomingLinks) >= 1 {
			// Only one incoming link
			observed := make(map[NetworkLinkID]struct{})
			multipleConnections := false
			for _, movement := range node.movements {
				if _, ok := observed[movement.OutcomingLinkID]; ok {
					multipleConnections = true
					break
				} else {
					observed[movement.OutcomingLinkID] = struct{}{}
				}
			}
			if multipleConnections {
				continue
			}
			node.movementIsNeeded = false
			linkID := node.incomingLinks[0]
			if link, ok := net.links[linkID]; ok {
				link.downstreamShortCut = true
				link.downstreamIsTarget = true
				for _, outcomingLinkID := range node.outcomingLinks {
					if outcomingLink, ok := net.links[outcomingLinkID]; ok {
						outcomingLink.upstreamShortCut = true
					} else {
						return nil, fmt.Errorf("nested outcoming link %d not found. Should not happen [Process movements]", linkID)
					}
				}
			} else {
				return nil, fmt.Errorf("incoming link %d not found. Should not happen [Process movements]", linkID)
			}
		} else if len(node.outcomingLinks) == 1 && len(node.incomingLinks) >= 1 {
			// Only one outcoming link
			observed := make(map[NetworkLinkID]struct{})
			multipleConnections := false
			for _, movement := range node.movements {
				if _, ok := observed[movement.IncomingLinkID]; ok {
					multipleConnections = true
					break
				} else {
					observed[movement.IncomingLinkID] = struct{}{}
				}
			}
			if multipleConnections {
				continue
			}
			node.movementIsNeeded = false
			linkID := node.outcomingLinks[0]
			if link, ok := net.links[linkID]; ok {
				link.upstreamShortCut = true
				link.upstreamIsTarget = true
				for _, incomingLinkID := range node.incomingLinks {
					if incomingLink, ok := net.links[incomingLinkID]; ok {
						incomingLink.downstreamShortCut = true
					} else {
						return nil, fmt.Errorf("nested incoming link %d not found. Should not happen [Process movements]", linkID)
					}
				}
			} else {
				return nil, fmt.Errorf("outcoming link %d not found. Should not happen [Process movements]", linkID)
			}
		}
	}

	/* */

	/* Process macro links */
	for linkID, link := range net.links {
		_ = linkID
		link.calcCutLen()
	}
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

func (link *NetworkLink) calcCutLen() {
	//  Defife a variable downstream_max_cut which is the maximum length of a cut that can be made downstream of the link,
	// calculated as the maximum of the _length_of_short_cut and the difference between the last two elements in the link.lanes_change_point_list minus 3.
	downStreamMaxCut := math.Max(shortcutLen, link.breakpoints[len(link.breakpoints)-1]-link.breakpoints[len(link.breakpoints)-2]-3)
	_ = downStreamMaxCut
	if link.upstreamShortCut && link.downstreamShortCut {
		totalLengthCut := 2 * shortcutLen * cutLenMin
		_ = totalLengthCut
		if link.lengthMetersOffset > totalLengthCut {
			link.upstreamCutLen = shortcutLen
			link.downstreamCutLen = shortcutLen
		} else {
			link.upstreamCutLen = (link.lengthMetersOffset / totalLengthCut) * shortcutLen
			link.downstreamCutLen = link.upstreamCutLen
		}
	} else if link.upstreamShortCut {
		cutIdx := 0
		cutPlaceFound := false
		for i := link.lanesList[len(link.lanesList)-1]; i >= 0; i-- {
			if link.lengthMetersOffset > math.Min(downStreamMaxCut, cutLen[i])+shortcutLen+cutLenMin {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			link.upstreamCutLen = shortcutLen
			link.downstreamCutLen = math.Min(downStreamMaxCut, cutLen[cutIdx])
		} else {
			downStreamCut := math.Min(downStreamMaxCut, cutLen[0])
			totalLen := downStreamCut + shortcutLen + cutLenMin
			link.upstreamCutLen = (link.lengthMetersOffset / totalLen) * shortcutLen
			link.downstreamCutLen = (link.lengthMetersOffset / totalLen) * downStreamCut
		}
	} else if link.downstreamShortCut {
		cutIdx := 0
		cutPlaceFound := false
		for i := link.lanesList[len(link.lanesList)-1]; i >= 0; i-- {
			if link.lengthMetersOffset > cutLen[i]+shortcutLen+cutLenMin {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			link.upstreamCutLen = cutLen[cutIdx]
			link.downstreamCutLen = shortcutLen
		} else {
			totalLen := cutLen[0] + shortcutLen + cutLenMin
			link.upstreamCutLen = (link.lengthMetersOffset / totalLen) * cutLen[0]
			link.downstreamCutLen = (link.lengthMetersOffset / totalLen) * shortcutLen
		}
	} else {
		cutIdx := 0
		cutPlaceFound := false
		for i := link.lanesList[len(link.lanesList)-1]; i >= 0; i-- {
			if link.lengthMetersOffset > cutLen[i]+math.Min(downStreamMaxCut, cutLen[i])+cutLenMin {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			link.upstreamCutLen = cutLen[cutIdx]
			link.downstreamCutLen = math.Min(downStreamMaxCut, cutLen[cutIdx])
		} else {
			downStreamCut := math.Min(downStreamMaxCut, cutLen[0])
			totalLen := downStreamCut + cutLen[0] + cutLenMin
			link.upstreamCutLen = (link.lengthMetersOffset / totalLen) * cutLen[0]
			link.downstreamCutLen = (link.lengthMetersOffset / totalLen) * downStreamCut
		}
	}
}

func linksToSlice(links map[NetworkLinkID]*NetworkLink) []NetworkLinkID {
	ans := make([]NetworkLinkID, 0, len(links))
	for k := range links {
		ans = append(ans, k)
	}
	return ans
}
