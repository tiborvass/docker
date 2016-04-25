package server

import (
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"strconv"

	"github.com/gorilla/mux"
)

func profilerSetup(mainRouter *mux.Router, path string) {
	var r = mainRouter.PathPrefix(path).Subrouter()
	r.HandleFunc("/vars", expVars)
	r.HandleFunc("/pprof/", pprof.Index)
	r.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/pprof/profile", pprof.Profile)
	r.HandleFunc("/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/pprof/trace", pprof.Trace)
	r.HandleFunc("/pprof/block", pprof.Handler("block").ServeHTTP)
	r.HandleFunc("/pprof/block/rate", blockRate)
	r.HandleFunc("/pprof/heap", pprof.Handler("heap").ServeHTTP)
	r.HandleFunc("/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	r.HandleFunc("/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
}

// Replicated from expvar.go as not public.
func expVars(w http.ResponseWriter, r *http.Request) {
	first := true
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}

func blockRate(w http.ResponseWriter, r *http.Request) {
	var (
		rate = 0
		err  error
	)
	if rateString := r.FormValue("rate"); rateString != "" {
		rate, err = strconv.Atoi(r.FormValue("rate"))
		if err != nil {
			http.Error(w, "rate has to be an integer, cf https://golang.org/pkg/runtime/#SetBlockProfileRate", http.StatusBadRequest)
			return
		}
	}
	runtime.SetBlockProfileRate(rate)
}
