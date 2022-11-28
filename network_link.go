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
	wasBidirectional   bool
}

type DirectionType uint16

const (
	DIRECTION_FORWARD = DirectionType(iota + 1)
	DIRECTION_BACKWARD
)

func networkLinkFromOSM(id NetworkLinkID, sourceNodeID, targetNodeID NetworkNodeID, direction DirectionType, wayOSM *WayData, segmentNodes []*Node) *NetworkLink {
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
