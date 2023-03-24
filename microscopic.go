package osm2ch

import (
	"fmt"
	"math"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/pkg/errors"
)

const (
	bikeLaneWidth = 0.5
	walkLaneWidth = 0.5
	cellLength    = 4.5
)

func genMicroscopicNetwork(macroNet *NetworkMacroscopic, mesoNet *NetworkMesoscopic, separateBikeWalk, verbose bool) (*NetworkMicroscopic, error) {
	if verbose {
		fmt.Print("Preparing microscopic...")
	}
	st := time.Now()
	microscopic := NetworkMicroscopic{
		nodes:     make(map[NetworkNodeID]*NetworkNodeMicroscopic),
		links:     make(map[NetworkLinkID]*NetworkLinkMicroscopic),
		maxLinkID: NetworkLinkID(0),
		maxNodeID: NetworkNodeID(0),
	}
	fmt.Println()

	lastNodeID := microscopic.maxNodeID
	lastLinkID := microscopic.maxLinkID

	// Iterate over macroscopic links
	for _, macroLink := range macroNet.links {
		// fmt.Println("create data for link", macroLink.ID)
		// Evaluate multimodal agent types for macroscopic link
		agentTypes := macroLink.allowedAgentTypes
		var multiModalAgentTypes []AgentType
		var bike, walk bool
		if separateBikeWalk {
			multiModalAgentTypes, bike, walk = prepareBikeWalkAgents(agentTypes)
		} else {
			bike, walk = false, false
			multiModalAgentTypes = make([]AgentType, len(agentTypes))
			copy(multiModalAgentTypes, agentTypes)
		}

		originalLanesNum := float64(macroLink.lanesList[0])

		// Iterate over mesoscopic links and create microscopic nodes
		for _, mesoLinkID := range macroLink.mesolinks {
			mesoLink, ok := mesoNet.links[mesoLinkID]
			if !ok {
				return nil, fmt.Errorf("genMicroscopicNetwork(): Mesoscopic link %d not found for macroscopic link %d", mesoLinkID, macroLink.ID)
			}

			laneChangesLeft := float64(mesoLink.lanesChange[0])
			lanesNumberInBetween := -1 * (originalLanesNum/2 - 0.5 + laneChangesLeft)

			// fmt.Println("\tmesolink", mesoLinkID)

			laneGeometries := []orb.LineString{}
			bikeGeometry := orb.LineString{}
			walkGeometry := orb.LineString{}
			laneOffset := 0.0
			// Iterate over mesoscopic link lanes and prepare geometries
			for i := 0; i < mesoLink.lanesNum; i++ {
				laneOffset := (lanesNumberInBetween + float64(i)) * laneWidth
				// fmt.Println("\titerate lane", i, laneOffset)
				// If offset is too small then neglect it and copy original geometry
				// Otherwise evaluate offset for geometry
				if laneOffset < -1e-2 || laneOffset > 1e-2 {
					laneGeomEuclidean := offsetCurve(mesoLink.geomEuclidean, -laneOffset) // Use "-" sign to make offset to the right side
					// if laneOffset > 0 {
					// laneGeomEuclidean.Reverse()
					// }
					laneGeometries = append(laneGeometries, lineToSpherical(laneGeomEuclidean))
				} else {
					laneGeometries = append(laneGeometries, mesoLink.geom.Clone())
				}
			}
			if bike && !walk {
				// Prepare only bike geometry: calculate offset and evaluate geometry
				bikeLaneOffset := laneOffset + bikeLaneWidth
				if bikeLaneOffset < -1e-2 || bikeLaneOffset > 1e-2 {
					bikeGeometryEuclidean := offsetCurve(mesoLink.geomEuclidean, -bikeLaneOffset)
					// if bikeLaneOffset > 0 {
					// 	bikeGeometryEuclidean.Reverse()
					// }
					bikeGeometry = lineToSpherical(bikeGeometryEuclidean)
				} else {
					bikeGeometry = mesoLink.geom.Clone()
				}
			} else if !bike && walk {
				// Prepare only walk geometry: calculate offset and evaluate geometry
				walkLaneOffset := laneOffset + walkLaneWidth
				if walkLaneOffset < -1e-2 || walkLaneOffset > 1e-2 {
					walkGeometryEuclidean := offsetCurve(mesoLink.geomEuclidean, -walkLaneOffset)
					// if walkLaneOffset > 0 {
					// 	walkGeometryEuclidean.Reverse()
					// }
					walkGeometry = lineToSpherical(walkGeometryEuclidean)
				} else {
					walkGeometry = mesoLink.geom.Clone()
				}
			} else if bike && walk {
				// Prepare both bike and walk geometry: calculate two offsets and evaluate geometries
				bikeLaneOffset := laneOffset + bikeLaneWidth
				walkLaneOffset := laneOffset + walkLaneWidth
				if bikeLaneOffset < -1e-2 || bikeLaneOffset > 1e-2 {
					bikeGeometryEuclidean := offsetCurve(mesoLink.geomEuclidean, -bikeLaneOffset)
					// if bikeLaneOffset > 0 {
					// 	bikeGeometryEuclidean.Reverse()
					// }
					bikeGeometry = lineToSpherical(bikeGeometryEuclidean)
				} else {
					bikeGeometry = mesoLink.geom.Clone()
				}
				if walkLaneOffset < -1e-2 || walkLaneOffset > 1e-2 {
					walkGeometryEuclidean := offsetCurve(mesoLink.geomEuclidean, -walkLaneOffset)
					// if walkLaneOffset > 0 {
					// 	walkGeometryEuclidean.Reverse()
					// }
					walkGeometry = lineToSpherical(walkGeometryEuclidean)
				} else {
					walkGeometry = mesoLink.geom.Clone()
				}
			}
			// Calculate number of cell which fit into link
			// If cell length > link length then use only one cell
			cellsNum := math.Max(1.0, math.Round(mesoLink.lengthMeters/cellLength))
			// Loop over lanes, get interpolated point for each cell
			// and collect them
			microNodesGeometries := [][]orb.Point{}
			microNodesGeometriesEuclidean := [][]orb.Point{}

			bikeMicroNodesGeometries := []orb.Point{}
			bikeMicroNodesGeometriesEuclidean := []orb.Point{}

			walkMicroNodesGeometries := []orb.Point{}
			walkMicroNodesGeometriesEuclidean := []orb.Point{}

			for _, laneGeom := range laneGeometries {
				laneNodes := []orb.Point{}
				laneNodesEuclidean := []orb.Point{}

				for i := 0; i < int(cellsNum)+1; i++ {
					fraction := float64(i) / float64(cellsNum)
					distance := mesoLink.lengthMeters * fraction
					point, _ := geo.PointAtDistanceAlongLine(laneGeom, distance)
					laneNodes = append(laneNodes, point)
					laneNodesEuclidean = append(laneNodesEuclidean, pointToEuclidean(point))
				}
				microNodesGeometries = append(microNodesGeometries, laneNodes)
				microNodesGeometriesEuclidean = append(microNodesGeometriesEuclidean, laneNodesEuclidean)
			}
			if bike {
				for i := 0; i < int(cellsNum)+1; i++ {
					fraction := float64(i) / float64(cellsNum)
					distance := mesoLink.lengthMeters * fraction
					point, _ := geo.PointAtDistanceAlongLine(bikeGeometry, distance)
					bikeMicroNodesGeometries = append(bikeMicroNodesGeometries, point)
					bikeMicroNodesGeometriesEuclidean = append(bikeMicroNodesGeometriesEuclidean, pointToEuclidean(point))
				}
			}
			if walk {
				for i := 0; i < int(cellsNum)+1; i++ {
					fraction := float64(i) / float64(cellsNum)
					distance := mesoLink.lengthMeters * fraction
					point, _ := geo.PointAtDistanceAlongLine(walkGeometry, distance)
					walkMicroNodesGeometries = append(walkMicroNodesGeometries, point)
					walkMicroNodesGeometriesEuclidean = append(walkMicroNodesGeometriesEuclidean, pointToEuclidean(point))
				}
			}

			// Prepare microscopic nodes for each lane of mesoscopic link
			for i := 0; i < mesoLink.lanesNum; i++ {
				laneNodesIDs := []NetworkNodeID{}
				for j, microNodeGeom := range microNodesGeometries[i] {
					microNode := NetworkNodeMicroscopic{
						ID:                         lastNodeID,
						geom:                       microNodeGeom,
						geomEuclidean:              microNodesGeometriesEuclidean[i][j],
						mesoLinkID:                 mesoLink.ID,
						laneID:                     i + 1,
						isLinkUpstreamTargetNode:   false,
						isLinkDownstreamTargetNode: false,
						zoneID:                     -1,
						boundaryType:               BOUNDARY_NONE,
					}
					laneNodesIDs = append(laneNodesIDs, microNode.ID)
					microscopic.nodes[microNode.ID] = &microNode
					lastNodeID++
				}
				mesoLink.microNodesPerLane = append(mesoLink.microNodesPerLane, laneNodesIDs)
			}
			if bike {
				for j, microNodeGeom := range bikeMicroNodesGeometries {
					microNode := NetworkNodeMicroscopic{
						ID:                         lastNodeID,
						geom:                       microNodeGeom,
						geomEuclidean:              bikeMicroNodesGeometriesEuclidean[j],
						mesoLinkID:                 mesoLink.ID,
						laneID:                     -1,
						isLinkUpstreamTargetNode:   false,
						isLinkDownstreamTargetNode: false,
						zoneID:                     -1,
						boundaryType:               BOUNDARY_NONE,
					}
					microscopic.nodes[microNode.ID] = &microNode
					mesoLink.microNodesBikeLane = append(mesoLink.microNodesBikeLane, microNode.ID)
					lastNodeID++
				}
			}
			if walk {
				for j, microNodeGeom := range walkMicroNodesGeometries {
					microNode := NetworkNodeMicroscopic{
						ID:                         lastNodeID,
						geom:                       microNodeGeom,
						geomEuclidean:              walkMicroNodesGeometriesEuclidean[j],
						mesoLinkID:                 mesoLink.ID,
						laneID:                     -2,
						isLinkUpstreamTargetNode:   false,
						isLinkDownstreamTargetNode: false,
						zoneID:                     -1,
						boundaryType:               BOUNDARY_NONE,
					}
					microscopic.nodes[microNode.ID] = &microNode
					mesoLink.microNodesWalkLane = append(mesoLink.microNodesWalkLane, microNode.ID)
					lastNodeID++
				}
			}
		}

		if len(macroLink.mesolinks) == 0 {
			fmt.Printf("[WARNING]: genMicroscopicNetwork(): Suspicious macroscopic link %v: no mesoscopic links\n", macroLink.ID)
			continue
		}

		// Mark upstream and downstream nodes for first and last mesoscopic link
		firstMesoLinkID := macroLink.mesolinks[0]
		firstMesoLink, ok := mesoNet.links[firstMesoLinkID]
		if !ok {
			return nil, fmt.Errorf("genMicroscopicNetwork(): First mesoscopic link %d not found for macroscopic link %d", firstMesoLinkID, macroLink.ID)
		}
		// Macroscopic source node will be needed to attach zone ID
		macroSourceNodeID := macroLink.sourceNodeID
		macroSourceNode, ok := macroNet.nodes[macroSourceNodeID]
		if !ok {
			return nil, fmt.Errorf("genMicroscopicNetwork(): Macroscopic source node %d not found for macroscopic link %d for mesoscopic link %d", macroSourceNodeID, macroLink.ID, firstMesoLinkID)
		}
		for _, microNodeLane := range firstMesoLink.microNodesPerLane {
			// @todo: check size of nodes per lane slice
			firstNodeID := microNodeLane[0]
			firstNode, ok := microscopic.nodes[firstNodeID]
			if !ok {
				return nil, fmt.Errorf("genMicroscopicNetwork(): Microscopic node %d not found for first mesoscopic link %d for macroscopic link %d", firstNodeID, firstMesoLinkID, macroLink.ID)
			}
			firstNode.isLinkUpstreamTargetNode = true
			// Attach zone ID to this node node
			firstNode.zoneID = macroSourceNode.zoneID
		}
		if bike {
			// @todo: check size of microNodeLane
			firstNodeID := firstMesoLink.microNodesBikeLane[0]
			firstNode, ok := microscopic.nodes[firstNodeID]
			if !ok {
				return nil, fmt.Errorf("genMicroscopicNetwork(): Microscopic node %d not found for first BIKE mesoscopic link %d for macroscopic link %d", firstNodeID, firstMesoLinkID, macroLink.ID)
			}
			firstNode.isLinkUpstreamTargetNode = true
			// Attach zone ID to this node node
			firstNode.zoneID = macroSourceNode.zoneID
		}
		if walk {
			// @todo: check size of microNodeLane
			firstNodeID := firstMesoLink.microNodesWalkLane[0]
			firstNode, ok := microscopic.nodes[firstNodeID]
			if !ok {
				return nil, fmt.Errorf("genMicroscopicNetwork(): Microscopic node %d not found for first WALK mesoscopic link %d for macroscopic link %d", firstNodeID, firstMesoLinkID, macroLink.ID)
			}
			firstNode.isLinkUpstreamTargetNode = true
			// Attach zone ID to this node node
			firstNode.zoneID = macroSourceNode.zoneID
		}

		lastMesoLinkID := macroLink.mesolinks[len(macroLink.mesolinks)-1]
		lastMesoLink, ok := mesoNet.links[lastMesoLinkID]
		if !ok {
			return nil, fmt.Errorf("genMicroscopicNetwork(): Last mesoscopic link %d not found for macroscopic link %d", lastMesoLinkID, macroLink.ID)
		}
		macroTargetNodeID := macroLink.targetNodeID
		macroTargetNode, ok := macroNet.nodes[macroTargetNodeID]
		if !ok {
			return nil, fmt.Errorf("genMicroscopicNetwork(): Macroscopic target node %d not found for macroscopic link %d for mesoscopic link %d", macroTargetNodeID, macroLink.ID, firstMesoLinkID)
		}
		for _, microNodeLane := range lastMesoLink.microNodesPerLane {
			// @todo: check size of microNodeLane
			lastNodeID := microNodeLane[len(microNodeLane)-1]
			lastNode, ok := microscopic.nodes[lastNodeID]
			if !ok {
				return nil, fmt.Errorf("genMicroscopicNetwork(): Microscopic node %d not found for last mesoscopic link %d for macroscopic link %d", lastNodeID, lastMesoLinkID, macroLink.ID)
			}
			lastNode.isLinkDownstreamTargetNode = true
			lastNode.zoneID = macroTargetNode.zoneID
		}
		if bike {
			// @todo: check size of microNodeLane
			lastNodeID := lastMesoLink.microNodesBikeLane[len(lastMesoLink.microNodesBikeLane)-1]
			lastNode, ok := microscopic.nodes[lastNodeID]
			if !ok {
				return nil, fmt.Errorf("genMicroscopicNetwork(): Microscopic node %d not found for last BIKE mesoscopic link %d for macroscopic link %d", lastNodeID, lastMesoLinkID, macroLink.ID)
			}
			lastNode.isLinkDownstreamTargetNode = true
			lastNode.zoneID = macroTargetNode.zoneID
		}
		if walk {
			// @todo: check size of microNodeLane
			lastNodeID := lastMesoLink.microNodesWalkLane[len(lastMesoLink.microNodesWalkLane)-1]
			lastNode, ok := microscopic.nodes[lastNodeID]
			if !ok {
				return nil, fmt.Errorf("genMicroscopicNetwork(): Microscopic node %d not found for last WALK mesoscopic link %d for macroscopic link %d", lastNodeID, lastMesoLinkID, macroLink.ID)
			}
			lastNode.isLinkDownstreamTargetNode = true
			lastNode.zoneID = macroTargetNode.zoneID
		}

		// Post-process microscopics nodes between two adjacent mesoscopic links
		for i := 0; i < len(macroLink.mesolinks)-1; i++ {
			upstreamMesolinkID := macroLink.mesolinks[i]
			downstreamMesolinkID := macroLink.mesolinks[i+1]

			upstreamMesolink, ok := mesoNet.links[upstreamMesolinkID]
			if !ok {
				return nil, fmt.Errorf("genMicroscopicNetwork(): Upstream mesoscopic link %d not found for macroscopic link %d", upstreamMesolinkID, macroLink.ID)
			}
			downstreamMesolink, ok := mesoNet.links[downstreamMesolinkID]
			if !ok {
				return nil, fmt.Errorf("genMicroscopicNetwork(): Downstream mesoscopic link %d not found for macroscopic link %d", downstreamMesolinkID, macroLink.ID)
			}

			upstreamLeftLaneOriginal := upstreamMesolink.lanesChange[0]
			downstreamLeftLaneOriginal := downstreamMesolink.lanesChange[0]

			minLeftLane := min(upstreamLeftLaneOriginal, downstreamLeftLaneOriginal)
			upstreamLaneStart := upstreamLeftLaneOriginal - minLeftLane
			downstreamLaneStart := downstreamLeftLaneOriginal - minLeftLane

			numberOfConnections := min(upstreamMesolink.lanesNum-upstreamLaneStart, downstreamMesolink.lanesNum-downstreamLaneStart)
			for j := 0; j < numberOfConnections; j++ {
				upstreamLane := upstreamLaneStart + j
				downstreamLane := downstreamLaneStart + j
				upstreamMicroNodeID := upstreamMesolink.microNodesPerLane[upstreamLane][len(upstreamMesolink.microNodesPerLane[upstreamLane])-1]
				downstreamMicroNodeID := downstreamMesolink.microNodesPerLane[downstreamLane][0]
				upstreamMesolink.microNodesPerLane[upstreamLane][len(upstreamMesolink.microNodesPerLane[upstreamLane])-1] = downstreamMicroNodeID
				delete(microscopic.nodes, upstreamMicroNodeID)
			}
			if bike {
				upstreamMicroNodeID := upstreamMesolink.microNodesBikeLane[len(upstreamMesolink.microNodesBikeLane)-1]
				downstreamMicroNodeID := downstreamMesolink.microNodesBikeLane[0]
				upstreamMesolink.microNodesBikeLane[len(upstreamMesolink.microNodesBikeLane)-1] = downstreamMicroNodeID
				delete(microscopic.nodes, upstreamMicroNodeID)
			}
			if walk {
				upstreamMicroNodeID := upstreamMesolink.microNodesWalkLane[len(upstreamMesolink.microNodesWalkLane)-1]
				downstreamMicroNodeID := downstreamMesolink.microNodesWalkLane[0]
				upstreamMesolink.microNodesWalkLane[len(upstreamMesolink.microNodesWalkLane)-1] = downstreamMicroNodeID
				delete(microscopic.nodes, upstreamMicroNodeID)
			}
		}

		// Create microscopic links (a.k.a. cells in terms of cellular automata)
		for _, mesoLinkID := range macroLink.mesolinks {
			mesoLink, ok := mesoNet.links[mesoLinkID]
			if !ok {
				return nil, fmt.Errorf("genMicroscopicNetwork(): Mesoscopic link %d not found for macroscopic link %d", mesoLinkID, macroLink.ID)
			}
			for i := 0; i < mesoLink.lanesNum; i++ {
				// Forward
				for j := 0; j < len(mesoLink.microNodesPerLane[i])-1; j++ {
					sourceNodeID := mesoLink.microNodesPerLane[i][j]
					targetNodeID := mesoLink.microNodesPerLane[i][j+1]
					sourceNode, ok := microscopic.nodes[sourceNodeID]
					if !ok {
						return nil, fmt.Errorf("genMicroscopicNetwork(): Source microscopic node %d not found for mesoscopic link %d", sourceNodeID, mesoLinkID)
					}
					targetNode, ok := microscopic.nodes[targetNodeID]
					if !ok {
						return nil, fmt.Errorf("genMicroscopicNetwork(): Target microscopic node %d not found for mesoscopic link %d", targetNodeID, mesoLinkID)
					}
					microLink := NetworkLinkMicroscopic{
						ID:                    lastLinkID,
						sourceNodeID:          sourceNodeID,
						targetNodeID:          targetNodeID,
						geom:                  orb.LineString{sourceNode.geom, targetNode.geom},
						geomEuclidean:         orb.LineString{sourceNode.geomEuclidean, targetNode.geomEuclidean},
						mesoLinkID:            mesoLinkID,
						macroLinkID:           mesoLink.macroLinkID,
						macroNodeID:           mesoLink.macroNodeID,
						mesoLinkType:          mesoLink.linkType,
						freeSpeed:             mesoLink.freeSpeed,
						additionalTravelCost:  0.0,
						laneID:                sourceNode.laneID,
						capacity:              mesoLink.capacity,
						cellType:              LINK_FORWARD,
						allowedAgentTypes:     make([]AgentType, len(multiModalAgentTypes)),
						isFirstMovement:       false,
						controlType:           mesoLink.controlType,
						movementCompositeType: MOVEMENT_NONE, // Could be changed later
					}
					microLink.lengthMeters = geo.LengthHaversign(microLink.geom)
					copy(microLink.allowedAgentTypes, multiModalAgentTypes)
					microscopic.links[lastLinkID] = &microLink
					lastLinkID++
					sourceNode.outcomingLinks = append(sourceNode.outcomingLinks, microLink.ID)
					targetNode.incomingLinks = append(targetNode.incomingLinks, microLink.ID)
				}
				// Lane change (left)
				if i <= mesoLink.lanesNum-2 {
					for j := 0; j < len(mesoLink.microNodesPerLane[i])-1; j++ {
						sourceNodeID := mesoLink.microNodesPerLane[i][j]
						targetNodeID := mesoLink.microNodesPerLane[i+1][j+1]
						sourceNode, ok := microscopic.nodes[sourceNodeID]
						if !ok {
							return nil, fmt.Errorf("genMicroscopicNetwork(): Source microscopic node %d for then left turn not found for mesoscopic link %d", sourceNodeID, mesoLinkID)
						}
						targetNode, ok := microscopic.nodes[targetNodeID]
						if !ok {
							return nil, fmt.Errorf("genMicroscopicNetwork(): Target microscopic node %d for then left turn not found for mesoscopic link %d", targetNodeID, mesoLinkID)
						}
						microLink := NetworkLinkMicroscopic{
							ID:                    lastLinkID,
							sourceNodeID:          sourceNodeID,
							targetNodeID:          targetNodeID,
							geom:                  orb.LineString{sourceNode.geom, targetNode.geom},
							geomEuclidean:         orb.LineString{sourceNode.geomEuclidean, targetNode.geomEuclidean},
							mesoLinkID:            mesoLinkID,
							macroLinkID:           mesoLink.macroLinkID,
							macroNodeID:           mesoLink.macroNodeID,
							mesoLinkType:          mesoLink.linkType,
							freeSpeed:             mesoLink.freeSpeed,
							capacity:              mesoLink.capacity,
							additionalTravelCost:  0.0,
							laneID:                sourceNode.laneID,
							cellType:              LINK_LANE_CHANGE,
							allowedAgentTypes:     make([]AgentType, len(multiModalAgentTypes)),
							isFirstMovement:       false,
							controlType:           mesoLink.controlType,
							movementCompositeType: MOVEMENT_NONE, // Could be changed later
						}
						microLink.lengthMeters = geo.LengthHaversign(microLink.geom)
						copy(microLink.allowedAgentTypes, multiModalAgentTypes)
						microscopic.links[lastLinkID] = &microLink
						lastLinkID++
						sourceNode.outcomingLinks = append(sourceNode.outcomingLinks, microLink.ID)
						targetNode.incomingLinks = append(targetNode.incomingLinks, microLink.ID)
					}
				}
				// Lane change (right)
				if i >= 1 {
					for j := 0; j < len(mesoLink.microNodesPerLane[i])-1; j++ {
						sourceNodeID := mesoLink.microNodesPerLane[i][j]
						targetNodeID := mesoLink.microNodesPerLane[i-1][j+1]
						sourceNode, ok := microscopic.nodes[sourceNodeID]
						if !ok {
							return nil, fmt.Errorf("genMicroscopicNetwork(): Source microscopic node %d for then right turn not found for mesoscopic link %d", sourceNodeID, mesoLinkID)
						}
						targetNode, ok := microscopic.nodes[targetNodeID]
						if !ok {
							return nil, fmt.Errorf("genMicroscopicNetwork(): Target microscopic node %d for the right turn not found for mesoscopic link %d", targetNodeID, mesoLinkID)
						}
						microLink := NetworkLinkMicroscopic{
							ID:                    lastLinkID,
							sourceNodeID:          sourceNodeID,
							targetNodeID:          targetNodeID,
							geom:                  orb.LineString{sourceNode.geom, targetNode.geom},
							geomEuclidean:         orb.LineString{sourceNode.geomEuclidean, targetNode.geomEuclidean},
							mesoLinkID:            mesoLinkID,
							macroLinkID:           mesoLink.macroLinkID,
							macroNodeID:           mesoLink.macroNodeID,
							mesoLinkType:          mesoLink.linkType,
							freeSpeed:             mesoLink.freeSpeed,
							capacity:              mesoLink.capacity,
							additionalTravelCost:  0.0,
							laneID:                sourceNode.laneID,
							cellType:              LINK_LANE_CHANGE,
							allowedAgentTypes:     make([]AgentType, len(multiModalAgentTypes)),
							isFirstMovement:       false,
							controlType:           mesoLink.controlType,
							movementCompositeType: MOVEMENT_NONE, // Could be changed later
						}
						microLink.lengthMeters = geo.LengthHaversign(microLink.geom)
						copy(microLink.allowedAgentTypes, multiModalAgentTypes)
						microscopic.links[lastLinkID] = &microLink
						lastLinkID++
						sourceNode.outcomingLinks = append(sourceNode.outcomingLinks, microLink.ID)
						targetNode.incomingLinks = append(targetNode.incomingLinks, microLink.ID)
					}
				}
			}
			if bike {
				for i := 0; i < len(mesoLink.microNodesBikeLane)-1; i++ {
					sourceNodeID := mesoLink.microNodesBikeLane[i]
					targetNodeID := mesoLink.microNodesBikeLane[i+1]
					sourceNode, ok := microscopic.nodes[sourceNodeID]
					if !ok {
						return nil, fmt.Errorf("genMicroscopicNetwork(): Source microscopic node %d not found for BIKE mesoscopic link %d", sourceNodeID, mesoLinkID)
					}
					targetNode, ok := microscopic.nodes[targetNodeID]
					if !ok {
						return nil, fmt.Errorf("genMicroscopicNetwork(): Target microscopic node %d not found for BIKE mesoscopic link %d", targetNodeID, mesoLinkID)
					}
					microLink := NetworkLinkMicroscopic{
						ID:                    lastLinkID,
						sourceNodeID:          sourceNodeID,
						targetNodeID:          targetNodeID,
						geom:                  orb.LineString{sourceNode.geom, targetNode.geom},
						geomEuclidean:         orb.LineString{sourceNode.geomEuclidean, targetNode.geomEuclidean},
						mesoLinkID:            mesoLinkID,
						macroLinkID:           mesoLink.macroLinkID,
						macroNodeID:           mesoLink.macroNodeID,
						mesoLinkType:          mesoLink.linkType,
						freeSpeed:             mesoLink.freeSpeed,
						capacity:              mesoLink.capacity,
						additionalTravelCost:  0.0,
						laneID:                sourceNode.laneID,
						cellType:              LINK_FORWARD,
						allowedAgentTypes:     []AgentType{AGENT_BIKE},
						isFirstMovement:       false,
						controlType:           mesoLink.controlType,
						movementCompositeType: MOVEMENT_NONE, // Could be changed later
					}
					microLink.lengthMeters = geo.LengthHaversign(microLink.geom)
					microscopic.links[lastLinkID] = &microLink
					lastLinkID++
					sourceNode.outcomingLinks = append(sourceNode.outcomingLinks, microLink.ID)
					targetNode.incomingLinks = append(targetNode.incomingLinks, microLink.ID)
				}
			}
			if walk {
				for i := 0; i < len(mesoLink.microNodesWalkLane)-1; i++ {
					sourceNodeID := mesoLink.microNodesWalkLane[i]
					targetNodeID := mesoLink.microNodesWalkLane[i+1]
					sourceNode, ok := microscopic.nodes[sourceNodeID]
					if !ok {
						return nil, fmt.Errorf("genMicroscopicNetwork(): Source microscopic node %d not found for WALK mesoscopic link %d", sourceNodeID, mesoLinkID)
					}
					targetNode, ok := microscopic.nodes[targetNodeID]
					if !ok {
						return nil, fmt.Errorf("genMicroscopicNetwork(): Target microscopic node %d not found for WALK mesoscopic link %d", targetNodeID, mesoLinkID)
					}
					microLink := NetworkLinkMicroscopic{
						ID:                    lastLinkID,
						sourceNodeID:          sourceNodeID,
						targetNodeID:          targetNodeID,
						geom:                  orb.LineString{sourceNode.geom, targetNode.geom},
						geomEuclidean:         orb.LineString{sourceNode.geomEuclidean, targetNode.geomEuclidean},
						mesoLinkID:            mesoLinkID,
						macroLinkID:           mesoLink.macroLinkID,
						macroNodeID:           mesoLink.macroNodeID,
						mesoLinkType:          mesoLink.linkType,
						freeSpeed:             mesoLink.freeSpeed,
						capacity:              mesoLink.capacity,
						additionalTravelCost:  0.0,
						laneID:                sourceNode.laneID,
						cellType:              LINK_FORWARD,
						allowedAgentTypes:     []AgentType{AGENT_WALK},
						isFirstMovement:       false,
						controlType:           mesoLink.controlType,
						movementCompositeType: MOVEMENT_NONE, // Could be changed later
					}
					microLink.lengthMeters = geo.LengthHaversign(microLink.geom)
					microscopic.links[lastLinkID] = &microLink
					lastLinkID++
					sourceNode.outcomingLinks = append(sourceNode.outcomingLinks, microLink.ID)
					targetNode.incomingLinks = append(targetNode.incomingLinks, microLink.ID)
				}
			}
		}
	}

	microscopic.maxNodeID = lastNodeID
	microscopic.maxLinkID = lastLinkID

	err := microscopic.connectLinks(macroNet, mesoNet)
	if err != nil {
		return nil, errors.Wrap(err, "Can't connect microscopic links for movement layer")
	}

	err = microscopic.updateBoundaryType(mesoNet)
	if err != nil {
		return nil, errors.Wrap(err, "Can't update boundary type for microscopic nodes")
	}

	// Update movement composite type
	// It could be done in previous loops, but it is more clear to do it here
	for _, link := range microscopic.links {
		if !link.isFirstMovement {
			link.movementCompositeType = MOVEMENT_NONE
			continue
		}
		mesoLink, ok := mesoNet.links[link.mesoLinkID]
		if !ok {
			return nil, fmt.Errorf("genMicroscopicNetwork(): Mesoscopic link %d for miscroscopic link %d not found", link.mesoLinkID, link.ID)
		}
		link.movementCompositeType = mesoLink.movementCompositeType
	}

	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}
	return &microscopic, nil
}

// connectLinks connects microscopic links via movements layer from both macroscopic and mesoscopic graphs
//
// generated connections between links are links too
//
func (microNet *NetworkMicroscopic) connectLinks(macroNet *NetworkMacroscopic, mesoNet *NetworkMesoscopic) error {
	lastNodeID := microNet.maxNodeID
	lastLinkID := microNet.maxLinkID

	// Iterate over all mesoscopic links and work with ones that contain movements
	for _, mesoLink := range mesoNet.links {
		// MovementID is not default, therefore this mesoscopic link is movement (from macroscopic node)
		if mesoLink.movementID > -1 {
			if mesoLink.movementLinkIncome < 0 || mesoLink.movementLinkOutcome < 0 {
				return fmt.Errorf("connectLinks(): Mesoscopic movement link %d has no income or outcome and movement is needed", mesoLink.ID)
			}
			if mesoLink.movementIncomeLaneStart < 0 || mesoLink.movementOutcomeLaneStart < 0 {
				return fmt.Errorf("connectLinks(): Mesoscopic movement link %d has no start lane index or end lane index and movement is needed", mesoLink.ID)
			}
			incomingMesoLink, ok := mesoNet.links[mesoLink.movementLinkIncome]
			if !ok {
				return fmt.Errorf("connectLinks(): Incoming mesoscopic link %d not found for mesoscopic movement link %d", mesoLink.movementLinkIncome, mesoLink.ID)
			}
			outcomingMesoLink, ok := mesoNet.links[mesoLink.movementLinkOutcome]
			if !ok {
				return fmt.Errorf("connectLinks(): Outcoming mesoscopic link %d not found for mesoscopic movement link %d", mesoLink.movementLinkOutcome, mesoLink.ID)
			}
			for i := 0; i < mesoLink.lanesNum; i++ {
				incomingMicroNodes := incomingMesoLink.microNodesPerLane[mesoLink.movementIncomeLaneStart+i]
				outcomingMicroNodes := outcomingMesoLink.microNodesPerLane[mesoLink.movementOutcomeLaneStart+i]

				startMicroNodeID := incomingMicroNodes[len(incomingMicroNodes)-1]
				endMicroNodeID := outcomingMicroNodes[0]

				startMicroNode, ok := microNet.nodes[startMicroNodeID]
				if !ok {
					return fmt.Errorf("connectLinks(): Incoming microscopic node %d not found for mesoscopic movement link %d on lane :%d", startMicroNodeID, mesoLink.ID, i)
				}
				endMicroNode, ok := microNet.nodes[endMicroNodeID]
				if !ok {
					return fmt.Errorf("connectLinks(): Outcoming microscopic node %d not found for mesoscopic movement link %d on lane :%d", endMicroNodeID, mesoLink.ID, i)
				}
				laneGeom := orb.LineString{startMicroNode.geom, endMicroNode.geom}
				laneLength := geo.LengthHaversign(laneGeom)

				// Calculate number of cell which fit into link
				// If cell length > link length then use only one cell
				cellsNum := math.Max(1.0, math.Round(laneLength/cellLength))
				laneNodes := []orb.Point{}
				laneNodesEuclidean := []orb.Point{}
				for j := 1; j < int(cellsNum); j++ {
					fraction := float64(j) / float64(cellsNum)
					distance := mesoLink.lengthMeters * fraction
					point, _ := geo.PointAtDistanceAlongLine(laneGeom, distance)
					laneNodes = append(laneNodes, point)
					laneNodesEuclidean = append(laneNodesEuclidean, pointToEuclidean(point))
				}

				// Prepare movement lanes
				laneNodesIDs := []NetworkNodeID{}
				lastMicroNodeID := startMicroNodeID // Track last node to connect it with next one
				firstMovement := true
				for geomIdx, nodeGeom := range laneNodes {
					// Create new microscopic node
					microNode := NetworkNodeMicroscopic{
						ID:                         lastNodeID,
						geom:                       nodeGeom,
						geomEuclidean:              laneNodesEuclidean[geomIdx],
						mesoLinkID:                 mesoLink.ID,
						laneID:                     i + 1,
						isLinkUpstreamTargetNode:   false,
						isLinkDownstreamTargetNode: false,
						zoneID:                     -1,
						boundaryType:               BOUNDARY_NONE,
					}
					laneNodesIDs = append(laneNodesIDs, microNode.ID)
					microNet.nodes[microNode.ID] = &microNode
					lastNodeID++

					// Create new miscroscopic link
					lastMicroNode, ok := microNet.nodes[lastMicroNodeID]
					if !ok {
						return fmt.Errorf("connectLinks(): Microscopic node %d not found for mesoscopic movement link %d on lane :%d", lastMicroNodeID, mesoLink.ID, i)
					}
					geom := orb.LineString{lastMicroNode.geom, microNode.geom}
					microLink := NetworkLinkMicroscopic{
						ID:                    lastLinkID,
						sourceNodeID:          lastMicroNodeID,
						targetNodeID:          microNode.ID,
						geom:                  geom,
						geomEuclidean:         orb.LineString{lastMicroNode.geomEuclidean, microNode.geomEuclidean},
						mesoLinkID:            mesoLink.ID,
						macroLinkID:           mesoLink.macroLinkID,
						macroNodeID:           mesoLink.macroNodeID,
						mesoLinkType:          mesoLink.linkType,
						freeSpeed:             mesoLink.freeSpeed,
						capacity:              mesoLink.capacity,
						additionalTravelCost:  0.0,
						laneID:                lastMicroNode.laneID,
						cellType:              LINK_FORWARD,
						isFirstMovement:       false,
						controlType:           mesoLink.controlType,
						movementCompositeType: MOVEMENT_NONE, // Could be changed later
					}
					microLink.lengthMeters = geo.LengthHaversign(microLink.geom)
					if firstMovement {
						microLink.isFirstMovement = true
						firstMovement = false
					}
					microNet.links[microLink.ID] = &microLink
					lastLinkID++
					lastMicroNode.outcomingLinks = append(lastMicroNode.outcomingLinks, microLink.ID)
					microNode.incomingLinks = append(microNode.incomingLinks, microLink.ID)

					// Go to next node
					lastMicroNodeID = microNode.ID
				}

				// Prepare very last microscopic link for each lane
				lastMicroNode, ok := microNet.nodes[lastMicroNodeID]
				if !ok {
					return fmt.Errorf("connectLinks(): Microscopic node %d not found for last mesoscopic movement link %d on lane :%d", lastMicroNodeID, mesoLink.ID, i)
				}
				geom := orb.LineString{lastMicroNode.geom, endMicroNode.geom}
				microLink := NetworkLinkMicroscopic{
					ID:                    lastLinkID,
					sourceNodeID:          lastMicroNodeID,
					targetNodeID:          endMicroNodeID,
					geom:                  geom,
					geomEuclidean:         orb.LineString{lastMicroNode.geomEuclidean, endMicroNode.geomEuclidean},
					mesoLinkID:            mesoLink.ID,
					macroLinkID:           mesoLink.macroLinkID,
					macroNodeID:           mesoLink.macroNodeID,
					mesoLinkType:          mesoLink.linkType,
					freeSpeed:             mesoLink.freeSpeed,
					capacity:              mesoLink.capacity,
					additionalTravelCost:  0.0,
					laneID:                lastMicroNode.laneID,
					cellType:              LINK_FORWARD,
					isFirstMovement:       false,
					controlType:           mesoLink.controlType,
					movementCompositeType: MOVEMENT_NONE, // Could be changed later
				}
				microLink.lengthMeters = geo.LengthHaversign(microLink.geom)
				if firstMovement {
					microLink.isFirstMovement = true
				}
				microNet.links[microLink.ID] = &microLink
				lastLinkID++
				lastMicroNode.outcomingLinks = append(lastMicroNode.outcomingLinks, microLink.ID)
				endMicroNode.incomingLinks = append(endMicroNode.incomingLinks, microLink.ID)

				// Add movement lane to mesoscopic link
				mesoLink.microNodesPerLane = append(mesoLink.microNodesPerLane, laneNodesIDs)
			}
		}
	}

	microNet.maxNodeID = lastNodeID
	microNet.maxLinkID = lastLinkID

	return microNet.fixGaps(macroNet, mesoNet)
}

// fixGaps fixes gaps between microscopic links
//
// Most of time there is one mesoscopic link between two intersections in one direction or two if there are two directions
// However, there are some cases when there are more than one consecutive mesoscopic links in same direction between two intersections (when there are some other macro nodes between two intersections)
//
func (microNet *NetworkMicroscopic) fixGaps(macroNet *NetworkMacroscopic, mesoNet *NetworkMesoscopic) error {
	// @TODO: this is copy-paste from mesoscopic network, should be refactored to avoid code duplication

	// Loop through each macroscopic
	for _, macroNode := range macroNet.nodes {
		// Loop through each movement for given node
		for _, movement := range macroNode.movements {
			// Extract macroscopic links
			incomingMacroLinkID, outcomingMacroLinkID := movement.IncomingLinkID, movement.OutcomingLinkID
			incomingMacroLink, ok := macroNet.links[incomingMacroLinkID]
			if !ok {
				return fmt.Errorf("fixGaps(): Incoming macroscopic link %d not found", incomingMacroLinkID)
			}
			outcomingMacroLink, ok := macroNet.links[outcomingMacroLinkID]
			if !ok {
				return fmt.Errorf("fixGaps(): Outcoming macroscopic link %d not found", outcomingMacroLinkID)
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
				return fmt.Errorf("fixGaps(): Incoming mesoscopic link %d not found", incomingMesoLinkID)
			}
			outcomingMesoLinkID := outcomingMacroLink.mesolinks[0]
			outcomingMesoLink, ok := mesoNet.links[outcomingMesoLinkID]
			if !ok {
				return fmt.Errorf("fixGaps(): Outcoming mesoscopic link %d not found", outcomingMesoLinkID)
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
				continue
			}
			// The same workflow as for mesoscopic links but for micro links. The geometry re-evaluation is not needed since we want to have constant geometry length (see `cellLength``)
			if incomingMacroLink.downstreamIsTarget && !outcomingMacroLink.upstreamIsTarget {
				for i := 0; i < lanesNum; i++ {
					incomeLaneIndex := incomeLaneStart + i
					outcomeLaneIndex := outcomeLaneStart + i

					incomeLane := incomingMesoLink.microNodesPerLane[incomeLaneIndex]
					outcomeLane := outcomingMesoLink.microNodesPerLane[outcomeLaneIndex]

					incomeMesoLinkOutMicroNodeID := incomeLane[len(incomeLane)-1]
					outcomeMesoLinkInMicroNodeID := outcomeLane[0]

					outcomeMesoLinkInMicroNode, ok := microNet.nodes[outcomeMesoLinkInMicroNodeID]
					if !ok {
						return fmt.Errorf("fixGaps(): Microscopic node %d at lane %d for outcoming mesoscopic link %d not found", outcomeMesoLinkInMicroNodeID, outcomeLaneIndex, outcomingMesoLinkID)
					}
					for _, microLinkID := range outcomeMesoLinkInMicroNode.outcomingLinks {
						microLink, ok := microNet.links[microLinkID]
						if !ok {
							return fmt.Errorf("fixGaps(): Outcoming microscopic link %d for microscopic node %d not found", microLinkID, outcomeMesoLinkInMicroNodeID)
						}
						microLink.sourceNodeID = incomeMesoLinkOutMicroNodeID
					}
					delete(microNet.nodes, outcomeMesoLinkInMicroNodeID)
				}
			} else if !incomingMacroLink.downstreamIsTarget && outcomingMacroLink.upstreamIsTarget {
				for i := 0; i < lanesNum; i++ {
					incomeLaneIndex := incomeLaneStart + i
					outcomeLaneIndex := outcomeLaneStart + i

					incomeLane := incomingMesoLink.microNodesPerLane[incomeLaneIndex]
					outcomeLane := outcomingMesoLink.microNodesPerLane[outcomeLaneIndex]

					incomeMesoLinkOutMicroNodeID := incomeLane[len(incomeLane)-1]
					outcomeMesoLinkInMicroNodeID := outcomeLane[0]

					incomeMesoLinkOutMicroNode, ok := microNet.nodes[incomeMesoLinkOutMicroNodeID]
					if !ok {
						return fmt.Errorf("fixGaps(): Microscopic node %d at lane %d for incoming mesoscopic link %d not found", incomeMesoLinkOutMicroNodeID, incomeLaneIndex, incomingMesoLinkID)
					}
					for _, microLinkID := range incomeMesoLinkOutMicroNode.incomingLinks {
						microLink, ok := microNet.links[microLinkID]
						if !ok {
							return fmt.Errorf("fixGaps(): Incoming microscopic link %d for microscopic node %d not found", microLinkID, incomeMesoLinkOutMicroNodeID)
						}
						microLink.targetNodeID = outcomeMesoLinkInMicroNodeID
					}
					delete(microNet.nodes, incomeMesoLinkOutMicroNodeID)
				}
			}
		}

	}
	return nil
}

// prepareBikeWalkAgents returns a list of agent types that should be used for the link
func prepareBikeWalkAgents(agentTypes []AgentType) (main []AgentType, bike bool, walk bool) {
	if len(agentTypes) == 0 {
		main := make([]AgentType, len(agentTypes))
		copy(main, agentTypes)
		return main, false, false
	}
	standart := map[AgentType]bool{
		AGENT_AUTO: false,
		AGENT_BIKE: false,
		AGENT_WALK: false,
	}
	for _, agent := range agentTypes {
		if _, ok := standart[agent]; ok {
			standart[agent] = true
		}
	}
	if standart[AGENT_AUTO] == true && standart[AGENT_BIKE] == true {
		return []AgentType{AGENT_AUTO}, true, false
	} else if standart[AGENT_AUTO] == true && standart[AGENT_WALK] == true {
		return []AgentType{AGENT_AUTO}, false, true
	} else if standart[AGENT_BIKE] == true && standart[AGENT_WALK] == true {
		return []AgentType{AGENT_BIKE}, false, true
	}
	return []AgentType{AGENT_AUTO}, true, true
}

// updateBoundaryType updates boundary type for each microscopic node
//
// this function should be called after all incident edges for nodes are set
//
func (microNet *NetworkMicroscopic) updateBoundaryType(mesoNet *NetworkMesoscopic) error {
	for _, microNode := range microNet.nodes {
		if microNode.mesoLinkID == -1 {
			microNode.boundaryType = BOUNDARY_NONE
			continue
		}
		mesoLink, ok := mesoNet.links[microNode.mesoLinkID]
		if !ok {
			return fmt.Errorf("connectNodes(): Mesoscopic link %d not found for microscopic node %d", microNode.mesoLinkID, microNode.ID)
		}
		mesoLinkSourceNodeID := mesoLink.sourceNodeID
		mesoLinkSourceNode, ok := mesoNet.nodes[mesoLinkSourceNodeID]
		if !ok {
			return fmt.Errorf("connectNodes(): Mesoscopic node %d not found for mesoscopic link %d for microscopic node %d", mesoLinkSourceNodeID, mesoLink.ID, microNode.ID)
		}
		if microNode.isLinkUpstreamTargetNode {
			microNode.boundaryType = mesoLinkSourceNode.boundaryType
		} else if microNode.isLinkDownstreamTargetNode {
			mesoLinkTargetNodeID := mesoLink.targetNodeID
			mesoLinkTargetNode, ok := mesoNet.nodes[mesoLinkTargetNodeID]
			if !ok {
				return fmt.Errorf("connectNodes(): Mesoscopic node %d not found for mesoscopic link %d for microscopic node %d", mesoLinkTargetNodeID, mesoLink.ID, microNode.ID)
			}
			microNode.boundaryType = mesoLinkTargetNode.boundaryType

		} else {
			microNode.boundaryType = BOUNDARY_NONE
		}
	}
	return nil
}
