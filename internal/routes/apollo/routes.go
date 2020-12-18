package apollo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/lalamove/mock-apollo-go/pkg/flagarray"
	"github.com/lalamove/mock-apollo-go/pkg/longpoll"
	"github.com/lalamove/mock-apollo-go/pkg/watcher"
	"github.com/lalamove/nui/nlogger"
)

// Config is an object that stores the package config
type Config struct {
	Log         nlogger.Provider
	ConfigPath  flagarray.FlagArray
	PollTimeout time.Duration
	Port        int
}

// Apollo serves the mock apollo http routes
type Apollo struct {
	mu    sync.Mutex
	cfg   Config
	w     []*watcher.Watcher
	polls map[*longpoll.Poll]bool
}

// New creates a new Apollo
func New(ctx context.Context, cfg Config) (*Apollo, error) {
	validateConfig(&cfg)
	a := &Apollo{
		cfg:   cfg,
		polls: make(map[*longpoll.Poll]bool),
	}
	// start watching the config file
	for _, f := range a.cfg.ConfigPath {
		if err := a.watch(ctx, f); err != nil {
			return a, err
		}
	}
	return a, nil
}

func validateConfig(cfg *Config) {
	if cfg.Log == nil {
		cfg.Log = nlogger.NewProvider(nlogger.New(os.Stdout, ""))
	}
}

// Routes registers the http handles for Apollo
func (a *Apollo) Routes(r *httprouter.Router) {
	r.GET("/healthz", a.healthz)
	r.GET("/configs/:appId/:cluster/:namespace", a.queryConfig)
	r.GET("/configfiles/json/:appId/:cluster/:namespace", a.queryConfigJSON)
	r.GET("/services/config", a.queryService)
	r.GET("/notifications/v2", a.longPolling)

	// capture invalid http calls
	r.HandleMethodNotAllowed = false
	r.NotFound = &notFoundHandler{a.cfg.Log}
}

type notFoundHandler struct {
	log nlogger.Provider
}

func (h *notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.log.Get().Warn(fmt.Sprintf("http path not found: %s %s", r.Method, r.URL.String()))
	w.WriteHeader(404)
	w.Write([]byte("path not found"))
}

func (a *Apollo) healthz(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// make sure there's no deadlock
	a.mu.Lock()
	defer a.mu.Unlock()

	w.Write([]byte("OK"))
}

func (a *Apollo) getNamespace(appID string, cluster string, namespace string) (watcher.Namespace, error) {
	for _, w := range a.w {
		cm := w.Config()
		fmt.Printf("%+v\n", cm)
		s := strings.TrimSuffix(namespace, ".properties")
		for _, v := range cm {
			ns, ok := v[cluster][s]
			if ok {
				return ns, nil
			}
			ns, ok = v[cluster][s+".properties"]
			if ok {
				return ns, nil
			}
		}
	}
	return watcher.Namespace{}, fmt.Errorf("namespace no found")
}

func (a *Apollo) queryService(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log := a.cfg.Log.Get()
	// q := r.URL.Query()
	// appID := q.Get("appId")
	type svc struct {
		AppName     string `json:"appName"`
		InstanceId  string `json:"instanceId"`
		HomepageUrl string `json:"homepageUrl"`
	}
	type rsp []*svc
	json, err := json.Marshal(rsp{
		&svc{
			AppName:     "APOLLO-CONFIGSERVICE",
			InstanceId:  fmt.Sprintf("localhost:apollo-configservice:%d", a.cfg.Port),
			HomepageUrl: fmt.Sprintf("http://localhost:%d/", a.cfg.Port),
		},
	})
	if err != nil {
		log.Error(err.Error())
		w.WriteHeader(500)
		return
	}
	w.Write(json)
	log.Debug(fmt.Sprintf("served service for request: %s", r.URL.String()))
}

func (a *Apollo) queryConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log := a.cfg.Log.Get()
	appID := ps.ByName("appId")
	cluster := ps.ByName("cluster")
	namespace := ps.ByName("namespace")

	ns, err := a.getNamespace(appID, cluster, namespace)
	if err != nil {
		log.Warn(fmt.Sprintf("no config for request: %s", r.URL.String()))
		w.WriteHeader(404)
		return
	}
	type rsp struct {
		AppID          string            `json:"appId"`
		Cluster        string            `json:"cluster"`
		Namespace      string            `json:"namespaceName"`
		ReleaseKey     string            `json:"releaseKey"`
		Configurations map[string]string `json:"configurations"`
	}
	json, err := json.Marshal(&rsp{
		AppID:          appID,
		Cluster:        cluster,
		Namespace:      namespace,
		ReleaseKey:     ns.ReleaseKey,
		Configurations: ns.Configurations,
	})
	if err != nil {
		log.Error(err.Error())
		w.WriteHeader(500)
		return
	}
	w.Write(json)
	log.Debug(fmt.Sprintf("served config for request: %s", r.URL.String()))
}

func (a *Apollo) queryConfigJSON(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log := a.cfg.Log.Get()
	appID := ps.ByName("appId")
	cluster := ps.ByName("cluster")
	namespace := ps.ByName("namespace")

	ns, err := a.getNamespace(appID, cluster, namespace)
	if err != nil {
		log.Warn(fmt.Sprintf("no config for request: %s", r.URL.String()))
		w.WriteHeader(404)
		return
	}

	json, err := json.Marshal(ns.Configurations)
	if err != nil {
		log.Error(err.Error())
		w.WriteHeader(500)
		return
	}
	w.Write(json)
	log.Debug(fmt.Sprintf("served config for request: %s", r.URL.String()))
}

func (a *Apollo) longPolling(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	v, ok := r.URL.Query()["notifications"]
	if !ok && len(v) != 1 {
		a.cfg.Log.Get().Warn(fmt.Sprintf("invalid request: %s", r.URL.String()))
		w.WriteHeader(400)
		return
	}
	notifications := []longpoll.Notification{}
	if err := json.Unmarshal([]byte(v[0]), &notifications); err != nil {
		a.cfg.Log.Get().Error(err.Error())
		w.WriteHeader(400)
		return
	}
	if err := a.newPoll(r.Context(), notifications, w); err != nil {
		a.cfg.Log.Get().Error(err.Error())
		w.WriteHeader(500)
		return
	}
	a.cfg.Log.Get().Debug(fmt.Sprintf("served poll for request: %s", r.URL.String()))
}

func (a *Apollo) newPoll(ctx context.Context, notifications []longpoll.Notification, w http.ResponseWriter) error {
	cfg := longpoll.Config{
		Log:           a.cfg.Log,
		Notifications: notifications,
		Timeout:       a.cfg.PollTimeout,
	}
	p, err := longpoll.New(ctx, cfg, w)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.polls[p] = true
	a.mu.Unlock()

	// wait until the poll has been closed
	p.Wait()

	a.mu.Lock()
	delete(a.polls, p)
	a.mu.Unlock()

	return nil
}

func (a *Apollo) watch(ctx context.Context, filePath string) error {
	cfg := watcher.Config{
		Log:  a.cfg.Log,
		File: filePath,
	}
	w, err := watcher.New(ctx, cfg)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.UpdateEvent:
				a.mu.Lock()
				for p := range a.polls {
					if err := p.Update(); err != nil {
						a.cfg.Log.Get().Error(err.Error())
					}
				}
				a.mu.Unlock()
			}
		}
	}()
	a.w = append(a.w, w)
	return err
}
