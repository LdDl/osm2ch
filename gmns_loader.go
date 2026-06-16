package osm2ch

import (
	"fmt"
	"time"

	"github.com/LdDl/go-gmns/generators"
	"github.com/LdDl/go-gmns/gmns"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/go-gmns/macro"
	"github.com/LdDl/go-gmns/movement"
	"github.com/LdDl/osm2gmns"
	"github.com/paulmach/orb"
)

// GMNSNetworks holds the intermediate GMNS networks produced while building the
// expanded graph. They are exposed so callers can optionally export GMNS CSV
// files (macro / movement) or generate and export the derived meso / micro
// networks without re-parsing the OSM data.
type GMNSNetworks struct {
	Macro     *macro.Net
	Movements movement.MovementsStorage
}

// ImportFromGMNS imports graph using osm2gmns library to build macro network + movements,
// then performs edge expansion based on movements.
func ImportFromGMNS(fileName string, agentTypes []types.AgentType) ([]ExpandedEdge, error) {
	expandedEdges, _, err := ImportFromGMNSWithNetworks(fileName, agentTypes)
	return expandedEdges, err
}

// ImportFromGMNSWithNetworks behaves like ImportFromGMNS but additionally returns
// the underlying GMNS networks (macro network and movements) so the caller can
// optionally export them as GMNS CSV files.
func ImportFromGMNSWithNetworks(fileName string, agentTypes []types.AgentType) ([]ExpandedEdge, *GMNSNetworks, error) {
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
		return nil, nil, fmt.Errorf("osm2gmns ReadOSM: %w", err)
	}
	fmt.Printf("Done in %v\n", time.Since(st))

	// 2. Generate macroscopic network
	fmt.Println("Generating macroscopic network...")
	st = time.Now()
	macroNet, err := osm2gmns.GenerateMacroscopic(osmData, false)
	if err != nil {
		return nil, nil, fmt.Errorf("osm2gmns GenerateMacroscopic: %w", err)
	}
	fmt.Printf("Done in %v\n\tLinks: %d\n\tNodes: %d\n", time.Since(st), len(macroNet.Links), len(macroNet.Nodes))

	// 3. Generate movements
	fmt.Println("Generating movements...")
	st = time.Now()
	movements, err := generators.GenerateMovements(macroNet)
	if err != nil {
		return nil, nil, fmt.Errorf("osm2gmns GenerateMovements: %w", err)
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
			ID:              EdgeID(link.ID),
			WayID:           link.OSMWay(),
			SourceNodeID:    link.SourceNode(),
			TargetNodeID:    link.TargetNode(),
			SourceOSMNodeID: link.SourceOSMNode(),
			TargetOSMNodeID: link.TargetOSMNode(),
			WasOneway:       !link.WasBidirectional(),
			LengthMeters:    lengthMeters,
			FreeSpeed:       freeSpeed,
			Geom:            geom,
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

		// Expanded edge geometry consists of the second half of the incoming macro link
		// and the first half of the outgoing macro link. Hence the total length is the
		// sum of the halves, which equals the arithmetic mean of the two full lengths.
		inHalf := inEdge.LengthMeters / 2.0
		outHalf := outEdge.LengthMeters / 2.0
		totalLength := inHalf + outHalf

		// Free-flow speed is combined as a length-weighted harmonic mean so that
		// travel time is preserved: totalLength / freeSpeed == inHalf/inSpeed + outHalf/outSpeed.
		// Arithmetic mean would over-estimate the effective speed when the two halves
		// differ substantially (e.g. motorway_link 30 km/h => motorway 100 km/h).
		var freeSpeed float64
		switch {
		case inEdge.FreeSpeed > 0 && outEdge.FreeSpeed > 0:
			freeSpeed = totalLength / (inHalf/inEdge.FreeSpeed + outHalf/outEdge.FreeSpeed)
		case inEdge.FreeSpeed > 0:
			freeSpeed = inEdge.FreeSpeed
		case outEdge.FreeSpeed > 0:
			freeSpeed = outEdge.FreeSpeed
		}

		expandedEdgesTotal++
		expandedEdges = append(expandedEdges, ExpandedEdge{
			ID:             expandedEdgesTotal,
			Source:         inEdge.ID,
			Target:         outEdge.ID,
			SourceOSMWayID: inEdge.WayID,
			TargetOSMWayID: outEdge.WayID,
			SourceComponent: expandedEdgeComponent{
				SourceNodeID:    inEdge.SourceNodeID,
				TargetNodeID:    inEdge.TargetNodeID,
				SourceOSMNodeID: inEdge.SourceOSMNodeID,
				TargetOSMNodeID: inEdge.TargetOSMNodeID,
			},
			TargeComponent: expandedEdgeComponent{
				SourceNodeID:    outEdge.SourceNodeID,
				TargetNodeID:    outEdge.TargetNodeID,
				SourceOSMNodeID: outEdge.SourceOSMNodeID,
				TargetOSMNodeID: outEdge.TargetOSMNodeID,
			},
			LengthMeters: totalLength,
			FreeSpeed:    freeSpeed,
			WasOneway:    inEdge.WasOneway,
			Geom:         completedNewGeom,
		})
	}

	fmt.Printf("Done in %v\n", time.Since(st))
	fmt.Printf("\tSkipped U-turns: %d\n", skippedUTurns)
	fmt.Printf("\tSkipped bad geometry: %d\n", skippedBadGeom)
	fmt.Printf("\tNumber of expanded edges: %d\n", expandedEdgesTotal)

	nets := &GMNSNetworks{
		Macro:     macroNet,
		Movements: movements,
	}

	return expandedEdges, nets, nil
}

// orbLineStringToGeoPoints converts orb.LineString to []GeoPoint
func orbLineStringToGeoPoints(ls orb.LineString) []GeoPoint {
	pts := make([]GeoPoint, len(ls))
	for i, p := range ls {
		pts[i] = GeoPoint{Lon: p[0], Lat: p[1]}
	}
	return pts
}
