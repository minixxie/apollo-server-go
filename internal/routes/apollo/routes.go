package apollo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/lalamove/mock-apollo-go/pkg/longpoll"
	"github.com/lalamove/mock-apollo-go/pkg/watcher"
	"github.com/lalamove/nui/nlogger"
)

// Config is an object that stores the package config
type Config struct {
	Log         nlogger.Provider
	ConfigPath  string
	PollTimeout time.Duration
}

// Apollo serves the mock apollo http routes
type Apollo struct {
	mu    sync.Mutex
	cfg   Config
	w     *watcher.Watcher
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
	if err := a.watch(ctx); err != nil {
		return a, err
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
	r.GET("/healthz", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(200)
	})
	r.GET("/configs/:appId/:cluster/:namespace", a.queryConfig)
	r.GET("/configfiles/json/:appId/:cluster/:namespace", a.queryConfigJSON)
	r.GET("/notifications/v2", a.notificationsLongPolling)
}

func (a *Apollo) queryConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	appID := ps.ByName("appId")
	cluster := ps.ByName("cluster")
	namespace := ps.ByName("namespace")

	cm := a.w.Config()
	ns, ok := cm[appID][cluster][namespace]
	if !ok {
		a.cfg.Log.Get().Warn(fmt.Sprintf("no config for request: %s", r.URL.String()))
		w.WriteHeader(404)
		return
	}
	if v, ok := r.URL.Query()["releaseKey"]; ok && len(v) == 1 {
		if ns.ReleaseKey != v[0] {
			a.cfg.Log.Get().Warn(fmt.Sprintf("no config for request with releaseKey: %s", r.URL.String()))
			w.WriteHeader(404)
			return
		}
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
		a.cfg.Log.Get().Error(err.Error())
		w.WriteHeader(500)
		return
	}
	w.Write(json)
}

func (a *Apollo) queryConfigJSON(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	appID := ps.ByName("appId")
	cluster := ps.ByName("cluster")
	namespace := ps.ByName("namespace")

	cm := a.w.Config()

	ns, ok := cm[appID][cluster][namespace]
	if !ok {
		a.cfg.Log.Get().Warn(fmt.Sprintf("no config for request: %s", r.URL.String()))
		w.WriteHeader(404)
		return
	}

	json, err := json.Marshal(ns.Configurations)
	if err != nil {
		a.cfg.Log.Get().Error(err.Error())
		w.WriteHeader(500)
		return
	}
	w.Write(json)
}

func (a *Apollo) notificationsLongPolling(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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

func (a *Apollo) watch(ctx context.Context) error {
	cfg := watcher.Config{
		Log:  a.cfg.Log,
		File: a.cfg.ConfigPath,
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
	a.w = w
	return err
}
