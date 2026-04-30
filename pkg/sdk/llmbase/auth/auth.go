package auth

import "net/http"

// Auth is a credential-application strategy. Implementations mutate the
// outgoing request to add whatever headers (or other state) the vendor
// requires.
type Auth interface {
	// Apply adds credentials to the request. Called per request, before send.
	Apply(req *http.Request)
}
