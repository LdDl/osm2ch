package osm2ch

type OSMRelation struct {
	osmID int
	name  string

	building string
	amenity  string
	leisure  string
}
