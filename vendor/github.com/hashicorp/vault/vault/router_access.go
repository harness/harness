package vault

// RouterAccess provides access into some things necessary for testing
type RouterAccess struct {
	c *Core
}

func NewRouterAccess(c *Core) *RouterAccess {
	return &RouterAccess{c: c}
}

func (r *RouterAccess) StoragePrefixByAPIPath(path string) (string, string, bool) {
	return r.c.router.MatchingStoragePrefixByAPIPath(path)
}
