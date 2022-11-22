package osm2ch

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/paulmach/orb"
	"github.com/paulmach/osm"
)

type Way struct {
	ID     osm.WayID
	Oneway bool
	Nodes  osm.WayNodes
	TagMap osm.Tags
}

type WayData struct {
	// Flatten tags
	name              string
	highway           string
	railway           string
	aeroway           string
	junction          string
	area              string
	motorVehicle      string
	motorcar          string
	service           string
	foot              string
	bicycle           string
	building          string
	amenity           string
	leisure           string
	turnLanes         string
	turnLanesForward  string
	turnLanesBackward string
	/* Main information */
	TagMap             osm.Tags
	Nodes              []osm.NodeID
	ID                 osm.WayID
	osmTargetNodeID    osm.NodeID         // Medium
	osmSourceNodeID    osm.NodeID         // Medium
	isCycle            bool               // Medium
	isPureCycle        bool               // Well-done
	linkClass          LinkClass          // Medium
	linkType           LinkType           // Medium
	linkConnectionType LinkConnectionType // Medium
	geom               orb.LineString     // Medium
	capacity           int                // Well-done
	freeSpeed          float64            // Well-done
	maxSpeed           float64
	lanes              int
	lanesForward       int
	lanesBackward      int
	Oneway             bool
	OnewayDefault      bool
	IsReversed         bool
}

var (
	mphRegExp   = regexp.MustCompile(`\d+\.?\d* mph`)
	kmhRegExp   = regexp.MustCompile(`\d+\.?\d* km/h`)
	lanesRegExp = regexp.MustCompile(`\d+\.?\d*`)
)

func (way *WayData) flattenTags(verbose bool) {
	way.name = way.TagMap.Find("name")
	way.highway = way.TagMap.Find("highway")
	way.railway = way.TagMap.Find("railway")
	way.aeroway = way.TagMap.Find("aeroway")

	way.turnLanes = way.TagMap.Find("turn:lanes")
	way.turnLanesForward = way.TagMap.Find("turn:lanes:forward")
	way.turnLanesBackward = way.TagMap.Find("turn:lanes:backward")

	var err error

	lanes := way.TagMap.Find("lanes")
	if lanes != "" {
		lanesNum := lanesRegExp.FindString(lanes)
		if lanesNum != "" {
			way.lanes, err = strconv.Atoi(lanes)
			if err != nil {
				way.lanes = -1
				if verbose {
					fmt.Printf("[WARNING]: Provided `lanes` tag value should be an integer. Got '%s'. Way ID: '%d'\n", lanes, way.ID)
				}
			}
		}
	}

	lanesForward := way.TagMap.Find("lanes:forward")
	if lanesForward != "" {
		way.lanesForward, err = strconv.Atoi(lanesForward)
		if err != nil {
			way.lanesForward = -1
			if verbose {
				fmt.Printf("[WARNING]: Provided `lanes:forward` tag value should be an integer. Got '%s'. Way ID: '%d'\n", lanesForward, way.ID)
			}
		}
	}

	lanesBackward := way.TagMap.Find("lanes:backward")
	if lanesBackward != "" {
		way.lanesBackward, err = strconv.Atoi(lanesBackward)
		if err != nil {
			way.lanesBackward = -1
			if verbose {
				fmt.Printf("[WARNING]: Provided `lanes:backward` tag value should be an integer. Got '%s'. Way ID: '%d'\n", lanesBackward, way.ID)
			}
		}
	}

	maxSpeed := way.TagMap.Find("maxspeed")
	if maxSpeed != "" {
		maxSpeedValue := -1.0
		kmhMaxSpeed := kmhRegExp.FindString(maxSpeed)
		if kmhMaxSpeed != "" {
			maxSpeedValue, err = strconv.ParseFloat(kmhMaxSpeed, 64)
			if err != nil {
				maxSpeedValue = -1
				if verbose {
					fmt.Printf("[WARNING]: Provided `lanes:maxspeed (km/h)` tag value should be an float (or integer?). Got '%s'. Way ID: '%d'\n", kmhMaxSpeed, way.ID)
				}
			}
		} else {
			mphMaxSpeed := mphRegExp.FindString(maxSpeed)
			if mphMaxSpeed != "" {
				maxSpeedValue, err = strconv.ParseFloat(mphMaxSpeed, 64)
				if err != nil {
					maxSpeedValue = -1
					if verbose {
						fmt.Printf("[WARNING]: Provided `lanes:maxspeed (mph)` tag value should be an float (or integer?). Got '%s'. Way ID: '%d'\n", mphMaxSpeed, way.ID)
					}
				}
			}
		}
		way.maxSpeed = maxSpeedValue
	}

	// Rest of tags
	way.junction = way.TagMap.Find("junction")
	way.area = way.TagMap.Find("area")
	way.motorVehicle = way.TagMap.Find("motor_vehicle")
	way.motorcar = way.TagMap.Find("motorcar")
	way.service = way.TagMap.Find("service")
	way.foot = way.TagMap.Find("foot")
	way.bicycle = way.TagMap.Find("bicycle")
	way.building = way.TagMap.Find("building")
	way.amenity = way.TagMap.Find("amenity")
	way.leisure = way.TagMap.Find("leisure")
}

func (way *WayData) isPOI() bool {
	if way.building != "" || way.amenity != "" || way.leisure != "" {
		return true
	}
	return false
}

func (way *WayData) isHighwayPOI() bool {
	if _, ok := poiHighwayTags[way.highway]; ok {
		return true
	}
	return false
}

func (way *WayData) isRailwayPOI() bool {
	if _, ok := poiRailwayTags[way.railway]; ok {
		return true
	}
	return false
}

func (way *WayData) isAerowayPOI() bool {
	if _, ok := poiAerowayTags[way.aeroway]; ok {
		return true
	}
	return false
}

func (way *WayData) isHighway() bool {
	return way.highway != ""
}

func (way *WayData) isRailway() bool {
	return way.railway != ""
}

func (way *WayData) isAeroway() bool {
	return way.aeroway != ""
}

func (way *WayData) isHighwayNegligible() bool {
	_, ok := negligibleHighwayTags[way.highway]
	return ok
}