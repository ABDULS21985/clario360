package profiling

import (
	"net/http"
	"net/http/pprof"

	"github.com/go-chi/chi/v5"
)

// RegisterPprof conditionally registers pprof endpoints on the chi router.
//
// If enabled=false (production default): does nothing.
// If enabled=true (set via ENABLE_PPROF=true env var): registers all pprof handlers
// under /debug/pprof/*.
//
// SECURITY: pprof endpoints expose sensitive runtime details. In production, they
// should only be exposed on an internal admin port (not the public API port).
func RegisterPprof(router chi.Router, enabled bool) {
	if !enabled {
		return
	}

	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)

	router.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	router.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	router.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	router.Handle("/debug/pprof/block", pprof.Handler("block"))
	router.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	router.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
}

// RegisterPprofMux registers pprof endpoints on a standard http.ServeMux.
// Useful when the admin server doesn't use chi.
func RegisterPprofMux(mux *http.ServeMux, enabled bool) {
	if !enabled {
		return
	}

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
}
