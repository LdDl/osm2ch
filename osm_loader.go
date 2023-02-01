package osm2ch

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/paulmach/osm"
	"github.com/pkg/errors"

	"github.com/paulmach/osm/osmpbf"
)

// ImportFromOSMFile Imports graph from file of PBF-format (in OSM terms)
/*
	File should have PBF (Protocolbuffer Binary Format) extension according to https://github.com/paulmach/osm
*/
func ImportFromOSMFile(fileName string, cfg *OsmConfiguration) ([]ExpandedEdge, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, errors.Wrap(err, "File open")
	}
	defer f.Close()

	scannerWays := osmpbf.New(context.Background(), f, 4)
	defer scannerWays.Close()

	ways := []Way{}
	nodes := make(map[osm.NodeID]Node)
	nodesSeen := make(map[osm.NodeID]struct{})

	fmt.Printf("Scanning ways...")
	st := time.Now()
	for scannerWays.Scan() {
		obj := scannerWays.Object()
		if obj.ObjectID().Type() != "way" {
			continue
		}
		way := obj.(*osm.Way)
		tagMap := way.TagMap()
		tag, ok := tagMap[cfg.EntityName]
		if !ok {
			continue
		}
		if !cfg.CheckTag(tag) {
			continue
		}
		oneway := false
		if v, ok := tagMap["oneway"]; ok {
			if v == "yes" || v == "1" {
				oneway = true
			}
		}
		nodes := way.Nodes
		preparedWay := Way{
			ID:     way.ID,
			Nodes:  make(osm.WayNodes, len(nodes)),
			Oneway: oneway,
			TagMap: make(osm.Tags, len(way.Tags)),
		}
		copy(preparedWay.Nodes, nodes)
		copy(preparedWay.TagMap, way.Tags)
		ways = append(ways, preparedWay)
		for _, node := range nodes {
			nodesSeen[node.ID] = struct{}{}
		}
	}
	if scannerWays.Err() != nil {
		return nil, errors.Wrap(scannerWays.Err(), "Scanner error on Ways")
	}
	fmt.Printf("Done in %v\n\tWays: %d\n", time.Since(st), len(ways))

	// Seek file to start
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(err, "Can't repeat seeking after ways scanning")
	}
	scannerNodes := osmpbf.New(context.Background(), f, 4)
	defer scannerNodes.Close()

	fmt.Printf("Scanning nodes...")
	st = time.Now()
	for scannerNodes.Scan() {
		obj := scannerNodes.Object()
		if obj.ObjectID().Type() != "node" {
			continue
		}
		node := obj.(*osm.Node)
		if _, ok := nodesSeen[node.ID]; ok {
			delete(nodesSeen, node.ID)
			nodes[node.ID] = Node{
				ID:       node.ID,
				useCount: 0,
				node:     *node,
			}
		}
	}
	if scannerNodes.Err() != nil {
		return nil, errors.Wrap(scannerNodes.Err(), "Scanner error on Nodes")
	}
	fmt.Printf("Done in %v\n\tNodes: %d\n", time.Since(st), len(nodes))

	// Seek file to start
	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return nil, errors.Wrap(err, "Can't repeat seeking after nodes scanning")
	}
	scannerManeuvers := osmpbf.New(context.Background(), f, 4)
	defer scannerManeuvers.Close()
	fmt.Printf("Scanning maneuvers (restrictions)...")
	st = time.Now()
	skippedRestrictions := 0
	unsupportedRestrictionRoles := 0
	possibleRestrictionCombos := make(map[string]map[string]bool)
	restrictions := make(map[string]map[restrictionComponent]map[restrictionComponent]restrictionComponent)
	for scannerManeuvers.Scan() {
		obj := scannerManeuvers.Object()
		if obj.ObjectID().Type() == "relation" {
			relation := obj.(*osm.Relation)
			tagMap := relation.TagMap()
			tag, ok := tagMap["restriction"]
			if !ok {
				continue
			}
			members := relation.Members
			if len(members) != 3 {
				skippedRestrictions++
				// fmt.Printf("Restriction does not contain 3 members, relation ID: %d. Skip it\n", relation.ID)
				continue
			}
			firstMember := restrictionComponent{-1, ""}
			secondMember := restrictionComponent{-1, ""}
			thirdMember := restrictionComponent{-1, ""}

			switch members[0].Role {
			case "from":
				firstMember = restrictionComponent{members[0].Ref, string(members[0].Type)}
				break
			case "via":
				thirdMember = restrictionComponent{members[0].Ref, string(members[0].Type)}
				break
			case "to":
				secondMember = restrictionComponent{members[0].Ref, string(members[0].Type)}
				break
			default:
				unsupportedRestrictionRoles++
				// fmt.Printf("Something went wrong for first member of relation with ID: %d\n", relation.ID)
				break
			}

			switch members[1].Role {
			case "from":
				firstMember = restrictionComponent{members[1].Ref, string(members[1].Type)}
				break
			case "via":
				thirdMember = restrictionComponent{members[1].Ref, string(members[1].Type)}
				break
			case "to":
				secondMember = restrictionComponent{members[1].Ref, string(members[1].Type)}
				break
			default:
				unsupportedRestrictionRoles++
				// fmt.Printf("Something went wrong for second member of relation with ID: %d\n", relation.ID)
				break
			}

			switch members[2].Role {
			case "from":
				firstMember = restrictionComponent{members[2].Ref, string(members[2].Type)}
				break
			case "via":
				thirdMember = restrictionComponent{members[2].Ref, string(members[2].Type)}
				break
			case "to":
				secondMember = restrictionComponent{members[2].Ref, string(members[2].Type)}
				break
			default:
				unsupportedRestrictionRoles++
				// fmt.Printf("Something went wrong for third member of relation with ID: %d\n", relation.ID)
				break
			}
			if _, ok := possibleRestrictionCombos[tag]; !ok {
				possibleRestrictionCombos[tag] = make(map[string]bool)
			}
			possibleRestrictionCombos[tag][fmt.Sprintf("%s;%s;%s", firstMember.Type, secondMember.Type, thirdMember.Type)] = true

			if _, ok := restrictions[tag]; !ok {
				restrictions[tag] = make(map[restrictionComponent]map[restrictionComponent]restrictionComponent)
			}
			if _, ok := restrictions[tag][firstMember]; !ok {
				restrictions[tag][firstMember] = make(map[restrictionComponent]restrictionComponent)
			}
			if _, ok := restrictions[tag][firstMember][secondMember]; !ok {
				restrictions[tag][firstMember][secondMember] = thirdMember
			}
		}
	}
	fmt.Printf("Done in %v\n", time.Since(st))
	fmt.Printf("\tSkipped restrictions (which have not exactly 3 members): %d\n", skippedRestrictions)
	fmt.Printf("\tNumber of unknow restriction roles (only 'from', 'to' and 'via' supported): %d\n", unsupportedRestrictionRoles)

	fmt.Printf("Counting node use cases...")
	st = time.Now()
	for _, way := range ways {
		for i, wayNode := range way.Nodes {
			if node, ok := nodes[wayNode.ID]; ok {
				if i == 0 || i == len(way.Nodes)-1 {
					node.useCount += 2
					nodes[wayNode.ID] = node
				} else {
					node.useCount += 1
					nodes[wayNode.ID] = node
				}
			} else {
				return nil, fmt.Errorf("Missing node with id: %d\n", wayNode.ID)
			}
		}
	}
	fmt.Printf("Done in %v\n", time.Since(st))

	fmt.Printf("Preparing edges...")
	st = time.Now()
	edges := []Edge{}
	onewayEdges := 0
	notOnewayEdges := 0
	totalEdgesNum := int64(0)
	waysSeen := make(map[osm.WayID]struct{})
	for _, way := range ways {
		var source osm.NodeID
		waysSeen[way.ID] = struct{}{}
		geometry := []GeoPoint{}
		for i, wayNode := range way.Nodes {
			node := nodes[wayNode.ID]
			if i == 0 {
				source = wayNode.ID
				geometry = append(geometry, GeoPoint{Lon: node.node.Lon, Lat: node.node.Lat})
			} else {
				geometry = append(geometry, GeoPoint{Lon: node.node.Lon, Lat: node.node.Lat})
				if node.useCount > 1 {
					totalEdgesNum++
					onewayEdges++
					cost := getSphericalLength(geometry)
					edges = append(edges, Edge{
						ID:           EdgeID(totalEdgesNum),
						WayID:        way.ID,
						SourceNodeID: source,
						TargetNodeID: wayNode.ID,
						CostMeters:   cost,
						Geom:         copyLine(geometry),
						WasOneway:    way.Oneway,
					})
					if !way.Oneway {
						totalEdgesNum++
						notOnewayEdges++
						edges = append(edges, Edge{
							ID:           EdgeID(totalEdgesNum),
							WayID:        way.ID,
							SourceNodeID: wayNode.ID,
							TargetNodeID: source,
							CostMeters:   cost,
							Geom:         reverseLine(geometry),
							WasOneway:    false,
						})
					}
					source = wayNode.ID
					geometry = []GeoPoint{GeoPoint{Lon: node.node.Lon, Lat: node.node.Lat}}
				}
			}
		}
	}
	fmt.Printf("Done in %v\n\tEdges: (oneway = %d), (not oneway = %d) (total = %d)\n", time.Since(st), onewayEdges, notOnewayEdges, totalEdgesNum)

	fmt.Printf("Preparing nodes...")
	st = time.Now()
	nodesFiltered := []Node{}
	for _, node := range nodes {
		if node.useCount > 1 {
			nodesFiltered = append(nodesFiltered, node)
		}
	}
	fmt.Printf("Done in %v\n\tNodes: %d\n", time.Since(st), len(nodesFiltered))

	fmt.Printf("Applying edge expanding technique...")
	st = time.Now()
	cycles := 0
	expandedEdges := []ExpandedEdge{}
	expandedEdgesTotal := int64(0)
	for _, edge := range edges {
		edgeAsFromVertex := edge
		costMetersFromVertex := edgeAsFromVertex.CostMeters
		outcomingEdges := findOutComingEdges(edgeAsFromVertex, edges)
		for _, outcomingEdge := range outcomingEdges {
			edgeAsToVertex := edges[outcomingEdge-1] // We assuming that EdgeID == (SliceIndex + 1) which is equivalent to SliceIndex == (EdgeID - 1)
			// cycles, u-turn?
			// @TODO: some of those are deadend (or 'boundary') edges
			if edgeAsFromVertex.Geom[0] == edgeAsToVertex.Geom[len(edgeAsToVertex.Geom)-1] && edgeAsFromVertex.Geom[len(edgeAsFromVertex.Geom)-1] == edgeAsToVertex.Geom[0] {
				// fmt.Println(PrepareGeoJSONLinestring(edgeAsFromVertex.Geom))
				cycles++
				continue
			}
			costMetersToVertex := edgeAsToVertex.CostMeters
			expandedEdgesTotal++
			beforeFromIdx, fromMiddlePoint := findMiddlePoint(edgeAsFromVertex.Geom)
			fromGeomHalf := append([]GeoPoint{fromMiddlePoint}, edgeAsFromVertex.Geom[beforeFromIdx+1:len(edgeAsFromVertex.Geom)]...)
			beforeToIdx, toMiddlePoint := findMiddlePoint(edgeAsToVertex.Geom)
			toGeomHalf := append(make([]GeoPoint, 0, len(edgeAsToVertex.Geom[:beforeToIdx+1])+1), edgeAsToVertex.Geom[:beforeToIdx+1]...)
			toGeomHalf = append(toGeomHalf, toMiddlePoint)
			completedNewGeom := append(fromGeomHalf, toGeomHalf...)
			expandedEdges = append(expandedEdges, ExpandedEdge{
				ID:             expandedEdgesTotal,
				Source:         edgeAsFromVertex.ID,
				Target:         edgeAsToVertex.ID,
				SourceOSMWayID: edgeAsFromVertex.WayID,
				TargetOSMWayID: edgeAsToVertex.WayID,
				SourceComponent: expandedEdgeComponent{
					SourceNodeID: edgeAsFromVertex.SourceNodeID,
					TargetNodeID: edgeAsFromVertex.TargetNodeID,
				},
				TargeComponent: expandedEdgeComponent{
					SourceNodeID: edgeAsToVertex.SourceNodeID,
					TargetNodeID: edgeAsToVertex.TargetNodeID,
				},
				CostMeters: (costMetersFromVertex + costMetersToVertex) / 2.0,
				WasOneway:  edgeAsFromVertex.WasOneway,
				Geom:       completedNewGeom,
			})
		}
	}
	fmt.Printf("Done in %v\n", time.Since(st))
	fmt.Printf("\tIgnored cycles: %d\n", cycles)
	fmt.Printf("\tNumber of expanded edges: %d\n", expandedEdgesTotal)

	// @TODO: work with maneuvers (restrictions)
	fmt.Printf("Working with maneuvers (restrictions)...")
	st = time.Now()
	// Handling restrictions of "no" type
	for i, k := range restrictions {
		switch i {
		case "no_left_turn", "no_right_turn", "no_straight_on":
			// handle only way(from)-way(to)-node(via)
			for j, v := range k {
				if j.Type != "way" { // way(from)
					continue
				}
				fromOSMWayID := osm.WayID(j.ID)
				if _, ok := waysSeen[fromOSMWayID]; !ok {
					continue
				}
				for n := range v {
					if n.Type != "way" { // way(to)
						continue
					}
					if v[n].Type != "node" { // node(via)
						continue
					}
					toOSMWayID := osm.WayID(n.ID)
					if _, ok := waysSeen[toOSMWayID]; !ok {
						continue
					}
					// Delete restricted expanded edge
					{
						temp := expandedEdges[:0]
						for _, expEdge := range expandedEdges {
							if expEdge.SourceOSMWayID != fromOSMWayID || expEdge.TargetOSMWayID != toOSMWayID {
								temp = append(temp, expEdge)
							}
						}
						expandedEdges = temp
					}
				}
			}
			break
		default:
			// @TODO: need to think about U-turns: "no_u_turn"
			break
		}
	}
	// Handling restrictions of "only" type
	for i, k := range restrictions {
		switch i {
		case "only_left_turn", "only_right_turn", "only_straight_on":
			// handle only way(from)-way(to)-node(via)
			for j, v := range k {
				if j.Type != "way" { // way(from)
					continue
				}
				fromOSMWayID := osm.WayID(j.ID)
				if _, ok := waysSeen[fromOSMWayID]; !ok {
					continue
				}
				for n := range v {
					if n.Type != "way" { // way(to)
						continue
					}
					if v[n].Type != "node" { // node(via)
						continue
					}
					toOSMWayID := osm.WayID(n.ID)
					if _, ok := waysSeen[toOSMWayID]; !ok {
						continue
					}
					rvertexVia := v[n].ID
					{
						temp := expandedEdges[:0]
						for _, expEdge := range expandedEdges {
							if !(expEdge.SourceOSMWayID == fromOSMWayID && expEdge.TargetOSMWayID != toOSMWayID && expEdge.SourceComponent.TargetNodeID == osm.NodeID(rvertexVia)) {
								temp = append(temp, expEdge)
							}
						}
						expandedEdges = temp
					}
				}
			}
			break
		default:
			// @TODO: need to think about U-turns: "no_u_turn"
			break
		}

	}

	fmt.Printf("Done in %v\n", time.Since(st))
	fmt.Printf("\tUpdated of expanded edges: %d\n", len(expandedEdges))
	return expandedEdges, nil
}
