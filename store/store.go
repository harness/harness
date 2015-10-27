package store

type Store interface {
	Nodes() NodeStore
	Users() UserStore
	Repos() RepoStore
	Keys() KeyStore
	Builds() BuildStore
	Jobs() JobStore
	Logs() LogStore
}

type store struct {
	name   string
	nodes  NodeStore
	users  UserStore
	repos  RepoStore
	keys   KeyStore
	builds BuildStore
	jobs   JobStore
	logs   LogStore
}

func (s *store) Nodes() NodeStore   { return s.nodes }
func (s *store) Users() UserStore   { return s.users }
func (s *store) Repos() RepoStore   { return s.repos }
func (s *store) Keys() KeyStore     { return s.keys }
func (s *store) Builds() BuildStore { return s.builds }
func (s *store) Jobs() JobStore     { return s.jobs }
func (s *store) Logs() LogStore     { return s.logs }
func (s *store) String() string     { return s.name }

func New(
	name string,
	nodes NodeStore,
	users UserStore,
	repos RepoStore,
	keys KeyStore,
	builds BuildStore,
	jobs JobStore,
	logs LogStore,
) Store {
	return &store{
		name,
		nodes,
		users,
		repos,
		keys,
		builds,
		jobs,
		logs,
	}
}
