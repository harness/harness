package jsonrpc2

// CallOption is an option that can be provided to (*Conn).Call to
// configure custom behavior. See Meta.
type CallOption interface {
	apply(r *Request) error
}

type callOptionFunc func(r *Request) error

func (c callOptionFunc) apply(r *Request) error { return c(r) }

// Meta returns a call option which attaches the given meta object to
// the JSON-RPC 2.0 request (this is a Sourcegraph extension to JSON
// RPC 2.0 for carrying metadata).
func Meta(meta interface{}) CallOption {
	return callOptionFunc(func(r *Request) error {
		return r.SetMeta(meta)
	})
}

// PickID returns a call option which sets the ID on a request. Care must be
// taken to ensure there are no conflicts with any previously picked ID, nor
// with the default sequence ID.
func PickID(id ID) CallOption {
	return callOptionFunc(func(r *Request) error {
		r.ID = id
		return nil
	})
}
