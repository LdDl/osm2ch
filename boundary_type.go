package osm2ch

type BoundaryType uint16

const (
	BOUNDARY_NONE = BoundaryType(iota)
	BOUNDARY_INCOME_ONLY
	BOUNDARY_INCOME_OUTCOME
)

func (iotaIdx BoundaryType) String() string {
	return [...]string{"none", "income_only", "income_outcome"}[iotaIdx]
}
