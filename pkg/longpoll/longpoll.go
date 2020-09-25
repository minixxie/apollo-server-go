package longpoll

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/lalamove/nui/nlogger"
)

// Config is an object that stores the poll config
type Config struct {
	Log           nlogger.Provider
	Notifications []Notification
	Timeout       time.Duration
}

// Notification contains Namespace and ID that is sent to the client on an update
type Notification struct {
	ID        int    `json:"notificationId"`
	Namespace string `json:"namespaceName"`
}

// Poll provides long polling functionality with an ability to notify the client at most once
type Poll struct {
	mu      sync.Mutex
	ctx     context.Context
	updated bool
	ns      []Notification
	c       chan<- struct{}
}

// New creates a new long Poll
func New(ctx context.Context, cfg Config, w http.ResponseWriter) (*Poll, error) {
	validateConfig(&cfg)
	c := make(chan struct{}, 1)
	// pollCtx is used to check whether the poll is still open
	// it it guaranteed to be done only after the input ctx is closed
	// and a response has been written to w
	pollCtx, cancel := context.WithCancel(context.Background())
	done := time.After(cfg.Timeout)
	p := &Poll{
		ctx:     pollCtx,
		updated: false,
		ns:      cfg.Notifications,
		c:       c,
	}
	go func() {
		defer cancel()
		select {
		case <-ctx.Done():
			cfg.Log.Get().Debug("poll context was cancelled, stoped watching for a change")
			w.WriteHeader(304)
		case <-done:
			cfg.Log.Get().Debug("poll timed out with no updates")
			w.WriteHeader(304)
		case <-c:
			cfg.Log.Get().Info("poll received a change notification")
			res, _ := json.Marshal(p.ns)
			_, err := w.Write(res)
			if err != nil {
				cfg.Log.Get().Error(fmt.Sprintf("error writing poll rsp: %v\n", err))
			}
		}
	}()
	return p, nil
}

func validateConfig(cfg *Config) {
	if cfg.Timeout == 0 {
		cfg.Timeout = time.Minute
	}
	if cfg.Log == nil {
		cfg.Log = nlogger.NewProvider(nlogger.New(os.Stdout, ""))
	}
}

// Wait waits until poll is closed
func (p *Poll) Wait() {
	<-p.ctx.Done()
}

// Update notifies the client of a version change
func (p *Poll) Update() error {
	// mutex guarantees that multiple concurrent calls to Update func will be handled gracefully
	p.mu.Lock()
	defer p.mu.Unlock()
	select {
	case <-p.ctx.Done():
		return errors.New("poll is closed")
	default:
		if p.updated {
			return errors.New("poll has already been updated")
		}
	}

	p.c <- struct{}{}
	p.updated = true
	return nil
}
