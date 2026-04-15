package osm2ch

import (
	"fmt"
	"time"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/movement"
	"github.com/LdDl/osm2gmns"
	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
)

// ImportFromGMNS imports graph using osm2gmns library to build macro network + movements,
// then performs edge expansion based on movements.
func ImportFromGMNS(fileName string, agentTypes []types.AgentType) ([]ExpandedEdge, error) {
	// 1. Parse PBF via osm2gmns
	parser := osm2gmns.NewParser(
		fileName,
		osm2gmns.WithVerbose(true),
		osm2gmns.WithAllowedAgentTypes(agentTypes),
	)

	fmt.Println("Reading OSM data via osm2gmns...")
	st := time.Now()
	osmData, err := parser.ReadOSM()
	if err != nil {
		return nil, fmt.Errorf("osm2gmns ReadOSM: %w", err)
	}
	fmt.Printf("Done in %v\n", time.Since(st))

	// 2. Generate macroscopic network
	fmt.Println("Generating macroscopic network...")
	st = time.Now()
	macroNet, err := osm2gmns.GenerateMacroscopic(osmData, false)
	if err != nil {
		return nil, fmt.Errorf("osm2gmns GenerateMacroscopic: %w", err)
	}
	fmt.Printf("Done in %v\n\tLinks: %d\n\tNodes: %d\n", time.Since(st), len(macroNet.Links), len(macroNet.Nodes))

	// 3. Generate movements
	fmt.Println("Generating movements...")
	st = time.Now()
	movements, err := generators.GenerateMovements(macroNet)
	if err != nil {
		return nil, fmt.Errorf("osm2gmns GenerateMovements: %w", err)
	}
	fmt.Printf("Done in %v\n\tMovements: %d\n", time.Since(st), len(movements))

	// 4. Convert macro links to internal Edge representation
	fmt.Println("Converting macro links to edges...")
	st = time.Now()
	edges := make(map[gmns.LinkID]Edge, len(macroNet.Links))
	for _, link := range macroNet.Links {
		geom := orbLineStringToGeoPoints(link.Geom())
		lengthMeters := link.LengthMeters()
		freeSpeed := link.FreeSpeed()
		edges[link.ID] = Edge{
			ID:           EdgeID(link.ID),
			WayID:        link.OSMWay(),
			SourceNodeID: osm.NodeID(link.SourceNode()),
			TargetNodeID: osm.NodeID(link.TargetNode()),
			WasOneway:    !link.WasBidirectional(),
			LengthMeters: lengthMeters,
			FreeSpeed:    freeSpeed,
			Geom:         geom,
		}
	}
	fmt.Printf("Done in %v\n\tEdges: %d\n", time.Since(st), len(edges))

	// 5. Edge expansion based on movements
	fmt.Println("Applying edge expansion from movements...")
	st = time.Now()

	expandedEdges := make([]ExpandedEdge, 0, len(movements))
	expandedEdgesTotal := int64(0)
	skippedUTurns := 0
	skippedBadGeom := 0

	for _, mvmt := range movements {
		// Skip U-turns
		if mvmt.Type() == movement.MOVEMENT_TYPE_U_TURN {
			skippedUTurns++
			continue
		}

		inLinkID := mvmt.IncomeMacroLink()
		outLinkID := mvmt.OutcomeMacroLink()

		inEdge, ok := edges[inLinkID]
		if !ok {
			continue
		}
		outEdge, ok := edges[outLinkID]
		if !ok {
			continue
		}

		if len(inEdge.Geom) < 2 || len(outEdge.Geom) < 2 {
			skippedBadGeom++
			continue
		}

		// Build expanded edge geometry: second half of income + first half of outcome
		beforeFromIdx, fromMiddlePoint := findMiddlePoint(inEdge.Geom)
		fromGeomHalf := append([]GeoPoint{fromMiddlePoint}, inEdge.Geom[beforeFromIdx+1:]...)

		beforeToIdx, toMiddlePoint := findMiddlePoint(outEdge.Geom)
		toGeomHalf := append(make([]GeoPoint, 0, len(outEdge.Geom[:beforeToIdx+1])+1), outEdge.Geom[:beforeToIdx+1]...)
		toGeomHalf = append(toGeomHalf, toMiddlePoint)

		completedNewGeom := append(fromGeomHalf, toGeomHalf...)

		expandedEdgesTotal++
		expandedEdges = append(expandedEdges, ExpandedEdge{
			ID:             expandedEdgesTotal,
			Source:         inEdge.ID,
			Target:         outEdge.ID,
			SourceOSMWayID: inEdge.WayID,
			TargetOSMWayID: outEdge.WayID,
			SourceComponent: expandedEdgeComponent{
				SourceNodeID: inEdge.SourceNodeID,
				TargetNodeID: inEdge.TargetNodeID,
			},
			TargeComponent: expandedEdgeComponent{
				SourceNodeID: outEdge.SourceNodeID,
				TargetNodeID: outEdge.TargetNodeID,
			},
			LengthMeters: (inEdge.LengthMeters + outEdge.LengthMeters) / 2.0,
			FreeSpeed:    (inEdge.FreeSpeed + outEdge.FreeSpeed) / 2.0,
			WasOneway:    inEdge.WasOneway,
			Geom:         completedNewGeom,
		})
	}

	fmt.Printf("Done in %v\n", time.Since(st))
	fmt.Printf("\tSkipped U-turns: %d\n", skippedUTurns)
	fmt.Printf("\tSkipped bad geometry: %d\n", skippedBadGeom)
	fmt.Printf("\tNumber of expanded edges: %d\n", expandedEdgesTotal)

	return expandedEdges, nil
}

// orbLineStringToGeoPoints converts orb.LineString to []GeoPoint
func orbLineStringToGeoPoints(ls orb.LineString) []GeoPoint {
	pts := make([]GeoPoint, len(ls))
	for i, p := range ls {
		pts[i] = GeoPoint{Lon: p[0], Lat: p[1]}
	}
	return pts
}
