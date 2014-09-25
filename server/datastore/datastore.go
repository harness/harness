package datastore

type Datastore interface {
	Userstore
	Permstore
	Repostore
	Commitstore
}
