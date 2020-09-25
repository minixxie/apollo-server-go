package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/lalamove/mock-apollo-go/internal/routes/apollo"
	"github.com/lalamove/nui/nlogger"
	"github.com/sirupsen/logrus"
)

var (
	filePath     string
	configPort   int
	internalPort int
	pollTimeout  time.Duration
	logger       nlogger.Provider
)

func init() {
	flag.StringVar(&filePath, "file", "../../configs/example.yaml", "config filepath")
	flag.IntVar(&internalPort, "internal-port", 9090, "internal http server port")
	flag.IntVar(&configPort, "config-port", 8070, "config http server port")
	flag.DurationVar(&pollTimeout, "poll-timeout", time.Minute, "long poll timeout")
	flag.Parse()

	logger = nlogger.NewProvider(newLogger(logrus.InfoLevel))
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	internalRouter := httprouter.New()
	internalRoutes(internalRouter)
	internalSrv := &http.Server{
		Addr:    ":" + strconv.Itoa(internalPort),
		Handler: internalRouter,
	}
	go func() {
		if err := internalSrv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	router := httprouter.New()
	a, err := apollo.New(ctx, apollo.Config{
		ConfigPath:  filePath,
		PollTimeout: pollTimeout,
		Log:         logger,
	})
	if err != nil {
		log.Fatal(err)
	}
	a.Routes(router)
	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(configPort),
		Handler: router,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()

	// graceful shutdown
	<-termChan
	cancel()
	internalSrv.Close()
	srv.Close()
}

func internalRoutes(r *httprouter.Router) {
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
	r.POST("/logging", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		// ability to dynamically change logging level via http request
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
