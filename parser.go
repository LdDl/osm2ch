package osm2ch

import (
	"fmt"
	"strings"
)

type Parser struct {
	filename         string
	networkTypes     []string
	linkTypes        []string
	preparePOI       bool
	poiSamplingRatio float64
	strictMode       bool
	offset           string
	minNodes         int
	combine          bool
	defaultLanes     map[string]interface{}
	defaultSpeed     map[string]float64
	defaultCapacity  map[string]float64
	startNodeID      int
	startLinkID      int
}

func (parser *Parser) String() string {
	return fmt.Sprintf(`
Network parser parameters:
	filename: '%s'
	network_types: '%s'
	link_types: '%s'
	prepare POI?: %t
	POI_sampling_ratio: %f
	strict_mode enabled?: %t
	offset: '%s'
	min_nodes: %d
	combine: %t
	default_lanes: %v
	default_speed: %v
	default_capacity: %v
	start_node_id: %d
	start_link_id: %d
	`,
		parser.filename,
		strings.Join(parser.networkTypes, ","),
		strings.Join(parser.linkTypes, ","),
		parser.preparePOI,
		parser.poiSamplingRatio,
		parser.strictMode,
		parser.offset,
		parser.minNodes,
		parser.combine,
		parser.defaultLanes,
		parser.defaultSpeed,
		parser.defaultCapacity,
		parser.startNodeID,
		parser.startLinkID,
	)
}

func NewParser(fileName string, options ...func(*Parser)) *Parser {
	parser := &Parser{
		filename:    fileName,
		preparePOI:  false,
		strictMode:  false,
		startNodeID: 0,
		startLinkID: 0,
	}
	for _, option := range options {
		option(parser)
	}
	return parser
}

func WithNetworkTypes(networkTypes []string) func(*Parser) {
	return func(parser *Parser) {
		parser.networkTypes = networkTypes
	}
}

func WithLinkTypes(linkTypes []string) func(*Parser) {
	return func(parser *Parser) {
		parser.linkTypes = linkTypes
	}
}

func WithPreparePOI(preparePOI bool) func(*Parser) {
	return func(parser *Parser) {
		parser.preparePOI = preparePOI
	}
}

func WithPOISamplingRatio(poiSamplingRatio float64) func(*Parser) {
	return func(parser *Parser) {
		parser.poiSamplingRatio = poiSamplingRatio
	}
}

func WithStrictMode(strictMode bool) func(*Parser) {
	return func(parser *Parser) {
		parser.strictMode = strictMode
	}
}

func WithOffset(offset string) func(*Parser) {
	return func(parser *Parser) {
		parser.offset = offset
	}
}

func WithMinNodes(minNodes int) func(*Parser) {
	return func(parser *Parser) {
		parser.minNodes = minNodes
	}
}

func WithCombine(combine bool) func(*Parser) {
	return func(parser *Parser) {
		parser.combine = combine
	}
}

func WithDefaultLanes(defaultLanes map[string]interface{}) func(*Parser) {
	return func(parser *Parser) {
		parser.defaultLanes = defaultLanes
	}
}

func WithDefaultSpeed(defaultSpeed map[string]float64) func(*Parser) {
	return func(parser *Parser) {
		parser.defaultSpeed = defaultSpeed
	}
}

func WithDefaultCapacity(defaultCapacity map[string]float64) func(*Parser) {
	return func(parser *Parser) {
		parser.defaultCapacity = defaultCapacity
	}
}

func WithStartNodeID(startNodeID int) func(*Parser) {
	return func(parser *Parser) {
		parser.startNodeID = startNodeID
	}
}

func WithStartLinkID(startLinkID int) func(*Parser) {
	return func(parser *Parser) {
		parser.startLinkID = startLinkID
	}
}
