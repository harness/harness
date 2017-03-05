package jsonrpc2

import (
	"encoding/json"
	"log"
	"sync"
)

// ConnOpt is the type of function that can be passed to NewConn to
// customize the Conn before it is created.
type ConnOpt func(*Conn)

// OnRecv causes all requests received on conn to invoke f(req, nil)
// and all responses to invoke f(req, resp),
func OnRecv(f func(*Request, *Response)) ConnOpt {
	return func(c *Conn) { c.onRecv = f }
}

// OnSend causes all requests sent on conn to invoke f(req, nil) and
// all responses to invoke f(nil, resp),
func OnSend(f func(*Request, *Response)) ConnOpt {
	return func(c *Conn) { c.onSend = f }
}

// LogMessages causes all messages sent and received on conn to be
// logged using the provided logger.
func LogMessages(log *log.Logger) ConnOpt {
	return func(c *Conn) {
		// Remember reqs we have received so we can helpfully show the
		// request method in OnSend for responses.
		var (
			mu         sync.Mutex
			reqMethods = map[ID]string{}
		)

		OnRecv(func(req *Request, resp *Response) {
			switch {
			case req != nil && resp == nil:
				mu.Lock()
				reqMethods[req.ID] = req.Method
				mu.Unlock()

				params, _ := json.Marshal(req.Params)
				if req.Notif {
					log.Printf("--> notif: %s: %s", req.Method, params)
				} else {
					log.Printf("--> request #%s: %s: %s", req.ID, req.Method, params)
				}

			case resp != nil:
				var method string
				if req != nil {
					method = req.Method
				} else {
					method = "(no matching request)"
				}
				switch {
				case resp.Result != nil:
					result, _ := json.Marshal(resp.Result)
					log.Printf("--> result #%s: %s: %s", resp.ID, method, result)
				case resp.Error != nil:
					err, _ := json.Marshal(resp.Error)
					log.Printf("--> error #%s: %s: %s", resp.ID, method, err)
				}
			}
		})(c)
		OnSend(func(req *Request, resp *Response) {
			switch {
			case req != nil:
				params, _ := json.Marshal(req.Params)
				if req.Notif {
					log.Printf("<-- notif: %s: %s", req.Method, params)
				} else {
					log.Printf("<-- request #%s: %s: %s", req.ID, req.Method, params)
				}

			case resp != nil:
				mu.Lock()
				method := reqMethods[resp.ID]
				delete(reqMethods, resp.ID)
				mu.Unlock()
				if method == "" {
					method = "(no previous request)"
				}

				if resp.Result != nil {
					result, _ := json.Marshal(resp.Result)
					log.Printf("<-- result #%s: %s: %s", resp.ID, method, result)
				} else {
					err, _ := json.Marshal(resp.Error)
					log.Printf("<-- error #%s: %s: %s", resp.ID, method, err)
				}
			}
		})(c)
	}
}
