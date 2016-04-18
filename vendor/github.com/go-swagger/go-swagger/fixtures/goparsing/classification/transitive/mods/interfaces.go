package mods

// ExtraInfo is an interface for things that have extra info
// swagger:model extra
type ExtraInfo interface {
	// swagger:name extraInfo
	ExtraInfo() string
}

// EmbeddedColor is a color
//
// swagger:model color
type EmbeddedColor interface {
	// swagger:name colorName
	ColorName() string
}
