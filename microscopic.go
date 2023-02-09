package osm2ch

import (
	"fmt"
	"time"

	"github.com/paulmach/orb"
)

const (
	bikeLaneWidth = 0.5
	walkLaneWidth = 0.5
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

	// Iterate over macroscopic links
	for _, macroLink := range macroNet.links {
		fmt.Println("create data for link", macroLink.ID)
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

			fmt.Println("\tmesolink", mesoLinkID)

			laneGeometries := []orb.LineString{}
			bikeGeometry := orb.LineString{}
			walkGeometry := orb.LineString{}
			laneOffset := 0.0
			// Iterate over mesoscopic link lanes and prepare geometries
			for i := 0; i < mesoLink.lanesNum; i++ {
				laneOffset := (lanesNumberInBetween + float64(i)) * laneWidth
				fmt.Println("\titerate lane", i, laneOffset)
				// If offset is too small then neglect it and copy original geometry
				// Otherwise evaluate geometry
				if !(laneOffset < -1e-2 || laneOffset > 1e-2) {
					laneGeometries = append(laneGeometries, mesoLink.geom.Clone())
				} else {

				}
				// @TODO: continue
			}
			if bike && !walk {
				// Prepare only bike geometry: calculate offset and evaluate geometry
				bikeLaneOffset := laneOffset + bikeLaneWidth
				if !(bikeLaneOffset < -1e-2 || bikeLaneOffset > 1e-2) {
					bikeGeometry = mesoLink.geom.Clone()
				} else {
					// @TODO: continue
				}
			} else if !bike && walk {
				// Prepare only walk geometry: calculate offset and evaluate geometry
				walkLaneOffset := laneOffset + walkLaneWidth
				if !(walkLaneOffset < -1e-2 || walkLaneOffset > 1e-2) {
					walkGeometry = mesoLink.geom.Clone()
				} else {
					// @TODO: continue
				}
			} else if bike && walk {
				// Prepare both bike and walk geometry: calculate two offsets and evaluate geometries
				bikeLaneOffset := laneOffset + bikeLaneWidth
				walkLaneOffset := laneOffset + walkLaneWidth
				if !(bikeLaneOffset < -1e-2 || bikeLaneOffset > 1e-2) {
					bikeGeometry = mesoLink.geom.Clone()
				} else {
					// @TODO: continue
				}
				if !(walkLaneOffset < -1e-2 || walkLaneOffset > 1e-2) {
					walkGeometry = mesoLink.geom.Clone()
				} else {
					// @TODO: continue
				}
			}
			_ = bikeGeometry
			_ = walkGeometry
		}
	}
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
