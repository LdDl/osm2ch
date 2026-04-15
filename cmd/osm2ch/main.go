package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/LdDl/ch"
	"github.com/LdDl/go-gmns/gmns/types"
	"github.com/LdDl/osm2ch"
)

var (
	osmFileName = flag.String("file", "my_graph.osm.pbf", "Filename of *.osm.pbf file (it has to be compressed)")
	out         = flag.String("out", "my_graph.csv", "Filename of 'Comma-Separated Values' (CSV) formatted file. E.g.: if file name is 'map.csv' then 3 files will be produced: 'map.csv' (edges), 'map_vertices.csv', 'map_shortcuts.csv'")
	geomFormat  = flag.String("geomf", "wkt", "Format of output geometry. Expected values: wkt / geojson")
	agentsStr   = flag.String("agents", "auto", "Comma-separated list of agent types. Expected values: auto, bike, walk")
)

func main() {
	flag.Parse()

	agentTypes := parseAgentTypes(*agentsStr)

	edgeExpandedGraph, err := osm2ch.ImportFromGMNS(*osmFileName, agentTypes)
	if err != nil {
		fmt.Println(err)
		return
	}

	fnamePart := strings.Split(*out, ".csv")
	fnameEdges := fnamePart[0] + ".csv"
	fnameVertices := fnamePart[0] + "_vertices.csv"
	fnameShortcuts := fnamePart[0] + "_shortcuts.csv"

	/* Edges file */
	fileEdges, err := os.Create(fnameEdges)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fileEdges.Close()
	writerEdges := csv.NewWriter(fileEdges)
	defer writerEdges.Flush()
	writerEdges.Comma = ';'
	err = writerEdges.Write([]string{
		"from_vertex_id", "to_vertex_id", "length_meters", "free_speed", "geom",
		"was_one_way", "edge_id",
		"osm_way_from", "osm_way_to",
		"osm_way_from_source_node", "osm_way_from_target_node",
		"osm_way_to_source_node", "osm_way_to_target_node",
		"macro_node_from_source", "macro_node_from_target",
		"macro_node_to_source", "macro_node_to_target",
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	/* Vertices file */
	fileVertices, err := os.Create(fnameVertices)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fileVertices.Close()
	writerVertices := csv.NewWriter(fileVertices)
	defer writerVertices.Flush()
	writerVertices.Comma = ';'
	err = writerVertices.Write([]string{"vertex_id", "order_pos", "importance", "geom"})
	if err != nil {
		fmt.Println(err)
		return
	}

	verticesGeoms := make(map[int64]osm2ch.GeoPoint)
	graph := ch.Graph{}

	for _, edge := range edgeExpandedGraph {
		source := int64(edge.Source)
		target := int64(edge.Target)
		err := graph.CreateVertex(source)
		if err != nil {
			fmt.Printf("Can not create source vertex: %v\n", err)
			return
		}
		err = graph.CreateVertex(target)
		if err != nil {
			fmt.Printf("Can not create target vertex: %v\n", err)
			return
		}
		// CH uses length_meters as weight for now
		err = graph.AddEdge(source, target, edge.LengthMeters)
		if err != nil {
			fmt.Printf("Can not add edge: %v\n", err)
			return
		}
		if len(edge.Geom) < 2 {
			continue
		}

		geomStr := ""
		if strings.ToLower(*geomFormat) == "geojson" {
			geomStr = osm2ch.PrepareGeoJSONLinestring(edge.Geom)
		} else {
			geomStr = osm2ch.PrepareWKTLinestring(edge.Geom)
		}

		if _, ok := verticesGeoms[source]; !ok {
			verticesGeoms[source] = osm2ch.GeoPoint{Lon: edge.Geom[0].Lon, Lat: edge.Geom[0].Lat}
		}
		if _, ok := verticesGeoms[target]; !ok {
			verticesGeoms[target] = osm2ch.GeoPoint{Lon: edge.Geom[len(edge.Geom)-1].Lon, Lat: edge.Geom[len(edge.Geom)-1].Lat}
		}

		err = writerEdges.Write([]string{
			fmt.Sprintf("%d", source),
			fmt.Sprintf("%d", target),
			fmt.Sprintf("%f", edge.LengthMeters),
			fmt.Sprintf("%f", edge.FreeSpeed),
			geomStr,
			fmt.Sprintf("%t", edge.WasOneway),
			fmt.Sprintf("%d", edge.ID),
			fmt.Sprintf("%d", edge.SourceOSMWayID),
			fmt.Sprintf("%d", edge.TargetOSMWayID),
			fmt.Sprintf("%d", edge.SourceComponent.SourceOSMNodeID),
			fmt.Sprintf("%d", edge.SourceComponent.TargetOSMNodeID),
			fmt.Sprintf("%d", edge.TargeComponent.SourceOSMNodeID),
			fmt.Sprintf("%d", edge.TargeComponent.TargetOSMNodeID),
			fmt.Sprintf("%d", edge.SourceComponent.SourceNodeID),
			fmt.Sprintf("%d", edge.SourceComponent.TargetNodeID),
			fmt.Sprintf("%d", edge.TargeComponent.SourceNodeID),
			fmt.Sprintf("%d", edge.TargeComponent.TargetNodeID),
		})
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	fmt.Println("Starting contraction process....")
	st := time.Now()
	graph.PrepareContractionHierarchies()
	fmt.Printf("Done contraction process in %v\n", time.Since(st))

	/* Write vertices */
	vertices := graph.Vertices
	for i := 0; i < len(vertices); i++ {
		currentVertexExternal := vertices[i].Label
		vertexGeom := verticesGeoms[currentVertexExternal]
		geomStr := ""
		if strings.ToLower(*geomFormat) == "geojson" {
			geomStr = osm2ch.PrepareGeoJSONPoint(vertexGeom)
		} else {
			geomStr = osm2ch.PrepareWKTPoint(vertexGeom)
		}
		err = writerVertices.Write([]string{
			fmt.Sprintf("%d", currentVertexExternal),
			fmt.Sprintf("%d", graph.Vertices[i].OrderPos()),
			fmt.Sprintf("%d", graph.Vertices[i].Importance()),
			geomStr,
		})
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	/* Write shortcuts */
	err = graph.ExportShortcutsToFile(fnameShortcuts)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func parseAgentTypes(s string) []types.AgentType {
	parts := strings.Split(s, ",")
	result := make([]types.AgentType, 0, len(parts))
	for _, p := range parts {
		switch strings.TrimSpace(strings.ToLower(p)) {
		case "auto":
			result = append(result, types.AGENT_AUTO)
		case "bike":
			result = append(result, types.AGENT_BIKE)
		case "walk":
			result = append(result, types.AGENT_WALK)
		default:
			fmt.Printf("Warning: unknown agent type '%s', skipping\n", p)
		}
	}
	if len(result) == 0 {
		return []types.AgentType{types.AGENT_AUTO}
	}
	return result
}
