package osm2ch

import (
	"math"

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
	lanesList          []int
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

	/* Mesoscopic */
	breakpoints            []float64
	lanesListCut           []int
	lanesChange            [][]int
	lanesChangeCut         [][]int
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
		lanesList:          make([]int, 1),
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
		link.lanesList[0] = wayOSM.lanes
	} else {
		switch direction {
		case DIRECTION_FORWARD:
			if wayOSM.lanesForward > 0 {
				link.lanesList[0] = wayOSM.lanesForward
			} else if wayOSM.lanes > 0 {
				link.lanesList[0] = int(math.Ceil(float64(wayOSM.lanes) / 2.0))
			} else {
				link.lanesList[0] = wayOSM.lanes
			}
		case DIRECTION_BACKWARD:
			if wayOSM.lanesBackward >= 0 {
				link.lanesList[0] = wayOSM.lanesBackward
			} else if wayOSM.lanes >= 0 {
				link.lanesList[0] = int(math.Ceil(float64(wayOSM.lanes) / 2.0))
			} else {
				link.lanesList[0] = wayOSM.lanes
			}
		default:
			panic("Should not happen!")
		}
	}
	if link.lanesList[0] <= 0 {
		link.lanesList[0] = defaultLanesByLinkType[link.linkType]
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

func (link *NetworkLink) GetLanes() int {
	return link.lanesList[0]
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
	// Defife a variable downstream_max_cut which is the maximum length of a cut that can be made downstream of the link,
	// calculated as the maximum of the _length_of_short_cut and the difference between the last two elements in the link.lanes_change_point_list minus 3.
	downStreamMaxCut := math.Max(shortcutLen, link.breakpoints[len(link.breakpoints)-1]-link.breakpoints[len(link.breakpoints)-2]-3)
	if link.upstreamShortCut && link.downstreamShortCut {
		totalLengthCut := 2 * shortcutLen * cutLenMin
		_ = totalLengthCut
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
			if link.lengthMetersOffset > math.Min(downStreamMaxCut, cutLen[i])+shortcutLen+cutLenMin {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			link.upstreamCutLen = shortcutLen
			link.downstreamCutLen = math.Min(downStreamMaxCut, cutLen[cutIdx])
		} else {
			downStreamCut := math.Min(downStreamMaxCut, cutLen[0])
			totalLen := downStreamCut + shortcutLen + cutLenMin
			link.upstreamCutLen = (link.lengthMetersOffset / totalLen) * shortcutLen
			link.downstreamCutLen = (link.lengthMetersOffset / totalLen) * downStreamCut
		}
	} else if link.downstreamShortCut {
		cutIdx := 0
		cutPlaceFound := false
		for i := link.lanesList[len(link.lanesList)-1]; i >= 0; i-- {
			if link.lengthMetersOffset > cutLen[i]+shortcutLen+cutLenMin {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			link.upstreamCutLen = cutLen[cutIdx]
			link.downstreamCutLen = shortcutLen
		} else {
			totalLen := cutLen[0] + shortcutLen + cutLenMin
			link.upstreamCutLen = (link.lengthMetersOffset / totalLen) * cutLen[0]
			link.downstreamCutLen = (link.lengthMetersOffset / totalLen) * shortcutLen
		}
	} else {
		cutIdx := 0
		cutPlaceFound := false
		for i := link.lanesList[len(link.lanesList)-1]; i >= 0; i-- {
			if link.lengthMetersOffset > cutLen[i]+math.Min(downStreamMaxCut, cutLen[i])+cutLenMin {
				cutIdx = i
				cutPlaceFound = true
				break
			}
		}
		if cutPlaceFound {
			link.upstreamCutLen = cutLen[cutIdx]
			link.downstreamCutLen = math.Min(downStreamMaxCut, cutLen[cutIdx])
		} else {
			downStreamCut := math.Min(downStreamMaxCut, cutLen[0])
			totalLen := downStreamCut + cutLen[0] + cutLenMin
			link.upstreamCutLen = (link.lengthMetersOffset / totalLen) * cutLen[0]
			link.downstreamCutLen = (link.lengthMetersOffset / totalLen) * downStreamCut
		}
	}
}

// Cuts redudant geometry
func (link *NetworkLink) performCut() {

	// Create copy for those since we will do mutations and want to keep original data
	breakpoints := make([]float64, len(link.breakpoints))
	copy(breakpoints, link.breakpoints)
	link.lanesListCut = make([]int, len(link.lanesList))
	copy(link.lanesListCut, link.lanesList)
	link.lanesChangeCut = make([][]int, len(link.lanesChange))
	copy(link.lanesChange, link.lanesChange)

	breakIdx := 1
	for breakIdx = 1; breakIdx < len(breakpoints); breakIdx++ {
		if breakpoints[breakIdx] > link.upstreamCutLen {
			break
		}
	}
	breakpoints = append(breakpoints[breakIdx:])
	breakpoints = append([]float64{link.upstreamCutLen}, breakpoints...)
	link.lanesListCut = link.lanesListCut[breakIdx-1:]
	link.lanesChange = link.lanesChange[breakIdx-1:]

	breakIdx = len(breakpoints) - 2
	for breakIdx := len(breakpoints) - 2; breakIdx >= 0; breakIdx-- {
		if link.lengthMetersOffset-breakpoints[breakIdx] > link.downstreamCutLen {
			break
		}
	}
	breakpoints = breakpoints[:breakIdx+1]
	breakpoints = append(breakpoints, link.lengthMetersOffset-link.downstreamCutLen)
	link.lanesListCut = link.lanesListCut[:breakIdx+1]
	link.lanesChange = link.lanesChange[:breakIdx+1]

	for i := range link.lanesListCut {
		start := breakpoints[i]
		end := breakpoints[i+1]
		geomCut := SubstringHaversine(link.geomOffset, start, end)
		geomEuclideanCut := lineToSpherical(geomCut)
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

type NetworkLinkMesoscopic struct {
	geom          orb.LineString
	geomEuclidean orb.LineString
	lanesNum      int
	lanesChange   []int

	ID NetworkLinkID

	sourceNodeID NetworkNodeID // Corresponds to ID of mesoscopic node (not to macro or OSM)
	targetNodeID NetworkNodeID // Corresponds to ID of mesoscopic node (not to macro or OSM)

	macroLinkID NetworkLinkID
	macroNodeID NetworkNodeID
	movementID  MovementID

	isConnection bool
}
