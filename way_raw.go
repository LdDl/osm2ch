package osm2ch

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/paulmach/osm"
)

type Way struct {
	ID     osm.WayID
	Oneway bool
	Nodes  osm.WayNodes
	TagMap osm.Tags
}

type WayData struct {
	name              string
	highway           string
	railway           string
	aeroway           string
	junction          string
	area              string
	motorVehicle      string
	access            string
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
	TagMap            osm.Tags
	// geom               orb.LineString
	Nodes              []osm.NodeID
	segments           [][]osm.NodeID
	osmSourceNodeID    osm.NodeID
	lanesBackward      int
	lanesForward       int
	lanes              int
	maxSpeed           float64
	capacity           int
	ID                 osm.WayID
	freeSpeed          float64
	osmTargetNodeID    osm.NodeID
	linkConnectionType LinkConnectionType
	linkType           LinkType
	linkClass          LinkClass
	isPureCycle        bool
	isCycle            bool
	Oneway             bool
	OnewayDefault      bool
	IsReversed         bool
}

var (
	mphRegExp   = regexp.MustCompile(`\d+\.?\d* mph`)
	kmhRegExp   = regexp.MustCompile(`\d+\.?\d* km/h`)
	lanesRegExp = regexp.MustCompile(`\d+\.?\d*`)
)

func (way *WayData) processTags(verbose bool) {
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
	way.access = ""
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

func (way *WayData) findIncludedAgent(agentType AgentType) bool {
	accessType, ok := agentsAccessIncludeValues[agentType]
	if !ok {
		return false
	}
	switch agentType {
	case AGENT_AUTO:
		// Check `motor_vehicle`
		if _, ok := accessType[ACCESS_MOTOR_VEHICLE][way.motorVehicle]; ok {
			return true
		}
		// Check `motorcar`
		if _, ok := accessType[ACCESS_MOTORCAR][way.motorcar]; ok {
			return true
		}
	case AGENT_BIKE:
		// Check `bicycle`
		if _, ok := accessType[ACCESS_BICYCLE][way.bicycle]; ok {
			return true
		}
	case AGENT_WALK:
		// Check `foot`
		if _, ok := accessType[ACCESS_FOOT][way.foot]; ok {
			return true
		}
	default:
		return false
	}
	return false
}

func (way *WayData) findExcludedAgent(agentType AgentType) bool {
	accessType, ok := agentsAccessExcludeValues[agentType]
	if !ok {
		return true
	}
	switch agentType {
	case AGENT_AUTO:
		// Check `highway`
		if _, ok := accessType[ACCESS_HIGHWAY][way.highway]; ok {
			return false
		}
		// Check `motor_vehicle`
		if _, ok := accessType[ACCESS_MOTOR_VEHICLE][way.motorVehicle]; ok {
			return false
		}
		// Check `motorcar`
		if _, ok := accessType[ACCESS_MOTORCAR][way.motorcar]; ok {
			return false
		}
		// Check `access`
		if _, ok := accessType[ACCESS_OSM_ACCESS][way.access]; ok {
			return false
		}
		// Check `service`
		if _, ok := accessType[ACCESS_SERVICE][way.service]; ok {
			return false
		}
	case AGENT_BIKE:
		// Check `highway`
		if _, ok := accessType[ACCESS_HIGHWAY][way.highway]; ok {
			return false
		}
		// Check `bicycle`
		if _, ok := accessType[ACCESS_BICYCLE][way.bicycle]; ok {
			return false
		}
		// Check `service`
		if _, ok := accessType[ACCESS_SERVICE][way.service]; ok {
			return false
		}
		// Check `access`
		if _, ok := accessType[ACCESS_OSM_ACCESS][way.access]; ok {
			return false
		}
	case AGENT_WALK:
		// Check `highway`
		if _, ok := accessType[ACCESS_HIGHWAY][way.highway]; ok {
			return false
		}
		// Check `foot`
		if _, ok := accessType[ACCESS_FOOT][way.foot]; ok {
			return false
		}
		// Check `service`
		if _, ok := accessType[ACCESS_SERVICE][way.service]; ok {
			return false
		}
		// Check `access`
		if _, ok := accessType[ACCESS_OSM_ACCESS][way.access]; ok {
			return false
		}
	default:
		return true
	}

	return true
}

func (way *WayData) getAllowableAgentType() []AgentType {
	allowedAgents := []AgentType{}
	for agentType := range agentTypesAll {
		included := way.findIncludedAgent(agentType)
		if included {
			allowedAgents = append(allowedAgents, agentType)
			continue
		}
		excluded := way.findExcludedAgent(agentType)
		if excluded {
			allowedAgents = append(allowedAgents, agentType)
			continue
		}
	}
	return allowedAgents
}
