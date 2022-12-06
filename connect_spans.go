package osm2ch

import (
	"sort"
)

func getSpansConnections(outcomingLink *NetworkLink, incomingLinksList []*NetworkLink) [][]connectionPair {

	// Sort outcoming links by angle in descending order (left to right)
	angles := make([]float64, len(incomingLinksList))
	for i, inLink := range incomingLinksList {
		angles[i] = angleBetweenLines(inLink.geomEuclidean, outcomingLink.geomEuclidean)
	}
	indicesMap := make(map[NetworkLinkID]int, len(incomingLinksList))
	for index, link := range incomingLinksList {
		indicesMap[link.ID] = index
	}
	indices := make([]int, len(incomingLinksList))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return angles[indices[i]] > angles[indices[j]]
	})
	incomingLinksSorted := make([]*NetworkLink, len(incomingLinksList))
	for i := range incomingLinksSorted {
		incomingLinksSorted[i] = incomingLinksList[indices[i]]
	}
	// Evaluate lanes connections
	connections := make([][]connectionPair, len(incomingLinksSorted))
	outcomingLanes := outcomingLink.lanesList[len(outcomingLink.lanesList)-1]
	leftLink := incomingLinksSorted[0]
	leftLinkOutcomingLanes := leftLink.lanesList[len(leftLink.lanesList)-1]
	minConnections := min(outcomingLanes, leftLinkOutcomingLanes)
	// In <-> Out
	connections[indicesMap[leftLink.ID]] = []connectionPair{{leftLinkOutcomingLanes - minConnections, leftLinkOutcomingLanes - 1}, {0, minConnections - 1}}
	for _, inLink := range incomingLinksSorted[1:] {
		inLinkOutcomingLanes := inLink.lanesList[len(inLink.lanesList)-1]
		minConnections := min(outcomingLanes, inLinkOutcomingLanes)
		// In <-> Out
		connections[indicesMap[inLink.ID]] = []connectionPair{{0, minConnections - 1}, {outcomingLanes - minConnections, outcomingLanes - 1}}
	}
	return connections
}
