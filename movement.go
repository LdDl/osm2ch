package osm2ch

import (
	"fmt"
	"math"
	"time"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geo"
	"github.com/paulmach/osm"
)

type MovementID int

type Movement struct {
	allowedAgentTypes []AgentType

	geom          orb.Geometry
	geomEuclidean orb.LineString

	ID              MovementID
	NodeID          NetworkNodeID
	osmNodeID       osm.NodeID
	fromOsmNodeID   osm.NodeID
	toOsmNodeID     osm.NodeID
	IncomingLinkID  NetworkLinkID
	OutcomingLinkID NetworkLinkID

	movementCompositeType            MovementCompositeType
	movementType                     MovementType
	controlType                      ControlType
	incomeLaneStart, incomeLaneEnd   int
	outcomeLaneStart, outcomeLaneEnd int
	lanesNum                         int
}

// movementBetweenLines returns movement information for given lines pair
//
// Note: panics if number of points in any line is less than 2
//
func movementBetweenLines(l1 orb.LineString, l2 orb.LineString) (MovementCompositeType, MovementType) {
	startL1, endL1 := l1[0], l1[len(l1)-1]
	endL2 := l2[len(l2)-1]

	var direction string

	angle1 := math.Atan2(endL1.Y()-startL1.Y(), endL1.X()-startL1.X())
	if -0.75*math.Pi <= angle1 && angle1 < -0.25*math.Pi {
		direction = "SB"
	} else if -0.25*math.Pi <= angle1 && angle1 < 0.25*math.Pi {
		direction = "EB"
	} else if 0.25*math.Pi <= angle1 && angle1 < 0.75*math.Pi {
		direction = "NB"
	} else {
		direction = "WB"
	}

	angle2 := math.Atan2(endL2.Y()-endL1.Y(), endL2.X()-endL1.X())

	angleDiff := angle2 - angle1
	if angleDiff < -1*math.Pi {
		angleDiff += 2 * math.Pi
	}
	if angleDiff > math.Pi {
		angleDiff -= 2 * math.Pi
	}

	var movement string
	var movementType MovementType
	if -0.25*math.Pi <= angleDiff && angleDiff <= 0.25*math.Pi {
		movement = "T"
		movementType = MOVEMENT_THRU
	} else if angleDiff < -0.25*math.Pi {
		movement = "R"
		movementType = MOVEMENT_RIGHT
	} else if angleDiff <= 0.75*math.Pi {
		movement = "L"
		movementType = MOVEMENT_LEFT
	} else {
		movement = "U"
		movementType = MOVEMENT_U_TURN
	}

	return movementTxt[direction+movement], movementType
}

// movementGeomBetweenLines returns movement geometry for given lines pair
//
// Note: panics if number of points in any line is less than 2
//
func movementGeomBetweenLines(l1 orb.LineString, l2 orb.LineString) orb.LineString {
	indent1 := indentationThreshold
	length1 := geo.Length(l1)
	if length1 <= indent1 {
		indent1 = length1 / 2.0
	}
	point1, _ := geo.PointAtDistanceAlongLine(l1, length1-indent1) // Ident from link end

	indent2 := indentationThreshold
	length2 := geo.Length(l2)
	if length2 <= indent2 {
		indent2 = length2 / 2.0
	}

	point2, _ := geo.PointAtDistanceAlongLine(l2, indent2)
	return orb.LineString{point1, point2}
}

const (
	indentationThreshold = 8.0
)

type MovementType uint16

const (
	MOVEMENT_THRU = MovementType(iota + 1)
	MOVEMENT_RIGHT
	MOVEMENT_LEFT
	MOVEMENT_U_TURN

	MOVEMENT_UNDEFINED = MovementType(0)
)

func (iotaIdx MovementType) String() string {
	return [...]string{"undefined", "thru", "right", "left", "uturn"}[iotaIdx]
}

type MovementCompositeType uint16

const (
	MOVEMENT_SBT = MovementCompositeType(iota + 1)
	MOVEMENT_SBR
	MOVEMENT_SBL
	MOVEMENT_SBU
	MOVEMENT_EBT
	MOVEMENT_EBR
	MOVEMENT_EBL
	MOVEMENT_EBU
	MOVEMENT_NBT
	MOVEMENT_NBR
	MOVEMENT_NBL
	MOVEMENT_NBU
	MOVEMENT_WBT
	MOVEMENT_WBR
	MOVEMENT_WBL
	MOVEMENT_WBU
	MOVEMENT_NONE = MovementCompositeType(0)
)

var (
	movementTxt = map[string]MovementCompositeType{
		"SBT": MOVEMENT_SBT,
		"SBR": MOVEMENT_SBR,
		"SBL": MOVEMENT_SBL,
		"SBU": MOVEMENT_SBU,
		"EBT": MOVEMENT_EBT,
		"EBR": MOVEMENT_EBR,
		"EBL": MOVEMENT_EBL,
		"EBU": MOVEMENT_EBU,
		"NBT": MOVEMENT_NBT,
		"NBR": MOVEMENT_NBR,
		"NBL": MOVEMENT_NBL,
		"NBU": MOVEMENT_NBU,
		"WBT": MOVEMENT_WBT,
		"WBR": MOVEMENT_WBR,
		"WBL": MOVEMENT_WBL,
		"WBU": MOVEMENT_WBU,
	}
)

func (iotaIdx MovementCompositeType) String() string {
	return [...]string{"undefined", "SBT", "SBR", "SBL", "SBU", "EBT", "EBR", "EBL", "EBU", "NBT", "NBR", "NBL", "NBU", "WBT", "WBR", "WBL", "WBU"}[iotaIdx]
}

func (net *NetworkMacroscopic) genMovement(verbose bool) error {
	if verbose {
		fmt.Print("Preparing movements...")
	}
	st := time.Now()
	mvmtID := MovementID(0)
	for _, node := range net.nodes {
		mvmtList := node.genMovement(&mvmtID, net.links)
		for _, mvmt := range mvmtList {
			net.movement[mvmt.ID] = mvmt
		}
	}
	if verbose {
		fmt.Printf("Done in %v\n\tMovements: %d\n", time.Since(st), len(net.movement))
	}
	return nil
}
