package watcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/lalamove/nui/nlogger"

	"github.com/radovskyb/watcher"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

// Namespace holds the namespace config
type Namespace struct {
	ReleaseKey     string            `yaml:"releaseKey" json:"releaseKey"`
	Configurations map[string]string `yaml:"configurations" json:"configurations"`
}

// ConfigMap holds the app config
type ConfigMap map[string]map[string]map[string]Namespace

// Config holds the watcher config
type Config struct {
	Log           nlogger.Provider
	File          string
	WatchInterval time.Duration
}

// Watcher holds information for the watcher
type Watcher struct {
	fs          afero.Fs
	fw          *watcher.Watcher
	cm          atomic.Value
	filePath    string
	UpdateEvent <-chan struct{}
}

// New returns a new Watcher
func New(ctx context.Context, cfg Config) (*Watcher, error) {
	validateConfig(&cfg)
	fw := watcher.New()
	if err := fw.Add(cfg.File); err != nil {
		return nil, err
	}
	if len(fw.WatchedFiles()) != 1 {
		return nil, fmt.Errorf("got an invalid file path to watch: %s", cfg.File)
	}
	updateChan := make(chan struct{})
	w := &Watcher{
		fs:          afero.NewOsFs(),
		fw:          fw,
		UpdateEvent: updateChan,
	}
	for path := range fw.WatchedFiles() {
		w.filePath = path
	}
	go func() {
		for {
			select {
			case <-fw.Closed:
				cfg.Log.Get().Debug("watcher is closed")
				return
			case <-ctx.Done():
				cfg.Log.Get().Debug("ctx was cancelled, stopping watcher")
				fw.Close()
				return
			case event := <-fw.Event:
				cfg.Log.Get().Debug(fmt.Sprintf("watcher received event: %s", event))
				if err := w.readConfigMap(); err != nil {
					cfg.Log.Get().Error(fmt.Sprintf("error reading file: %v", err))
				} else {
					updateChan <- struct{}{}
					cfg.Log.Get().Info("watcher loaded new config")
				}
			case err := <-fw.Error:
				cfg.Log.Get().Error(fmt.Sprintf("watcher received error: %v", err))
			}
		}
	}()

	go func() {
		cfg.Log.Get().Info(fmt.Sprintf("started watching file: %s", w.filePath))
		if err := fw.Start(cfg.WatchInterval); err != nil {
			cfg.Log.Get().Error(fmt.Sprintf("error starting watcher: %v", err))
			return
		}
	}()

	err := w.readConfigMap()
	return w, err
}

func validateConfig(cfg *Config) {
	if cfg.WatchInterval < time.Second {
		cfg.WatchInterval = time.Second
	}
	if cfg.Log == nil {
		cfg.Log = nlogger.NewProvider(nlogger.New(os.Stdout, ""))
	}
}

// MockFS injects mocked fs into Watcher
// this should only be called immediately after watcher is initialized
// since it's not a thread safe operation
func (w *Watcher) MockFS(fs afero.Fs) {
	w.fs = fs
	return
}

// ReloadConfig reloads file config without senging an update event
func (w *Watcher) ReloadConfig() error {
	return w.readConfigMap()
}

// TriggerEvent triggers the update event
func (w *Watcher) TriggerEvent() {
	w.fw.TriggerEvent(watcher.Write, nil)
}

func (w *Watcher) readConfigMap() error {
	b, err := afero.ReadFile(w.fs, w.filePath)
	if err != nil {
		return err
	}
	cm := ConfigMap{}
	err = yaml.Unmarshal(b, &cm)
	if err != nil {
		return err
	}
	// validate configuration
	if len(cm) == 0 {
		return errors.New("invalid config file")
	}
	for appKey, app := range cm {
		if appKey == "" {
			return fmt.Errorf("invalid app name '%s'", appKey)
		}
		if len(app) == 0 {
			return fmt.Errorf("invalid app '%s'", appKey)
		}
		for clusterKey, cluster := range app {
			if clusterKey == "" {
				return fmt.Errorf("invalid cluster name '%s' in %s", clusterKey, appKey)
			}
			if len(cluster) == 0 {
				return fmt.Errorf("invalid cluster '%s' in %s", clusterKey, appKey)
			}
			for nsKey, ns := range cluster {
				if nsKey == "" {
					return fmt.Errorf("invalid namespace name '%s' in %s/%s", nsKey, appKey, clusterKey)
				}
				if ns.Configurations == nil {
					return fmt.Errorf("invalid namespace '%s' in %s/%s", nsKey, appKey, clusterKey)
				}
				for configKey := range ns.Configurations {
					if configKey == "" {
						return fmt.Errorf("invalid config key '%s' in %s/%s/%s", configKey, appKey, clusterKey, nsKey)
					}
				}
			}
		}
	}
	w.cm.Store(cm)
	return nil
}

// Config returns a stored read-only ConfigMap
func (w *Watcher) Config() ConfigMap {
	return w.cm.Load().(ConfigMap)
}
