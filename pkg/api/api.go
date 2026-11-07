package api

import (
	"crypto/subtle"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

const tokenMissingMsg = "api token is empty or has not been set. exiting"

// Timeouts protecting the API from clients that keep connections open without completing
// requests. There is no write timeout, as update requests only finish when the update does.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	idleTimeout       = 2 * time.Minute
)

// API is the http server responsible for serving the HTTP API endpoints
type API struct {
	Token       string
	hasHandlers bool
}

// New is a factory function creating a new API instance
func New(token string) *API {
	return &API{
		Token:       token,
		hasHandlers: false,
	}
}

// RequireToken is wrapper around http.HandleFunc that checks token validity
func (api *API) RequireToken(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		want := "Bearer " + api.Token
		if subtle.ConstantTimeCompare([]byte(auth), []byte(want)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		log.Debug("Valid token found.")
		fn(w, r)
	}
}

// RegisterFunc is a wrapper around http.HandleFunc that also sets the flag used to determine whether to launch the API
func (api *API) RegisterFunc(path string, fn http.HandlerFunc) {
	api.hasHandlers = true
	http.HandleFunc(path, api.RequireToken(fn))
}

// RegisterHandler is a wrapper around http.Handler that also sets the flag used to determine whether to launch the API
func (api *API) RegisterHandler(path string, handler http.Handler) {
	api.hasHandlers = true
	http.Handle(path, api.RequireToken(handler.ServeHTTP))
}

// Start the API and serve over HTTP. Requires an API Token to be set.
func (api *API) Start(block bool) error {

	if !api.hasHandlers {
		log.Debug("Watchtower HTTP API skipped.")
		return nil
	}

	if api.Token == "" {
		log.Fatal(tokenMissingMsg)
	}

	if block {
		runHTTPServer()
	} else {
		go func() {
			runHTTPServer()
		}()
	}
	return nil
}

func runHTTPServer() {
	server := &http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		IdleTimeout:       idleTimeout,
	}
	log.Fatal(server.ListenAndServe())
}
