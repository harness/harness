package citadel

// Scheduler is able to return a yes or know decision on if the specified Engine is
// able to run the specified image
type Scheduler interface {
	// Schedule returns true if the engine can run the specified image
	Schedule(*Image, *Engine) (bool, error)
}

type ResourceManager interface {
	PlaceContainer(*Container, []*EngineSnapshot) (*EngineSnapshot, error)
}
