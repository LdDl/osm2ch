package osm2ch

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
)

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
		fmt.Print("Preparing mesoscopic...")
	}
	st := time.Now()
	mesoscopic := NetworkMesoscopic{
		nodes:             make(map[NetworkNodeID]*NetworkNodeMesoscopic),
		links:             make(map[NetworkLinkID]*NetworkLinkMesoscopic),
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

	/* Build meso */
	mesoscopic.generateLinks(net)
	mesoscopic.connectLinks(net)
	mesoscopic.updateBoundaryType(net)
	mesoscopic.updateLinksProperties(net)
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
				mesoNode.geomEuclidean = node.geomEuclidean.copy()
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
				ID:               sourceMacroNodeID*100 + NetworkNodeID(expNodesNum),
				geom:             link.geomOffsetCut[0][0], // No explicit copy or clone method since Point is not slice, but array
				geomEuclidean:    link.geomEuclideanOffsetCut[0][0],
				macroNodeID:      sourceMacroNodeID,
				macroLinkID:      -1,
				zoneID:           sourceMacroNode.zoneID,
				activityLinkType: sourceMacroNode.activityLinkType,
				boundaryType:     BOUNDARY_NONE,
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
					boundaryType:  BOUNDARY_NONE,
				}

				if segmentIdx == segmentsToCut-1 {
					downstreamMesoNode.macroNodeID = targetMacroNodeID
					downstreamMesoNode.macroLinkID = -1
					downstreamMesoNode.zoneID = targetMacroNode.zoneID
					downstreamMesoNode.activityLinkType = targetMacroNode.activityLinkType
				} else {
					downstreamMesoNode.macroNodeID = -1
					downstreamMesoNode.macroLinkID = link.ID
					downstreamMesoNode.zoneID = -1
					downstreamMesoNode.activityLinkType = LINK_UNDEFINED
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
				isConnection:  false,
				movementID:    -1,
				macroNodeID:   -1,
				lengthMeters:  geo.LengthHaversign(link.geomOffsetCut[segmentIdx]),
				// Default movement
				movementCompositeType:    MOVEMENT_NONE,
				movementLinkIncome:       NetworkLinkID(-1),
				movementLinkOutcome:      NetworkLinkID(-1),
				movementIncomeLaneStart:  -1,
				movementOutcomeLaneStart: -1,
			}

			link.mesolinks = append(link.mesolinks, mesoLink.ID)
			mesoNet.nodes[upstreamNodeID].outcomingLinks = append(mesoNet.nodes[upstreamNodeID].outcomingLinks, lastMesoLinkID)
			mesoNet.nodes[downstreamMesoNode.ID].incomingLinks = append(mesoNet.nodes[downstreamMesoNode.ID].incomingLinks, lastMesoLinkID)

			mesoNet.links[mesoLink.ID] = &mesoLink
			lastMesoLinkID += 1
			upstreamNodeID = downstreamMesoNode.ID // This must be done since current upstream node is downstream node for next segment
		}

		// @TODO: Create microscopic links since it could be done here
		// Consider to have some flag to enable/disable this feature
	}

	mesoNet.maxLinkID = lastMesoLinkID
	return nil
}

// connectLinks connects mesoscopic links via movements layer from macroscopic graph
//
// generated connections between links are links too
//
func (mesoNet *NetworkMesoscopic) connectLinks(macroNet *NetworkMacroscopic) error {
	lastMesoLinkID := mesoNet.maxLinkID

	// Loop through each macroscopic
	for _, macroNode := range macroNet.nodes {
		// Loop through each movement of give node
		for _, movement := range macroNode.movements {
			// Extract macroscopic links
			incomingMacroLinkID, outcomingMacroLinkID := movement.IncomingLinkID, movement.OutcomingLinkID
			incomingMacroLink, ok := macroNet.links[incomingMacroLinkID]
			if !ok {
				return fmt.Errorf("connectLinks(): Incoming macroscopic link %d not found", incomingMacroLinkID)
			}
			outcomingMacroLink, ok := macroNet.links[outcomingMacroLinkID]
			if !ok {
				return fmt.Errorf("connectLinks(): Outcoming macroscopic link %d not found", outcomingMacroLinkID)
			}

			// Collect lanes info
			incomeLanes := make([]int, 0, movement.incomeLaneStart+movement.incomeLaneEnd)
			for laneNo := movement.incomeLaneStart; laneNo <= movement.incomeLaneEnd; laneNo++ {
				incomeLanes = append(incomeLanes, laneNo)
			}
			outcomeLanes := make([]int, 0, movement.outcomeLaneStart+movement.outcomeLaneEnd)
			for laneNo := movement.outcomeLaneStart; laneNo <= movement.outcomeLaneEnd; laneNo++ {
				outcomeLanes = append(outcomeLanes, laneNo)
			}

			// Minor check. If this conditions met, then something is wrong with movements layer
			if len(incomeLanes) != len(outcomeLanes) {
				fmt.Printf("Warning. Income and outcome lanes number mismatch for movement %d. Income: %d, outcome: %d. This movement will be ignored\n", movement.ID, len(incomeLanes), len(outcomeLanes))
				continue
			}
			if intSliceContains(incomeLanes, 0) {
				fmt.Printf("Warning. Income lanes contains 0 for movement %d. This movement will be ignored\n", movement.ID)
				continue
			}
			if intSliceContains(outcomeLanes, 0) {
				fmt.Printf("Warning. Outcome lanes contains 0 for movement %d. This movement will be ignored\n", movement.ID)
				continue
			}

			// Extract corresponding mesoscopic links
			incomingMesoLinkID := incomingMacroLink.mesolinks[len(incomingMacroLink.mesolinks)-1]
			incomingMesoLink, ok := mesoNet.links[incomingMesoLinkID]
			if !ok {
				return fmt.Errorf("connectLinks(): Incoming mesoscopic link %d not found", incomingMesoLinkID)
			}
			outcomingMesoLinkID := outcomingMacroLink.mesolinks[0]
			outcomingMesoLink, ok := mesoNet.links[outcomingMesoLinkID]
			if !ok {
				return fmt.Errorf("connectLinks(): Outcoming mesoscopic link %d not found", outcomingMesoLinkID)
			}

			// Calculate lanes indices
			incomeLaneStart := incomingMesoLink.lanesChange[0] + incomeLanes[0]
			if incomeLanes[0] >= 0 {
				incomeLaneStart -= 1
			}
			incomeLaneEnd := incomingMesoLink.lanesChange[0] + incomeLanes[len(incomeLanes)-1]
			if incomeLanes[len(incomeLanes)-1] >= 0 {
				incomeLaneEnd -= 1
			}
			outcomeLaneStart := outcomingMesoLink.lanesChange[0] + outcomeLanes[0]
			if outcomeLanes[0] >= 0 {
				outcomeLaneStart -= 1
			}
			outcomeLaneEnd := outcomingMesoLink.lanesChange[0] + outcomeLanes[len(outcomeLanes)-1]
			if outcomeLanes[len(outcomeLanes)-1] >= 0 {
				outcomeLaneEnd -= 1
			}

			// Minor check. Ignore movements when inbound or outbound lane is not consistent (negative value)
			if incomeLaneStart < 0 {
				fmt.Printf("Warning. Income lane start is negative for movement %d. This movement will be ignored\n", movement.ID)
				continue
			}
			if outcomeLaneStart < 0 {
				fmt.Printf("Warning. Outcome lane start is negative for movement %d. This movement will be ignored\n", movement.ID)
				continue
			}
			// Minor check. Ignore movements when inbound or outbound lane is greater than number of lanes
			if incomeLaneEnd > incomingMesoLink.lanesNum-1 {
				fmt.Printf("Warning. Income lane end %d is greater than number of lanes %d for movement %d. This movement will be ignored\n", incomeLaneEnd, incomingMesoLink.lanesNum-1, movement.ID)
				continue
			}
			if outcomeLaneEnd > outcomingMesoLink.lanesNum-1 {
				fmt.Printf("Warning. Outcome lane end %d is greater than number of lanes %d for movement %d. This movement will be ignored\n", outcomeLaneEnd, outcomingMesoLink.lanesNum-1, movement.ID)
				continue
			}

			// Generate mesoscopic link if it's needed
			lanesNum := len(incomeLanes)
			if macroNode.movementIsNeeded {
				mesoLink := NetworkLinkMesoscopic{
					ID:            lastMesoLinkID,
					sourceNodeID:  incomingMesoLink.targetNodeID,
					targetNodeID:  outcomingMesoLink.sourceNodeID,
					lanesNum:      lanesNum,
					lanesChange:   make([]int, 0),
					geom:          orb.LineString{incomingMesoLink.geom[len(incomingMesoLink.geom)-1], outcomingMesoLink.geom[0]},
					geomEuclidean: orb.LineString{incomingMesoLink.geomEuclidean[len(incomingMesoLink.geomEuclidean)-1], outcomingMesoLink.geomEuclidean[0]},
					macroLinkID:   -1,
					isConnection:  true,
					movementID:    movement.ID,
					macroNodeID:   macroNode.ID,
					lengthMeters:  geo.LengthHaversign(orb.LineString{incomingMesoLink.geom[len(incomingMesoLink.geom)-1], outcomingMesoLink.geom[0]}),
					// Inherit movement properties
					movementCompositeType:    movement.movementCompositeType,
					movementLinkIncome:       incomingMesoLinkID,
					movementLinkOutcome:      outcomingMesoLinkID,
					movementIncomeLaneStart:  incomeLaneStart,
					movementOutcomeLaneStart: outcomeLaneStart,
				}
				mesoNet.links[mesoLink.ID] = &mesoLink
				lastMesoLinkID += 1

				// Update incident edges lists for nodes
				mesoNet.nodes[mesoLink.sourceNodeID].outcomingLinks = append(mesoNet.nodes[mesoLink.sourceNodeID].outcomingLinks, mesoLink.ID)
				mesoNet.nodes[mesoLink.targetNodeID].incomingLinks = append(mesoNet.nodes[mesoLink.targetNodeID].incomingLinks, mesoLink.ID)
			} else {

				if incomingMacroLink.downstreamIsTarget && !outcomingMacroLink.upstreamIsTarget {
					// remove incoming micro nodes and links of outcomingMesoLink, then connect to incomingMesoLink
					incomingMesoLinkTargetNodeID := incomingMesoLink.targetNodeID
					outcomingMesoLinkSourceNodeID := outcomingMesoLink.sourceNodeID

					outcomingMesoLink.sourceNodeID = incomingMesoLinkTargetNodeID
					outcomingMesoLink.geom = append(orb.LineString{incomingMesoLink.geom[len(incomingMesoLink.geom)-1]}, outcomingMesoLink.geom[1:]...)
					outcomingMesoLink.geomEuclidean = append(orb.LineString{incomingMesoLink.geomEuclidean[len(incomingMesoLink.geomEuclidean)-1]}, outcomingMesoLink.geomEuclidean[1:]...)

					delete(mesoNet.nodes, outcomingMesoLinkSourceNodeID)

					//@todo process micro? @CRITICAL
				} else if !incomingMacroLink.downstreamIsTarget && outcomingMacroLink.upstreamIsTarget {
					//remove outgoing micro nodes and links of incomingMesoLink, then connect to outcomingMesoLink
					incomingMesoLinkTargetNodeID := incomingMesoLink.targetNodeID
					outcomingMesoLinkSourceNodeID := outcomingMesoLink.sourceNodeID

					incomingMesoLink.targetNodeID = outcomingMesoLinkSourceNodeID
					incomingMesoLink.geom = append(incomingMesoLink.geom[:len(incomingMesoLink.geom)-1], outcomingMesoLink.geom[0])
					incomingMesoLink.geomEuclidean = append(incomingMesoLink.geomEuclidean[:len(incomingMesoLink.geomEuclidean)-1], outcomingMesoLink.geomEuclidean[0])

					delete(mesoNet.nodes, incomingMesoLinkTargetNodeID)

					//@todo process micro? @CRITICAL
				}
			}
		}
	}
	mesoNet.maxLinkID = lastMesoLinkID
	return nil
}

// updateBoundaryType updates boundary type for each mesoscopic node
//
// this function should be called after all incident edges for nodes are set
//
func (mesoNet *NetworkMesoscopic) updateBoundaryType(macroNet *NetworkMacroscopic) error {
	for _, mesoNode := range mesoNet.nodes {
		if mesoNode.macroNodeID == -1 {
			if mesoNode.macroLinkID == -1 {
				fmt.Printf("Warning. Suspicious mesoscopic node %d: either macroscopic node ir link not found\n", mesoNode.ID)
			} else {
				mesoNode.boundaryType = BOUNDARY_NONE
			}
		} else {
			macroNode, ok := macroNet.nodes[mesoNode.macroNodeID]
			if !ok {
				return fmt.Errorf("connectNodes(): Macroscopic node %d not found for mesoscopic node %d", mesoNode.macroNodeID, mesoNode.ID)
			}
			if macroNode.boundaryType != BOUNDARY_INCOME_OUTCOME {
				mesoNode.boundaryType = macroNode.boundaryType
			} else {
				if len(mesoNode.incomingLinks) != 0 {
					mesoNode.boundaryType = BOUNDARY_INCOME_ONLY
				} else {
					mesoNode.boundaryType = BOUNDARY_OUTCOME_ONLY
				}
			}
		}
	}
	return nil
}

// updateLinksProperties updates mesoscopic links properties
func (mesoNet *NetworkMesoscopic) updateLinksProperties(macroNet *NetworkMacroscopic) error {
	movementLinks := make(map[NetworkLinkID]struct{})

	for _, mesoLink := range mesoNet.links {
		if mesoLink.macroNodeID == -1 {
			if mesoLink.macroLinkID == -1 {
				fmt.Printf("Warning. Suspicious mesoscopic link %d: either macroscopic node ir link not found\n", mesoLink.ID)
			} else {
				// Inherit macroscopic link properties
				macroLink, ok := macroNet.links[mesoLink.macroLinkID]
				if !ok {
					return fmt.Errorf("updateLinksProperties(): Macroscopic link %d not found for mesoscopic link %d", mesoLink.macroLinkID, mesoLink.ID)
				}
				mesoLink.linkType = macroLink.linkType
				mesoLink.freeSpeed = macroLink.freeSpeed
				mesoLink.capacity = macroLink.capacity
				mesoLink.allowedAgentTypes = macroLink.allowedAgentTypes

				// Reset macroscopic node properties to defaults
				mesoLink.controlType = NOT_SIGNAL
			}
		} else {
			// Collect movement-based links and inherit macroscopic link properties later
			movementLinks[mesoLink.ID] = struct{}{}

			// Inherit macroscopic node properties
			macroNode, ok := macroNet.nodes[mesoLink.macroNodeID]
			if !ok {
				return fmt.Errorf("updateLinksProperties(): Macroscopic node %d not found for mesoscopic link %d", mesoLink.macroNodeID, mesoLink.ID)
			}
			mesoLink.controlType = macroNode.controlType

			if mesoLink.movementID == -1 {
				fmt.Printf("Warning. Suspicious mesoscopic link %d: it should have movement ID since it is movement-based\n", mesoLink.ID)
			}
		}

		// Update movement. @TODO: probably redundant
		if mesoLink.movementID != -1 {
			movement, ok := macroNet.movement[mesoLink.movementID]
			if !ok {
				return fmt.Errorf("updateLinksProperties(): Movement %d not found for mesoscopic link %d", mesoLink.movementID, mesoLink.ID)
			}
			mesoLink.movementCompositeType = movement.movementCompositeType
		}
	}

	// Inherit macroscopic link properties for movement links
	for movementLinks := range movementLinks {
		mesoLink, ok := mesoNet.links[movementLinks]
		if !ok {
			return fmt.Errorf("updateLinksProperties(): Movement link %d not found in mesoscopic network", movementLinks)
		}
		sourceMesoNode, ok := mesoNet.nodes[mesoLink.sourceNodeID]
		if !ok {
			return fmt.Errorf("updateLinksProperties(): Mesoscopic source node %d not found for mesoscopic link %d", mesoLink.sourceNodeID, mesoLink.ID)
		}
		if len(sourceMesoNode.incomingLinks) == 0 {
			fmt.Printf("Warning. Mesoscopic link %d has no incoming links for source node %d. Skipping\n", mesoLink.ID, mesoLink.sourceNodeID)
			continue
		}
		upstreamLinkID := sourceMesoNode.incomingLinks[0]
		upstreamLink, ok := mesoNet.links[upstreamLinkID]
		if !ok {
			return fmt.Errorf("updateLinksProperties(): Mesoscopic upstream link %d not found for mesoscopic link %d. Source node is %d", upstreamLinkID, mesoLink.ID, mesoLink.sourceNodeID)
		}
		// Inherit upstream macroscopic link properties
		mesoLink.linkType = upstreamLink.linkType
		mesoLink.freeSpeed = upstreamLink.freeSpeed
		mesoLink.capacity = upstreamLink.capacity
		mesoLink.allowedAgentTypes = upstreamLink.allowedAgentTypes
	}
	return nil
}

// intSliceContains returns true if element is in slice
func intSliceContains(slice []int, element int) bool {
	for _, el := range slice {
		if el == element {
			return true
		}
	}
	return false
}
