package osm2ch

import (
	"fmt"
	"math"
	"sort"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/osm"
)

/* Links stuff */
type NetworkLinkID int

type NetworkLink struct {
	name               string
	geom               orb.LineString
	geomEuclidean      orb.LineString
	lengthMeters       float64
	freeSpeed          float64
	maxSpeed           float64
	capacity           int
	ID                 NetworkLinkID
	osmWayID           osm.WayID
	linkClass          LinkClass
	linkType           LinkType
	linkConnectionType LinkConnectionType
	controlType        ControlType
	allowedAgentTypes  []AgentType
	sourceNodeID       NetworkNodeID
	targetNodeID       NetworkNodeID

	sourceOsmNodeID osm.NodeID
	targetOsmNodeID osm.NodeID

	wasBidirectional bool

	lanesNew int
	/* For Mesoscopic and Microscopic */
	mesolinks              []NetworkLinkID
	lanesList              []int
	lanesListCut           []int
	lanesChangePoints      []float64
	lanesChange            [][2]int
	lanesChangeCut         [][2]int
	geomOffset             orb.LineString
	geomOffsetCut          []orb.LineString
	geomEuclideanOffset    orb.LineString
	geomEuclideanOffsetCut []orb.LineString
	lengthMetersOffset     float64

	downstreamShortCut bool
	upstreamShortCut   bool

	downstreamIsTarget bool
	upstreamIsTarget   bool

	upstreamCutLen   float64
	downstreamCutLen float64
}

type DirectionType uint16

const (
	DIRECTION_FORWARD = DirectionType(iota + 1)
	DIRECTION_BACKWARD
)

func networkLinkFromOSM(id NetworkLinkID, sourceNodeID, targetNodeID NetworkNodeID, sourceOsmNodeID, targetOsmNodeID osm.NodeID, direction DirectionType, wayOSM *WayData, segmentNodes []*Node) *NetworkLink {
	freeSpeed := -1.0
	maxSpeed := -1.0
	capacity := -1

	if wayOSM.capacity < 0 {
		if defaultCap, ok := defaultCapacityByLinkType[wayOSM.linkType]; ok {
			capacity = defaultCap
		}
	}
	if wayOSM.freeSpeed < 0 {
		if wayOSM.maxSpeed >= 0 {
			freeSpeed = wayOSM.maxSpeed
		} else {
			if defaultSpeed, ok := defaultSpeedByLinkType[wayOSM.linkType]; ok {
				freeSpeed = defaultSpeed
				maxSpeed = defaultSpeed
			}
		}
	}

	link := NetworkLink{
		name:               wayOSM.name,
		lanesList:          make([]int, 0),
		freeSpeed:          freeSpeed,
		maxSpeed:           maxSpeed,
		capacity:           capacity,
		ID:                 id,
		osmWayID:           wayOSM.ID,
		linkClass:          wayOSM.linkClass,
		linkType:           wayOSM.linkType,
		linkConnectionType: wayOSM.linkConnectionType,
		sourceNodeID:       sourceNodeID,
		targetNodeID:       targetNodeID,
		sourceOsmNodeID:    sourceOsmNodeID,
		targetOsmNodeID:    targetOsmNodeID,
		controlType:        NOT_SIGNAL,
		allowedAgentTypes:  make([]AgentType, len(wayOSM.allowedAgentTypes)),
	}
	copy(link.allowedAgentTypes, wayOSM.allowedAgentTypes)

	if !wayOSM.Oneway {
		link.wasBidirectional = true
	}
	if wayOSM.Oneway {
		link.lanesNew = wayOSM.lanes
	} else {
		switch direction {
		case DIRECTION_FORWARD:
			if wayOSM.lanesForward > 0 {
				link.lanesNew = wayOSM.lanesForward
			} else if wayOSM.lanes > 0 {
				link.lanesNew = int(math.Ceil(float64(wayOSM.lanes) / 2.0))
			} else {
				link.lanesNew = wayOSM.lanes
			}
		case DIRECTION_BACKWARD:
			if wayOSM.lanesBackward >= 0 {
				link.lanesNew = wayOSM.lanesBackward
			} else if wayOSM.lanes >= 0 {
				link.lanesNew = int(math.Ceil(float64(wayOSM.lanes) / 2.0))
			} else {
				link.lanesNew = wayOSM.lanes
			}
		default:
			panic("Should not happen!")
		}
	}
	if link.lanesNew <= 0 {
		link.lanesNew = defaultLanesByLinkType[link.linkType]
	}

	// Walk all segment nodes except the first and the last one to detect links under traffic light control
	for i := 1; i < len(segmentNodes)-1; i++ {
		node := segmentNodes[i]
		if node.controlType == IS_SIGNAL {
			link.controlType = IS_SIGNAL
		}
	}

	// Prepare geometry
	link.geom = make(orb.LineString, 0, len(segmentNodes))
	switch direction {
	case DIRECTION_FORWARD:
		for _, node := range segmentNodes {
			pt := orb.Point{node.node.Lon, node.node.Lat}
			link.geom = append(link.geom, pt)
		}
	case DIRECTION_BACKWARD:
		for i := len(segmentNodes) - 1; i >= 0; i-- {
			node := segmentNodes[i]
			pt := orb.Point{node.node.Lon, node.node.Lat}
			link.geom = append(link.geom, pt)
		}
	default:
		panic("Should not happen!")
	}
	link.lengthMeters = geo.LengthHaversign(link.geom)
	return &link
}

func (link *NetworkLink) prepareLanes() {
	link.lanesList = make([]int, 0)
	link.lanesChange = make([][2]int, 0)
	link.lanesChangePoints = make([]float64, 0)

	lanesChangePointsTemp := []float64{0.0, link.lengthMeters}
	if link.lengthMeters < resolution {
		link.lanesChangePoints = []float64{0.0, link.lengthMeters}
	} else {
		for len(lanesChangePointsTemp) != 0 {
			target := lanesChangePointsTemp[0]
			remove := make(map[int]struct{})
			for idx, point := range lanesChangePointsTemp {
				if target-resolution <= point && point <= target+resolution {
					remove[idx] = struct{}{}
				}
			}
			link.lanesChangePoints = append(link.lanesChangePoints, target)
			for idx := range remove {
				lanesChangePointsTemp = append(lanesChangePointsTemp[:idx], lanesChangePointsTemp[idx+1:]...)
			}
		}
		sort.Float64s(link.lanesChangePoints)
	}

	for i := 0; i < len(link.lanesChangePoints)-1; i++ {
		link.lanesList = append(link.lanesList, link.lanesNew)
		link.lanesChange = append(link.lanesChange, [2]int{0.0, 0.0})
	}
}

func (link *NetworkLink) GetIncomingLanes() int {
	return link.lanesList[0]
}

func (link *NetworkLink) GetOutcomingLanes() int {
	idx := len(link.lanesList) - 1
	if idx < 0 {
		fmt.Printf("[WARNING]: Macroscopic link %d has no outcoming lanes", link.ID)
		return -1
	}
	return link.lanesList[idx]
}

func (link *NetworkLink) GetIncomingLaneIndices() []int {
	return link._laneIndices(link.lanesChange[0][0], link.lanesChange[0][1])
}

func (link *NetworkLink) GetOutcomingLaneIndices() []int {
	idx := len(link.lanesChange) - 1
	if idx < 0 {
		fmt.Printf("[WARNING]: Macroscopic link %d has no lanes change", link.ID)
		return make([]int, 0)
	}
	return link._laneIndices(link.lanesChange[idx][0], link.lanesChange[idx][1])
}

func (link *NetworkLink) _laneIndices(lanesChangeLeft int, lanesChangeRight int) []int {
	return laneIndices(link.lanesNew, lanesChangeLeft, lanesChangeRight)
}

func laneIndices(lanes int, lanesChangeLeft int, lanesChangeRight int) []int {
	laneIndices := make([]int, lanes)
	for i := 1; i <= lanes; i++ {
		laneIndices[i-1] = i
	}
	if lanesChangeLeft < 0 {
		laneIndices = laneIndices[-lanesChangeLeft:]
	} else if lanesChangeLeft > 0 {
		left := make([]int, lanesChangeLeft)
		for i := range left {
			left[i] = -lanesChangeLeft + i
		}
		laneIndices = append(left, laneIndices...)
	}
	if lanesChangeRight < 0 {
		laneIndices = laneIndices[:lanes+lanesChangeRight]
	} else if lanesChangeRight > 0 {
		right := make([]int, lanesChangeRight)
		for i := range right {
			right[i] = lanes + 1 + i
		}
		laneIndices = append(laneIndices, right...)
	}
	return laneIndices
}

func (link *NetworkLink) MaxLanes() int {
	if len(link.lanesList) == 0 {
		return -1
	}
	max := link.lanesList[0]
	for _, lane := range link.lanesList {
		if lane > max {
			max = lane
		}
	}
	return max
}

// Prepares cut length for link
func (link *NetworkLink) calcCutLen() {
	// Dodge potential change of number of lanes on two ends of the macroscopic link
	upstreamMaxCut := math.Max(shortcutLen, link.lanesChangePoints[1]-link.lanesChangePoints[0]-3)
	// Defife a variable downstreamMaxCut which is the maximum length of a cut that can be made downstream of the link,
	// calculated as the maximum of the shortcutLen and the difference between the last two elements in the link.lanesChangePoints minus 3.
	downstreamMaxCut := math.Max(shortcutLen, link.lanesChangePoints[len(link.lanesChangePoints)-1]-link.lanesChangePoints[len(link.lanesChangePoints)-2]-3)
	if link.upstreamShortCut && link.downstreamShortCut {
		totalLengthCut := 2 * shortcutLen * cutLenMin
		if link.lengthMetersOffset > totalLengthCut {
			link.upstreamCutLen = shortcutLen
			link.downstreamCutLen = shortcutLen
		} else {
			link.upstreamCutLen = (link.lengthMetersOffset / totalLengthCut) * shortcutLen
			link.downstreamCutLen = link.upstreamCutLen
		}
	} else if link.upstreamShortCut {
		cutIdx := 0
		cutPlaceFound := false
		for i := link.lanesList[len(link.lanesList)-1]; i >= 0; i-- {
			if link.lengthMetersOffset > math.Min(downstreamMaxCut, cutLen[i])+shortcutLen+cutLenMin {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			link.upstreamCutLen = shortcutLen
			link.downstreamCutLen = math.Min(downstreamMaxCut, cutLen[cutIdx])
		} else {
			downStreamCut := math.Min(downstreamMaxCut, cutLen[0])
			totalLen := downStreamCut + shortcutLen + cutLenMin
			link.upstreamCutLen = (link.lengthMetersOffset / totalLen) * shortcutLen
			link.downstreamCutLen = (link.lengthMetersOffset / totalLen) * downStreamCut
		}
	} else if link.downstreamShortCut {
		cutIdx := 0
		cutPlaceFound := false
		for i := link.lanesList[len(link.lanesList)-1]; i >= 0; i-- {
			if link.lengthMetersOffset > math.Min(upstreamMaxCut, cutLen[i])+shortcutLen+cutLenMin {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			link.upstreamCutLen = math.Min(upstreamMaxCut, cutLen[cutIdx])
			link.downstreamCutLen = shortcutLen
		} else {
			upStreamCut := math.Min(upstreamMaxCut, cutLen[0])
			totalLen := upStreamCut + shortcutLen + cutLenMin
			link.upstreamCutLen = (link.lengthMetersOffset / totalLen) * cutLen[0]
			link.downstreamCutLen = (link.lengthMetersOffset / totalLen) * shortcutLen
		}
	} else {
		cutIdx := 0
		cutPlaceFound := false
		for i := link.lanesList[len(link.lanesList)-1]; i >= 0; i-- {
			if link.lengthMetersOffset > math.Min(upstreamMaxCut, cutLen[i])+math.Min(downstreamMaxCut, cutLen[i])+cutLenMin {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			link.upstreamCutLen = math.Min(upstreamMaxCut, cutLen[cutIdx])
			link.downstreamCutLen = math.Min(downstreamMaxCut, cutLen[cutIdx])
		} else {
			upStreamCut := math.Min(upstreamMaxCut, cutLen[0])
			downStreamCut := math.Min(downstreamMaxCut, cutLen[0])
			totalLen := downStreamCut + upStreamCut + cutLenMin
			link.upstreamCutLen = (link.lengthMetersOffset / totalLen) * upStreamCut
			link.downstreamCutLen = (link.lengthMetersOffset / totalLen) * downStreamCut
		}
	}
}

// Cuts redudant geometry
func (link *NetworkLink) performCut() {

	// Create copy for those since we will do mutations and want to keep original data
	lanesChangePoints := make([]float64, len(link.lanesChangePoints))
	copy(lanesChangePoints, link.lanesChangePoints)
	link.lanesListCut = make([]int, len(link.lanesList))
	copy(link.lanesListCut, link.lanesList)
	link.lanesChangeCut = make([][2]int, len(link.lanesChange))
	copy(link.lanesChangeCut, link.lanesChange)

	lanesChangePoints[0] = link.upstreamCutLen
	lanesChangePoints[len(lanesChangePoints)-1] = link.lengthMetersOffset - link.downstreamCutLen
	// breakIdx := 1
	// for breakIdx = 1; breakIdx < len(lanesChangePoints); breakIdx++ {
	// 	if lanesChangePoints[breakIdx] > link.upstreamCutLen {
	// 		break
	// 	}
	// }
	// lanesChangePoints = append(lanesChangePoints[breakIdx:])
	// lanesChangePoints = append([]float64{link.upstreamCutLen}, lanesChangePoints...)
	// link.lanesListCut = link.lanesListCut[breakIdx-1:]
	// link.lanesChange = link.lanesChange[breakIdx-1:]

	// breakIdx = len(lanesChangePoints) - 2
	// for breakIdx := len(lanesChangePoints) - 2; breakIdx >= 0; breakIdx-- {
	// 	if link.lengthMetersOffset-lanesChangePoints[breakIdx] > link.downstreamCutLen {
	// 		break
	// 	}
	// }
	// lanesChangePoints = lanesChangePoints[:breakIdx+1]
	// lanesChangePoints = append(lanesChangePoints, link.lengthMetersOffset-link.downstreamCutLen)
	// link.lanesListCut = link.lanesListCut[:breakIdx+1]
	// link.lanesChange = link.lanesChange[:breakIdx+1]

	for i := range link.lanesListCut {
		start := lanesChangePoints[i]
		end := lanesChangePoints[i+1]
		geomCut := SubstringHaversine(link.geomOffset, start, end)
		geomEuclideanCut := lineToEuclidean(geomCut)
		link.geomOffsetCut = append(link.geomOffsetCut, geomCut)
		link.geomEuclideanOffsetCut = append(link.geomEuclideanOffsetCut, geomEuclideanCut)
	}
}

func linksToSlice(links map[NetworkLinkID]*NetworkLink) []NetworkLinkID {
	ans := make([]NetworkLinkID, 0, len(links))
	for k := range links {
		ans = append(ans, k)
	}
	return ans
}
