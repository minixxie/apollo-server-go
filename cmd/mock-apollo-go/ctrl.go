package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/sirupsen/logrus"
)

// registers ctrl routes used for controlling certain features/abilities of this process
func ctrlRoutes(r *httprouter.Router) {
	// ability to dynamically change logging level
	r.PATCH("/ctrl/logging", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		v, ok := r.URL.Query()["level"]
		if !ok && len(v) != 1 {
			w.WriteHeader(400)
			return
		}
		switch v[0] {
		case "debug":
			logger.Replace(newLogger(logrus.DebugLevel))
		case "info":
			logger.Replace(newLogger(logrus.InfoLevel))
		case "warn":
			logger.Replace(newLogger(logrus.WarnLevel))
		case "error":
			logger.Replace(newLogger(logrus.ErrorLevel))
		default:
			w.WriteHeader(400)
			return
		}
		w.Write([]byte("OK"))
	})
}
