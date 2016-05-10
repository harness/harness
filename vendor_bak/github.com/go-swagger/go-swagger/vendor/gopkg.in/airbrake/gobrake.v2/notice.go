package gobrake

import (
	"fmt"
	"net/http"
)

type Error struct {
	Type      string       `json:"type"`
	Message   string       `json:"message"`
	Backtrace []StackFrame `json:"backtrace"`
}

type Notice struct {
	Errors  []Error                `json:"errors"`
	Context map[string]interface{} `json:"context"`
	Env     map[string]interface{} `json:"environment"`
	Session map[string]interface{} `json:"session"`
	Params  map[string]interface{} `json:"params"`
}

func NewNotice(e interface{}, req *http.Request, depth int) *Notice {
	stack := stack(depth)
	notice := &Notice{
		Errors: []Error{
			{
				Type:      fmt.Sprintf("%T", e),
				Message:   fmt.Sprint(e),
				Backtrace: stack,
			},
		},
		Context: map[string]interface{}{
			"notifier": map[string]interface{}{
				"name":    "gobrake",
				"version": "2.0.3",
				"url":     "https://github.com/airbrake/gobrake",
			},
		},
		Env:     map[string]interface{}{},
		Session: map[string]interface{}{},
		Params:  map[string]interface{}{},
	}

	if req != nil {
		notice.Context["url"] = req.URL.String()
		if ua := req.Header.Get("User-Agent"); ua != "" {
			notice.Context["userAgent"] = ua
		}

		for k, v := range req.Header {
			if len(v) == 1 {
				notice.Env[k] = v[0]
			} else {
				notice.Env[k] = v
			}
		}

		if err := req.ParseForm(); err == nil {
			for k, v := range req.Form {
				if len(v) == 1 {
					notice.Params[k] = v[0]
				} else {
					notice.Params[k] = v
				}
			}
		}
	}

	return notice
}
