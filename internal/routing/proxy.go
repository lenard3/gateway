package routing

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

const defaultTimeout = 30 * time.Second

type backendEntry struct {
	Proxy   *httputil.ReverseProxy
	Timeout time.Duration
	Headers map[string]string
}

type Table struct {
	entries map[string]backendEntry
	Routes  []Route
}

// NewTable fills the Table struct with the proxy and routes.
// It parses the urls of the backend services and builds a ReverseProxy for it.
// Then it adds all routes to the service to the table.
func NewTable(file *File) (*Table, error) {
	entries := make(map[string]backendEntry)

	for _, backend := range file.Backends {
		target, taErr := url.Parse(backend.URL)
		if taErr != nil {
			return nil, fmt.Errorf("parse backend %q url: %w", backend.Name, taErr)
		}
		proxy := &httputil.ReverseProxy{
			// Rewriting the ProxyRequest
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(target)
				pr.SetXForwarded()
				for k, v := range backend.Headers {
					pr.Out.Header.Set(k, v)
				}
			},
			// Adding error handler to send a uniform error on gateway unreachable
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				slog.Error("proxy error", "error", err, "path", r.URL.Path)
				w.Header().Set("content-type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{
						"code":    "bad_gateway",
						"message": "backend unreachable",
					},
				})
			},
		}

		var timeout time.Duration
		var err error
		if backend.Timeout == "" {
			timeout = defaultTimeout
		} else {
			timeout, err = time.ParseDuration(backend.Timeout)
			if err != nil {
				return nil, fmt.Errorf("backend %q: invalid timeout %q: %w", backend.Name, backend.Timeout, err)
			}
		}

		entries[backend.Name] = backendEntry{
			Proxy:   proxy,
			Timeout: timeout,
			Headers: backend.Headers,
		}

	}

	routes := make([]Route, len(file.Routes))
	copy(routes, file.Routes)

	return &Table{
		entries: entries,
		Routes:  routes,
	}, nil
}

// ProxyFor checks if proxy exists for a given name.
func (t *Table) ProxyFor(name string) (backendEntry, bool) {
	entry, ok := t.entries[name]
	return entry, ok
}
