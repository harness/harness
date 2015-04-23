package server

import (
	"net/http"

	"github.com/drone/drone/common"
)

type MockSession struct {
	Token *common.Token
}

func (s MockSession) GenerateToken(*common.Token) (string, error) {
	return "session", nil
}

func (s MockSession) GetLogin(*http.Request) *common.Token {
	return s.Token
}
