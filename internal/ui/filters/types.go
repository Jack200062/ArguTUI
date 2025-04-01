package filters

const (
	FilterTypeProject      = "project"
	FilterTypeHealth       = "health"
	FilterTypeSync         = "sync"
	FilterTypeResourceKind = "kind"
)

type Filter struct {
	Type  string
	Value string
}

type FilterCategory struct {
	Type      string
	Title     string
	Options   []string
	Shortcuts map[string]rune
}

type FilterResult struct {
	Canceled bool
	Filters  []Filter
}
