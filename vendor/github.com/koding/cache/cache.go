package cache

// Cache is the contract for all of the cache backends that are supported by
// this package
type Cache interface {
	// Get returns single item from the backend if the requested item is not
	// found, returns NotFound err
	Get(key string) (interface{}, error)

	// Set sets a single item to the backend
	Set(key string, value interface{}) error

	// Delete deletes single item from backend
	Delete(key string) error
}
