package osm2ch

import (
	"errors"
	"strconv"
	"strings"
)

// OsmConfiguration Allows to filter ways by certain tags from OSM data
type OsmConfiguration struct {
	EntityName string // Currrently we support 'highway' only
	Tags       []string
	CostType   string
	VLim       *VelocityLimit
}

// VelocityLimit if cost_type = minutes or seconds
type VelocityLimit struct {
	Tag     string // osm tag value
	Default float64
}

// CheckTag Checks if incoming tag is represented in configuration
func (cfg *OsmConfiguration) CheckTag(tag string) bool {
	for i := range cfg.Tags {
		if cfg.Tags[i] == tag {
			return true
		}
	}
	return false
}

// ParseCostType parsing flag cost_type
func (cfg *OsmConfiguration) ParseCostType(tag *string) error {
	var err error
	paramsTag := strings.Split(*tag, "->")

	switch paramsTag[0] {
	case "kilometers":
		cfg.CostType = "kilometers"
	case "meters":
		cfg.CostType = "meters"
	case "hours":
		cfg.CostType = "hours"
	case "seconds":
		cfg.CostType = "seconds"
	default:
		err = errors.New("first param bad for tag cost_type")
	}
	if err != nil {
		return err
	}
	if cfg.CostType == "kilometers" || cfg.CostType == "meters" {
		return nil
	}
	cfg.VLim = &VelocityLimit{}
	if len(paramsTag) >= 2 {
		switch paramsTag[1] {
		case "static":
			cfg.VLim.Tag = "static"
		case "maxspeed":
			cfg.VLim.Tag = "maxspeed"
		default:
			err = errors.New("second param bad for tag cost_type")
		}
	}
	if err != nil {
		return err
	}
	if len(paramsTag) == 3 {
		value, err := strconv.ParseFloat(paramsTag[2], 64)
		if err != nil {
			return err
		}
		cfg.VLim.Default = value
		return nil
	}
	if cfg.CostType == "hours" {
		cfg.VLim.Default = 40.0 // km/h
	}
	if cfg.CostType == "seconds" {
		cfg.VLim.Default = 11.11 // m/s
	}
	return err
}
