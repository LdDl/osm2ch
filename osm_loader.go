package osm2ch

import (
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"

	geojson "github.com/paulmach/go.geojson"
	"github.com/paulmach/osm"
	"github.com/pkg/errors"

	"github.com/paulmach/osm/osmpbf"
)

const (
	earthRadius = 6370.986884258304
	pi180       = math.Pi / 180.0
	pi180Rev    = 180.0 / math.Pi
)

// GeoPoint Representation of point on Earth
type GeoPoint struct {
	Lat float64
	Lon float64
}

// String Pretty printing for GeoPoint
func (gp GeoPoint) String() string {
	return fmt.Sprintf("Lon: %f | Lat: %f", gp.Lon, gp.Lat)
}

// edgeComponent Representation of edge (vertex_from -> vertex_to)
type edgeComponent struct {
	from int64
	to   int64
}

// wayComponent First and last edges of osm.Way
type wayComponent struct {
	FirstEdge edgeComponent
	LastEdge  edgeComponent
}

// restrictionComponent Representation of member of restriction relation. Could be way or node.
type restrictionComponent struct {
	ID   int64
	Type string
}

// expandedEdge New edge built on top of two adjacent edges
type expandedEdge struct {
	ID        int64
	Cost      float64
	Geom      []GeoPoint
	WasOneWay bool // Former OSM object was one way.
}

// ExpandedGraph Representation of edge expanded graph
/*
	map[newSourceVertexID]map[newTargetVertexID]newExpandedEdge
*/
type ExpandedGraph map[int64]map[int64]expandedEdge

type preExpandedGraph map[osm.NodeID]map[osm.NodeID]Edge

type EdgeID int64

type Edge struct {
	ID           EdgeID
	WayID        osm.WayID
	SourceNodeID osm.NodeID
	TargetNodeID osm.NodeID
	WasOneway    bool
	Geom         []GeoPoint
}

type ExpandedEdge struct {
	ID              int64
	Source          EdgeID
	Target          EdgeID
	SourceOSMWayID  osm.WayID
	TargetOSMWayID  osm.WayID
	SourceComponent expandedEdgeComponent
	TargeComponent  expandedEdgeComponent
	Geom            []GeoPoint
}

type expandedEdgeComponent struct {
	SourceNodeID osm.NodeID
	TargetNodeID osm.NodeID
}

// ImportFromOSMFile Imports graph from file of PBF-format (in OSM terms)
/*
	File should have PBF (Protocolbuffer Binary Format) extension according to https://github.com/paulmach/osm
*/
func ImportFromOSMFile(fileName string, cfg *OsmConfiguration) (ExpandedGraph, error) {
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

	// @todo: scan maneuvers (restrictions)
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
			_ = tag
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
					edges = append(edges, Edge{
						ID:           EdgeID(totalEdgesNum),
						WayID:        way.ID,
						SourceNodeID: source,
						TargetNodeID: wayNode.ID,
						Geom:         geometry,
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

	// @todo: expand
	fmt.Printf("Applying edge expanding technique...")
	st = time.Now()
	expandedGraph := make(ExpandedGraph)
	cycles := 0
	expandedEdges := []ExpandedEdge{}
	expandedEdgesTotal := int64(0)
	for _, edge := range edges {
		edgeAsFromVertex := edge
		outcomingEdges := findOutComingEdges(edgeAsFromVertex, edges)
		for _, outcomingEdge := range outcomingEdges {
			edgeAsToVertex := edges[outcomingEdge-1] // We assuming that EdgeID == (SliceIndex + 1) which is equivalent to SliceIndex == (EdgeID - 1)
			// cycles, u-turn?
			// @todo: some of those are deadend (or 'boundary') edges
			if edgeAsFromVertex.Geom[0] == edgeAsToVertex.Geom[len(edgeAsToVertex.Geom)-1] && edgeAsFromVertex.Geom[len(edgeAsFromVertex.Geom)-1] == edgeAsToVertex.Geom[0] {
				// fmt.Println(PrepareGeoJSONLinestring(edgeAsFromVertex.Geom))
				cycles++
				continue
			}
			expandedEdgesTotal++
			// @todo: remove debug print :D
			// fmt.Println()
			// fmt.Println(edgeAsFromVertex)
			// fmt.Println(edgeAsToVertex)
			// fmt.Println(PrepareGeoJSONLinestring(edgeAsFromVertex.Geom))
			// fmt.Println(PrepareGeoJSONLinestring(edgeAsToVertex.Geom))
			beforeFromIdx, fromMiddlePoint := findMiddlePoint(edgeAsFromVertex.Geom)
			fromGeomHalf := append([]GeoPoint{fromMiddlePoint}, edgeAsFromVertex.Geom[beforeFromIdx+1:len(edgeAsFromVertex.Geom)]...)
			beforeToIdx, toMiddlePoint := findMiddlePoint(edgeAsToVertex.Geom)
			// toGeomHalf := append(edgeAsToVertex.Geom[:beforeToIdx+1], toMiddlePoint) // that's big mistake due the nature of slicing
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
				Geom: completedNewGeom,
			})
			// fmt.Println(PrepareGeoJSONPoint(fromMiddlePoint))
			// fmt.Println(PrepareGeoJSONPoint(toMiddlePoint))
			// fmt.Println(PrepareGeoJSONLinestring(fromGeomHalf))
			// fmt.Println(PrepareGeoJSONLinestring(toGeomHalf))
			// fmt.Println(PrepareGeoJSONLinestring(completedNewGeom))
		}
	}
	fmt.Printf("Done in %v\n", time.Since(st))
	fmt.Printf("\tIgnored cycles: %d\n", cycles)
	fmt.Printf("\tNumber of expanded edges: %d\n", expandedEdgesTotal)

	// @todo: work with maneuvers (restrictions)
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
					// rvertexVia := v[n].ID
					// Delete restricted expanded edge
					{
						temp := expandedEdges[:0]
						for _, expEdge := range expandedEdges {
							// if expEdge.SourceOSMWayID == fromOSMWayID && expEdge.TargetOSMWayID == toOSMWayID {
							// 	fmt.Println(expEdge.ID)
							// }
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
			// @todo: need to think about U-turns: "no_u_turn"
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
					// if rvertexVia == 3832596114 {
					// fmt.Println()
					fmt.Printf("Restriction type: %s, Via: %d, From OSM Way: %d, To OSM Way: %d\n", i, rvertexVia, fromOSMWayID, toOSMWayID)
					// Save only allowed expanded edge and delete others
					{
						temp := expandedEdges[:0]
						for _, expEdge := range expandedEdges {
							if expEdge.SourceOSMWayID == fromOSMWayID && expEdge.TargetOSMWayID != toOSMWayID {
								if expEdge.SourceComponent.TargetNodeID == osm.NodeID(rvertexVia) {
									fmt.Printf("\t\tEdge to delete: %d, From OSM Way: %d, To OSM Way: %d, FromEdge: %d, ToEdge:%d, FromSourceComponent:%v ToSourceComponent:%v\n", expEdge.ID, fromOSMWayID, expEdge.TargetOSMWayID, expEdge.Source, expEdge.Target, expEdge.SourceComponent, expEdge.TargeComponent)
								} else {
									fmt.Printf("\t\tEdge NOT delete (no Via as target in source component): %d, From OSM Way: %d, To OSM Way: %d, FromEdge: %d, ToEdge:%d, FromSourceComponent:%v ToSourceComponent:%v\n", expEdge.ID, fromOSMWayID, expEdge.TargetOSMWayID, expEdge.Source, expEdge.Target, expEdge.SourceComponent, expEdge.TargeComponent)
								}
							}
							if !(expEdge.SourceOSMWayID == fromOSMWayID && expEdge.TargetOSMWayID != toOSMWayID && expEdge.SourceComponent.TargetNodeID == osm.NodeID(rvertexVia)) {
								temp = append(temp, expEdge)
							}
						}
						expandedEdges = temp
					}
					// }
				}
			}
			break
		default:
			// @todo: need to think about U-turns: "no_u_turn"
			break
		}

	}

	fmt.Printf("Done in %v\n", time.Since(st))
	fmt.Printf("\tUpdated of expanded edges: %d\n", len(expandedEdges))

	return expandedGraph, nil
}

// degreesToRadians deg = r * pi / 180
func degreesToRadians(d float64) float64 {
	return d * pi180
}

// radiansTodegrees r = deg  * 180 / pi
func radiansTodegrees(d float64) float64 {
	return d * pi180Rev
}

// greatCircleDistance Returns distance between two geo-points (kilometers)
func greatCircleDistance(p, q GeoPoint) float64 {
	lat1 := degreesToRadians(p.Lat)
	lon1 := degreesToRadians(p.Lon)
	lat2 := degreesToRadians(q.Lat)
	lon2 := degreesToRadians(q.Lon)
	diffLat := lat2 - lat1
	diffLon := lon2 - lon1
	a := math.Pow(math.Sin(diffLat/2), 2) + math.Cos(lat1)*math.Cos(lat2)*math.Pow(math.Sin(diffLon/2), 2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	ans := c * earthRadius
	return ans
}

// getShericalLength returns length for given line (kilometers)
func getShericalLength(line []GeoPoint) float64 {
	totalLength := 0.0
	if len(line) < 2 {
		return totalLength
	}
	for i := 1; i < len(line); i++ {
		totalLength += greatCircleDistance(line[i-1], line[i])
	}
	return totalLength
}
func middlePoint(p, q GeoPoint) GeoPoint {
	lat1 := degreesToRadians(p.Lat)
	lon1 := degreesToRadians(p.Lon)
	lat2 := degreesToRadians(q.Lat)
	lon2 := degreesToRadians(q.Lon)

	Bx := math.Cos(lat2) * math.Cos(lon2-lon1)
	By := math.Cos(lat2) * math.Sin(lon2-lon1)

	latMid := math.Atan2(math.Sin(lat1)+math.Sin(lat2), math.Sqrt((math.Cos(lat1)+Bx)*(math.Cos(lat1)+Bx)+By*By))
	lonMid := lon1 + math.Atan2(By, math.Cos(lat1)+Bx)
	return GeoPoint{Lat: radiansTodegrees(latMid), Lon: radiansTodegrees(lonMid)}
}

// findCentroid returns center point for given line (not middle point)
func findCentroid(line []GeoPoint) GeoPoint {
	totalPoints := len(line)
	if totalPoints == 1 {
		return line[0]
	}
	x, y, z := 0.0, 0.0, 0.0
	for i := 0; i < totalPoints; i++ {
		longitude := degreesToRadians(line[i].Lon)
		latitude := degreesToRadians(line[i].Lat)
		c1 := math.Cos(latitude)
		x += c1 * math.Cos(longitude)
		y += c1 * math.Sin(longitude)
		z += math.Sin(latitude)
	}

	x /= float64(totalPoints)
	y /= float64(totalPoints)
	z /= float64(totalPoints)

	centralLongitude := math.Atan2(y, x)
	centralSquareRoot := math.Sqrt(x*x + y*y)
	centralLatitude := math.Atan2(z, centralSquareRoot)

	return GeoPoint{
		Lon: radiansTodegrees(centralLongitude),
		Lat: radiansTodegrees(centralLatitude),
	}
}

// findDistance returns distance between two points (assuming they are Euclidean: Lon == X, Lat == Y)
func findDistance(p, q GeoPoint) float64 {
	xdistance := p.Lon - q.Lon
	ydistance := p.Lat - q.Lat
	return math.Sqrt(xdistance*xdistance + ydistance*ydistance)
}

// getLength returns length for given line  (assuming points of the line are Euclidean: Lon == X, Lat == Y)
func getLength(line []GeoPoint) float64 {
	totalLength := 0.0
	if len(line) < 2 {
		return totalLength
	}
	for i := 1; i < len(line); i++ {
		totalLength += findDistance(line[i-1], line[i])
	}
	return totalLength
}

// findMiddlePoint returns middle point for give line (not center point) and index of point in line right before middle one
// Purpose of returning index of point in line right before middle point is to give the ability to split line in a half
func findMiddlePoint(line []GeoPoint) (int, GeoPoint) {
	euclideanLength := getLength(line)
	halfDistance := euclideanLength / 2.0
	cl := 0.0
	ol := 0.0
	var result GeoPoint
	var idx int
	for i := 1; i < len(line); i++ {
		ol = cl
		tmpDist := findDistance(line[i-1], line[i])
		cl += tmpDist
		if halfDistance <= cl && halfDistance > ol {
			halfSub := halfDistance - ol
			result = pointOnSegmentByFraction(line[i-1], line[i], halfSub/tmpDist, halfSub)
			idx = i - 1
		}
	}
	return idx, result
}

// pointOnSegment returns a point on given segment using distance
func pointOnSegment(p, q GeoPoint, distance float64) GeoPoint {
	fraction := distance / findDistance(p, q)
	return GeoPoint{
		Lon: (1-fraction)*p.Lon + (fraction * q.Lon),
		Lat: (1-fraction)*p.Lat + (fraction * q.Lat),
	}
}

// pointOnSegmentByFraction returns a point on given segment using distance assuming knowledge about fraction
func pointOnSegmentByFraction(p, q GeoPoint, fraction, distance float64) GeoPoint {
	return GeoPoint{
		Lon: (1-fraction)*p.Lon + (fraction * q.Lon),
		Lat: (1-fraction)*p.Lat + (fraction * q.Lat),
	}
}

func reverseLine(pts []GeoPoint) []GeoPoint {
	inputLen := len(pts)
	output := make([]GeoPoint, inputLen)
	for i, n := range pts {
		j := inputLen - i - 1
		output[j] = n
	}
	return output
}

func reverseLineInPlace(pts []GeoPoint) {
	inputLen := len(pts)
	inputMid := inputLen / 2
	for i := 0; i < inputMid; i++ {
		j := inputLen - i - 1
		pts[i], pts[j] = pts[j], pts[i]
	}
}

func findEdgesBySource(edges []Edge, sourceID osm.NodeID) []EdgeID {
	result := []EdgeID{}
	for _, edge := range edges {
		if edge.SourceNodeID == sourceID {
			result = append(result, edge.ID)
		}
	}
	return result
}

func findOutComingEdges(givenEdge Edge, edges []Edge) []EdgeID {
	result := []EdgeID{}
	for _, edge := range edges {
		if edge.SourceNodeID == givenEdge.TargetNodeID && edge.ID != givenEdge.ID {
			result = append(result, edge.ID)
		}
	}
	return result
}

// PrepareWKTLinestring Creates WKT LineString from set of points
func PrepareWKTLinestring(pts []GeoPoint) string {
	ptsStr := make([]string, len(pts))
	for i := range pts {
		ptsStr[i] = fmt.Sprintf("%f %f", pts[i].Lon, pts[i].Lat)
	}
	return fmt.Sprintf("LINESTRING(%s)", strings.Join(ptsStr, ","))
}

// PrepareGeoJSONLinestring Creates GeoJSON LineString from set of points
func PrepareGeoJSONLinestring(pts []GeoPoint) string {
	pts2d := make([][]float64, len(pts))
	for i := range pts {
		pts2d[i] = []float64{pts[i].Lon, pts[i].Lat}
	}
	b, err := geojson.NewLineStringGeometry(pts2d).MarshalJSON()
	if err != nil {
		fmt.Printf("Warning. Can not convert geometry to geojson format: %s", err.Error())
		return ""
	}
	return string(b)
}

// PrepareWKTPoint Creates WKT Point from given points
func PrepareWKTPoint(pt GeoPoint) string {
	return fmt.Sprintf("POINT(%f %f)", pt.Lon, pt.Lat)
}

// PrepareGeoJSONPoint Creates GeoJSON Point from given point
func PrepareGeoJSONPoint(pt GeoPoint) string {
	b, err := geojson.NewPointGeometry([]float64{pt.Lon, pt.Lat}).MarshalJSON()
	if err != nil {
		fmt.Printf("Warning. Can not convert geometry to geojson format: %s", err.Error())
		return ""
	}
	return string(b)
}
