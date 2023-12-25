package osm2ch

import (
	"sort"
)

const (
	defaultRightMostLanes = 1
	defaultLeftMostLanes  = 1
)

type connectionPair struct {
	first  int
	second int
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getIntersectionsConnections(incomingLink *NetworkLink, outcomingLinks []*NetworkLink) [][]connectionPair {

	// Sort outcoming links by angle in descending order (left to right)
	angles := make([]float64, len(outcomingLinks))
	for i, outLink := range outcomingLinks {
		angles[i] = angleBetweenLines(incomingLink.geomEuclidean, outLink.geomEuclidean)
	}
	indicesMap := make(map[NetworkLinkID]int, len(outcomingLinks))
	for index, link := range outcomingLinks {
		indicesMap[link.ID] = index
	}
	indices := make([]int, len(outcomingLinks))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return angles[indices[i]] > angles[indices[j]]
	})
	outcomingLinksSorted := make([]*NetworkLink, len(outcomingLinks))
	for i := range outcomingLinksSorted {
		outcomingLinksSorted[i] = outcomingLinks[indices[i]]
	}

	// Evaluate lanes connections
	connections := make([][]connectionPair, len(outcomingLinksSorted))
	outcomingLanes := incomingLink.GetOutcomingLanes()
	if outcomingLanes == 1 {
		leftLink := outcomingLinksSorted[0]
		connections[indicesMap[leftLink.ID]] = []connectionPair{{0, 0}, {0, 0}}
		for _, link := range outcomingLinksSorted[1:] {
			connections[indicesMap[link.ID]] = []connectionPair{{0, 0}, {link.GetIncomingLanes() - 1, link.GetIncomingLanes() - 1}}
		}
		// fmt.Println("\t\t\t", connections)
		return connections
	}
	if len(outcomingLinksSorted) == 1 { // Full connection
		link := outcomingLinksSorted[0]
		minConnections := min(outcomingLanes, link.GetIncomingLanes())
		connections[indicesMap[link.ID]] = []connectionPair{{0, minConnections - 1}, {0, minConnections - 1}}
	} else if len(outcomingLinksSorted) == 2 { // Default right, remaining left
		leftLink := outcomingLinksSorted[0]
		minConnections := min(outcomingLanes-defaultLeftMostLanes, leftLink.GetIncomingLanes()) // If link has incoming lanes
		connections[indicesMap[leftLink.ID]] = []connectionPair{{0, minConnections - 1}, {0, minConnections - 1}}
		rightLink := outcomingLinksSorted[len(outcomingLinksSorted)-1]
		connections[indicesMap[rightLink.ID]] = []connectionPair{{outcomingLanes - defaultRightMostLanes, outcomingLanes - 1}, {rightLink.GetIncomingLanes() - defaultRightMostLanes, rightLink.GetIncomingLanes() - 1}}
	} else { // >= 3, default left, default right, remaining middle
		leftLink := outcomingLinksSorted[0]
		connections[indicesMap[leftLink.ID]] = []connectionPair{{0, defaultLeftMostLanes - 1}, {0, defaultLeftMostLanes - 1}}

		middleLinks := outcomingLinksSorted[1 : len(outcomingLinksSorted)-1]
		assignedToMiddle := make([]int, len(middleLinks))
		middleLinksLanes := make([]int, len(middleLinks))
		for i, midLink := range middleLinks {
			middleLinksLanes[i] = midLink.GetIncomingLanes()
		}
		leftLanesNum := outcomingLanes - defaultLeftMostLanes - defaultRightMostLanes
		if leftLanesNum >= len(middleLinks) {
			startLaneNumber := defaultLeftMostLanes
			for leftLanesNum > 0 && total(middleLinksLanes) > 0 {
				for idx := range middleLinks {
					if middleLinksLanes[idx] == 0 {
						continue
					}
					if leftLanesNum == 0 {
						continue
					}
					middleLinksLanes[idx]--
					assignedToMiddle[idx]++
					leftLanesNum--
				}
			}
			for idx, middleLink := range middleLinks {
				connections[indicesMap[middleLink.ID]] = []connectionPair{{startLaneNumber, startLaneNumber + assignedToMiddle[idx] - 1}, {middleLink.GetIncomingLanes() - assignedToMiddle[idx], middleLink.GetIncomingLanes() - 1}}
				startLaneNumber += assignedToMiddle[idx]
			}
		} else if outcomingLanes < len(middleLinks) {
			laneNumber := -1
			linkIndex := -1
			for laneNumber = 0; laneNumber < outcomingLanes-1; laneNumber++ {
				linkIndex = laneNumber
				middleLink := middleLinks[linkIndex]
				connections[indicesMap[middleLink.ID]] = []connectionPair{{laneNumber, laneNumber}, {middleLink.GetIncomingLanes() - 1, middleLink.GetIncomingLanes() - 1}}
			}
			laneNumber++
			startLinkIndex := linkIndex + 1
			for linkIndex = startLinkIndex; linkIndex < len(middleLinks); linkIndex++ {
				middleLink := middleLinks[linkIndex]
				connections[indicesMap[middleLink.ID]] = []connectionPair{{laneNumber, laneNumber}, {middleLink.GetIncomingLanes() - 1, middleLink.GetIncomingLanes() - 1}}
			}
		} else {
			startLaneNumber := 0
			if outcomingLanes-defaultLeftMostLanes == len(middleLinks) {
				startLaneNumber = defaultLeftMostLanes
			} else {
				startLaneNumber = 0
			}
			for _, midLink := range middleLinks {
				connections[indicesMap[midLink.ID]] = []connectionPair{{startLaneNumber, startLaneNumber}, {midLink.GetIncomingLanes() - 1, midLink.GetIncomingLanes() - 1}}
				startLaneNumber++
			}
		}
		rightLink := outcomingLinksSorted[len(outcomingLinksSorted)-1]
		connections[indicesMap[rightLink.ID]] = []connectionPair{{outcomingLanes - defaultRightMostLanes, outcomingLanes - 1}, {rightLink.GetIncomingLanes() - defaultRightMostLanes, rightLink.GetIncomingLanes() - 1}}
	}

	return connections
}

func total(slice []int) int {
	sum := 0
	for _, val := range slice {
		sum += val
	}
	return sum
}
