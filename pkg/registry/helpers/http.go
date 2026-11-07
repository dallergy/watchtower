package helpers

import (
	"net/http"
	"time"
)

// RequestTimeout limits how long a single request to a registry may take
const RequestTimeout = 30 * time.Second

var httpClient = &http.Client{
	Timeout:   RequestTimeout,
	Transport: http.DefaultTransport.(*http.Transport).Clone(),
}

// HTTPClient returns the client used for requests to registries. It verifies TLS certificates
// against the system roots, which can be extended using SSL_CERT_FILE or SSL_CERT_DIR.
func HTTPClient() *http.Client {
	return httpClient
}
