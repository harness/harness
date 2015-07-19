package cluster

type Cluster interface {
	CollectDockers() error
}
