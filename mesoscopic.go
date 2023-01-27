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
	nodes map[NetworkNodeID]*NetworkNodeMesoscopic
	links map[NetworkLinkID]NetworkLinkMesoscopic
	// Storage to track number of generated mesoscopic nodes for each macroscopic node which is centroid
	// Key: NodeID, Value: Number of expanded nodes
	expandedMesoNodes map[NetworkNodeID]int
	// Track ID generator
	maxLinkID NetworkLinkID
}

const (
	resolution  = 5.0
	laneWidth   = 3.5
	shortcutLen = 0.1
	cutLenMin   = 2.0
	cellLength  = 4.5
)

var (
	cutLen = [100]float64{2.0, 8.0, 12.0, 14.0, 16.0, 18.0, 20, 22, 24, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25, 25}
)

func (net *NetworkMacroscopic) genMesoscopicNetwork(verbose bool) (*NetworkMesoscopic, error) {
	if verbose {
		fmt.Print("Preparing mesocopic...")
	}
	st := time.Now()
	mesoscopic := NetworkMesoscopic{
		nodes:             make(map[NetworkNodeID]*NetworkNodeMesoscopic),
		links:             make(map[NetworkLinkID]NetworkLinkMesoscopic),
		expandedMesoNodes: make(map[NetworkNodeID]int),
	}

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
			offsetGeom := offsetCurve(geomEuclidean, -offsetDistance) // Use "-" sign to make offset to the right side
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
		// fmt.Println("cut", linkID)
		// Prepare cut length for each link
		link.calcCutLen()
		// Perform the cut
		link.performCut()
	}
	/* */

	/* Gen movement (if needed) */
	// I has to be done before the micro/meso generation currently.
	// @todo: Consider optional auto-generation
	/* */

	/* Build meso/micro */
	mesoscopic.generateLinks(net)
	/* */

	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}
	return &mesoscopic, nil
}

// createSubNodes creates expanded nodes for nodes in macroscopic network
// @TODO: Consider additional field `isCentroid` for NetworkNode type.
func (mesoNet *NetworkMesoscopic) createExpandedNodes(macroNodes map[NetworkNodeID]*NetworkNode) {
	for _, node := range macroNodes {
		_ = node
		// Pseudo-code
		/*
			if node.isCentroid {
				if mesoNet.expandedMesoNodes.has(node.id) {
					mesoNet.expandedMesoNodes.set(node.id, 0)
				}
				expandedNodesNum := mesoNet.expandedMesoNodes.get(node.id)
				mesoNet.expandedMesoNodes[node.id] += 1
				mesoNode := NewMesoNode(node.id, expandedNodesNum)
				mesoNode.geom = node.geom.copy()
				mesoNode.geomEuclidead = node.geomEuclidead.copy()
				mesoNode.macro_node_id = node.id
				node.centroid_meso_node_id = mesoNode.id
				mesoNet.nodes[mesoNode.id] = mesoNode
			}
		*/
	}
}

// generateLinks generates mesoscopic links from post-processed macroscopic links (with needed cuts)
func (mesoNet *NetworkMesoscopic) generateLinks(macroNet *NetworkMacroscopic) error {
	lastMesoLinkID := NetworkLinkID(0)

	for _, link := range macroNet.links {
		// Prepare source mesoscopic node
		var upstreamMesoNode NetworkNodeMesoscopic
		sourceMacroNodeID := link.sourceNodeID
		sourceMacroNode, ok := macroNet.nodes[sourceMacroNodeID]
		if !ok {
			return fmt.Errorf("generateLinks(): Source node %d not found", sourceMacroNodeID)
		}
		if sourceMacroNode.isCentroid {
			// @TODO: upstream == macro.centroid ???
		} else {
			expNodesNum, ok := mesoNet.expandedMesoNodes[sourceMacroNodeID]
			if !ok {
				mesoNet.expandedMesoNodes[sourceMacroNodeID] = 0
			}
			mesoNet.expandedMesoNodes[sourceMacroNodeID] += 1
			upstreamMesoNode = NetworkNodeMesoscopic{
				ID:            sourceMacroNodeID*100 + NetworkNodeID(expNodesNum),
				geom:          link.geomOffsetCut[0][0], // No explicit copy or clone method since Point is not slice, but array
				geomEuclidean: link.geomEuclideanOffsetCut[0][0],
				macroNodeID:   sourceMacroNodeID,
				macroLinkID:   -1,
			}
			mesoNet.nodes[upstreamMesoNode.ID] = &upstreamMesoNode
		}

		// Prepare link and target mesoscopic node
		segmentsToCut := len(link.lanesListCut)
		upstreamNodeID := upstreamMesoNode.ID
		var downstreamMesoNode NetworkNodeMesoscopic
		targetMacroNodeID := link.targetNodeID
		targetMacroNode, ok := macroNet.nodes[targetMacroNodeID]
		if !ok {
			return fmt.Errorf("generateLinks(): Target node %d not found", sourceMacroNodeID)
		}
		for segmentIdx := 0; segmentIdx < segmentsToCut; segmentIdx++ {
			// Prepare mesoscopic node
			if targetMacroNode.isCentroid && segmentIdx == segmentsToCut-1 {
				// @TODO: downstream == macro.centroid ???
			} else {
				expNodesNum, ok := mesoNet.expandedMesoNodes[targetMacroNodeID]
				if !ok {
					mesoNet.expandedMesoNodes[targetMacroNodeID] = 0
				}
				mesoNet.expandedMesoNodes[targetMacroNodeID] += 1
				downstreamMesoNode = NetworkNodeMesoscopic{
					ID:            targetMacroNodeID*100 + NetworkNodeID(expNodesNum),
					geom:          link.geomOffsetCut[segmentIdx][len(link.geomOffsetCut[segmentIdx])-1], // No explicit copy or clone method since Point is not slice, but array
					geomEuclidean: link.geomEuclideanOffsetCut[segmentIdx][len(link.geomEuclideanOffsetCut[segmentIdx])-1],
				}
				if segmentIdx == segmentsToCut-1 {
					downstreamMesoNode.macroNodeID = targetMacroNodeID
					downstreamMesoNode.macroLinkID = -1
				} else {
					downstreamMesoNode.macroNodeID = -1
					downstreamMesoNode.macroLinkID = link.ID
				}
				mesoNet.nodes[downstreamMesoNode.ID] = &downstreamMesoNode
			}

			// Prepare mesoscopic link

			mesoLink := NetworkLinkMesoscopic{
				ID:            lastMesoLinkID,
				sourceNodeID:  upstreamNodeID,
				targetNodeID:  downstreamMesoNode.ID,
				lanesNum:      link.lanesListCut[segmentIdx],
				lanesChange:   link.lanesChangeCut[segmentIdx],
				geom:          link.geomOffsetCut[segmentIdx].Clone(),
				geomEuclidean: link.geomEuclideanOffsetCut[segmentIdx].Clone(),
				macroLinkID:   link.ID,
			}

			mesoNet.nodes[upstreamNodeID].outcomingLinks = append(mesoNet.nodes[upstreamNodeID].outcomingLinks, lastMesoLinkID)
			mesoNet.nodes[downstreamMesoNode.ID].incomingLinks = append(mesoNet.nodes[downstreamMesoNode.ID].incomingLinks, lastMesoLinkID)

			mesoNet.links[mesoLink.ID] = mesoLink
			lastMesoLinkID += 1
			upstreamNodeID = downstreamMesoNode.ID // This must be done since current upstream node is downstream node for next segment
		}

		// @TODO: Create microscopic links since it could be done here
		// Consider to have some flag to enable/disable this feature
	}
	mesoNet.maxLinkID = lastMesoLinkID
	return nil
}
