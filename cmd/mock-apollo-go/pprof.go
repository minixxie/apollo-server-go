package main

import (
	"net/http"
	"net/http/pprof"

	"github.com/julienschmidt/httprouter"
)

func pprofRoutes(r *httprouter.Router) {
	r.GET("/debug/pprof/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		pprof.Index(w, r)
	})
	r.GET("/debug/pprof/:profile", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		switch ps.ByName("profile") {
		case "cmdline":
			pprof.Cmdline(w, r)
		case "profile":
			pprof.Profile(w, r)
		case "symbol":
			pprof.Symbol(w, r)
		case "trace":
			pprof.Trace(w, r)
		default:
			pprof.Index(w, r)
		}
	})
}
