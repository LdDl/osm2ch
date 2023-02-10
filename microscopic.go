package osm2ch

import (
	"fmt"
	"math"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
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

			laneGeometriesEuclidean := []orb.LineString{}
			laneGeometries := []orb.LineString{}

			bikeGeometryEuclidean := orb.LineString{}
			bikeGeometry := orb.LineString{}

			walkGeometryEuclidean := orb.LineString{}
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
					if laneOffset > 0 {
						laneGeomEuclidean.Reverse()
					}
					laneGeometriesEuclidean = append(laneGeometriesEuclidean, laneGeomEuclidean)
					laneGeometries = append(laneGeometries, lineToSpherical(laneGeomEuclidean))
				} else {
					laneGeometriesEuclidean = append(laneGeometriesEuclidean, mesoLink.geomEuclidean.Clone())
					laneGeometries = append(laneGeometries, mesoLink.geom.Clone())
				}
			}
			if bike && !walk {
				// Prepare only bike geometry: calculate offset and evaluate geometry
				bikeLaneOffset := laneOffset + bikeLaneWidth
				if bikeLaneOffset < -1e-2 || bikeLaneOffset > 1e-2 {
					bikeGeometryEuclidean = offsetCurve(mesoLink.geomEuclidean, -bikeLaneOffset)
					if bikeLaneOffset > 0 {
						bikeGeometryEuclidean.Reverse()
					}
					bikeGeometry = lineToSpherical(bikeGeometryEuclidean)
				} else {
					bikeGeometryEuclidean = mesoLink.geomEuclidean.Clone()
					bikeGeometry = mesoLink.geom.Clone()
				}
			} else if !bike && walk {
				// Prepare only walk geometry: calculate offset and evaluate geometry
				walkLaneOffset := laneOffset + walkLaneWidth
				if walkLaneOffset < -1e-2 || walkLaneOffset > 1e-2 {
					walkGeometryEuclidean = offsetCurve(mesoLink.geomEuclidean, -walkLaneOffset)
					if walkLaneOffset > 0 {
						walkGeometryEuclidean.Reverse()
					}
					walkGeometry = lineToSpherical(walkGeometryEuclidean)
				} else {
					walkGeometryEuclidean = mesoLink.geomEuclidean.Clone()
					walkGeometry = mesoLink.geom.Clone()
				}
			} else if bike && walk {
				// Prepare both bike and walk geometry: calculate two offsets and evaluate geometries
				bikeLaneOffset := laneOffset + bikeLaneWidth
				walkLaneOffset := laneOffset + walkLaneWidth
				if bikeLaneOffset < -1e-2 || bikeLaneOffset > 1e-2 {
					bikeGeometryEuclidean = offsetCurve(mesoLink.geomEuclidean, -bikeLaneOffset)
					if bikeLaneOffset > 0 {
						bikeGeometryEuclidean.Reverse()
					}
					bikeGeometry = lineToSpherical(bikeGeometryEuclidean)
				} else {
					bikeGeometryEuclidean = mesoLink.geomEuclidean.Clone()
					bikeGeometry = mesoLink.geom.Clone()
				}
				if walkLaneOffset < -1e-2 || walkLaneOffset > 1e-2 {
					walkGeometryEuclidean = offsetCurve(mesoLink.geomEuclidean, -walkLaneOffset)
					if walkLaneOffset > 0 {
						walkGeometryEuclidean.Reverse()
					}
					walkGeometry = lineToSpherical(walkGeometryEuclidean)
				} else {
					walkGeometryEuclidean = mesoLink.geomEuclidean.Clone()
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
				laneNodes := []NetworkNodeID{}
				for j, microNodeGeom := range microNodesGeometries[i] {
					microNode := NetworkNodeMicroscopic{
						ID:                         lastNodeID,
						geom:                       microNodeGeom,
						geomEuclidead:              microNodesGeometriesEuclidean[i][j],
						mesoLinkID:                 mesoLink.ID,
						laneID:                     i + 1,
						isLinkUpstreamTargetNode:   false,
						isLinkDownstreamTargetNode: false,
					}
					laneNodes = append(laneNodes, microNode.ID)
					microscopic.nodes[microNode.ID] = &microNode
					lastNodeID++
				}
				mesoLink.microNodesPerLane = append(mesoLink.microNodesPerLane, laneNodes)
			}
			if bike {
				for j, microNodeGeom := range bikeMicroNodesGeometries {
					microNode := NetworkNodeMicroscopic{
						ID:                         lastNodeID,
						geom:                       microNodeGeom,
						geomEuclidead:              bikeMicroNodesGeometriesEuclidean[j],
						mesoLinkID:                 mesoLink.ID,
						laneID:                     -1,
						isLinkUpstreamTargetNode:   false,
						isLinkDownstreamTargetNode: false,
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
						geomEuclidead:              walkMicroNodesGeometriesEuclidean[j],
						mesoLinkID:                 mesoLink.ID,
						laneID:                     -2,
						isLinkUpstreamTargetNode:   false,
						isLinkDownstreamTargetNode: false,
					}
					microscopic.nodes[microNode.ID] = &microNode
					mesoLink.microNodesWalkLane = append(mesoLink.microNodesWalkLane, microNode.ID)
					lastNodeID++
				}
			}
		}
		fmt.Println(lastNodeID)
	}

	microscopic.maxNodeID = lastNodeID
	if verbose {
		fmt.Printf("Done in %v\n", time.Since(st))
	}
	return &microscopic, nil
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
