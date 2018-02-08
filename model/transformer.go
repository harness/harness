package model

type TransformerService interface {
	Transform(repo *Repo, data []byte) ([]byte, error)
}
