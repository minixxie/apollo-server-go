package apollo

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/lalamove/mock-apollo-go/pkg/watcher"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

var stubConfigs = []watcher.ConfigMap{
	map[string]map[string]map[string]watcher.Namespace{
		"app": {
			"cluster": {
				"ns": {
					ReleaseKey: "abc",
					Configurations: map[string]string{
						"mysql": "mysql://root@localhost/mysql",
					},
				},
			},
		},
	},
}

func TestQueryService(t *testing.T) {

}

func TestQueryConfig(t *testing.T) {
	// mock fs
	appFS := afero.NewMemMapFs()
	appFS.MkdirAll("/dev", 0755)
	data, err := yaml.Marshal(stubConfigs[0])
	require.Nil(t, err)
	require.Nil(t, afero.WriteFile(appFS, "/dev/null", data, 0644))

	// setup apollo
	filepaths := []string{"/dev/null"}
	a, err := New(context.Background(), Config{ConfigPath: filepaths})
	require.EqualError(t, err, "invalid config file")
	for _, w := range a.w {
		w.MockFS(appFS)
		require.Nil(t, w.ReloadConfig())
	}

	t.Run("status 200", func(t *testing.T) {
		// call the handler
		req := httptest.NewRequest("GET", "/configs/app/cluster/ns", nil)
		w := httptest.NewRecorder()
		ps := httprouter.Params{
			httprouter.Param{Key: "appId", Value: "app"},
			httprouter.Param{Key: "cluster", Value: "cluster"},
			httprouter.Param{Key: "namespace", Value: "ns"},
		}
		a.queryConfig(w, req, ps)
		rsp := w.Result()
		require.Equal(t, 200, rsp.StatusCode)
		b, err := ioutil.ReadAll(rsp.Body)
		require.Nil(t, err)
		require.JSONEq(
			t,
			`{"appId":"app","cluster":"cluster","namespaceName":"ns","releaseKey":"abc","configurations":{"mysql":"mysql://root@localhost/mysql"}}`,
			string(b),
			string(b),
		)
	})

	t.Run("status 200 - with releaseKey", func(t *testing.T) {
		// call the handler
		req := httptest.NewRequest("GET", "/configs/app/cluster/ns?releaseKey=abc", nil)
		w := httptest.NewRecorder()
		ps := httprouter.Params{
			httprouter.Param{Key: "appId", Value: "app"},
			httprouter.Param{Key: "cluster", Value: "cluster"},
			httprouter.Param{Key: "namespace", Value: "ns"},
		}
		a.queryConfig(w, req, ps)
		rsp := w.Result()
		require.Equal(t, 200, rsp.StatusCode)
		b, err := ioutil.ReadAll(rsp.Body)
		require.Nil(t, err)
		require.JSONEq(
			t,
			`{"appId":"app","cluster":"cluster","namespaceName":"ns","releaseKey":"abc","configurations":{"mysql":"mysql://root@localhost/mysql"}}`,
			string(b),
			string(b),
		)
	})

	t.Run("status 404 - namespace", func(t *testing.T) {
		// call the handler
		req := httptest.NewRequest("GET", "/configs/app/cluster/ns404", nil)
		w := httptest.NewRecorder()
		ps := httprouter.Params{
			httprouter.Param{Key: "appId", Value: "app"},
			httprouter.Param{Key: "cluster", Value: "cluster"},
			httprouter.Param{Key: "namespace", Value: "ns404"},
		}
		a.queryConfig(w, req, ps)
		rsp := w.Result()
		require.Equal(t, 404, rsp.StatusCode)
		b, err := ioutil.ReadAll(rsp.Body)
		require.Nil(t, err)
		require.Equal(t, "", string(b))
	})
}

func TestQueryConfigJSON(t *testing.T) {
	// mock fs
	appFS := afero.NewMemMapFs()
	appFS.MkdirAll("/dev", 0755)
	data, err := yaml.Marshal(stubConfigs[0])
	require.Nil(t, err)
	require.Nil(t, afero.WriteFile(appFS, "/dev/null", data, 0644))

	// setup apollo
	filepaths := []string{"/dev/null"}
	a, err := New(context.Background(), Config{ConfigPath: filepaths})
	require.EqualError(t, err, "invalid config file")
	for _, w := range a.w {
		w.MockFS(appFS)
		require.Nil(t, w.ReloadConfig())
	}

	t.Run("status 200", func(t *testing.T) {
		// call the handler
		req := httptest.NewRequest("GET", "/configfiles/json/app/cluster/ns", nil)
		w := httptest.NewRecorder()
		ps := httprouter.Params{
			httprouter.Param{Key: "appId", Value: "app"},
			httprouter.Param{Key: "cluster", Value: "cluster"},
			httprouter.Param{Key: "namespace", Value: "ns"},
		}
		a.queryConfigJSON(w, req, ps)
		rsp := w.Result()
		require.Equal(t, 200, rsp.StatusCode)
		b, err := ioutil.ReadAll(rsp.Body)
		require.Nil(t, err)
		require.JSONEq(
			t,
			`{"mysql":"mysql://root@localhost/mysql"}`,
			string(b),
			string(b),
		)
	})
	t.Run("status 404", func(t *testing.T) {
		// call the handler
		req := httptest.NewRequest("GET", "/configfiles/json/app/cluster/ns404", nil)
		w := httptest.NewRecorder()
		ps := httprouter.Params{
			httprouter.Param{Key: "appId", Value: "app"},
			httprouter.Param{Key: "cluster", Value: "cluster"},
			httprouter.Param{Key: "namespace", Value: "ns404"},
		}
		a.queryConfigJSON(w, req, ps)
		rsp := w.Result()
		require.Equal(t, 404, rsp.StatusCode)
		b, err := ioutil.ReadAll(rsp.Body)
		require.Nil(t, err)
		require.Equal(t, "", string(b))
	})
}

func TestNotificationsLongPolling(t *testing.T) {
	t.Run("change", func(t *testing.T) {
		// setup apollo
		filepaths := []string{"/dev/null"}
		a, err := New(context.Background(), Config{ConfigPath: filepaths})
		require.Error(t, err)

		// mock fs
		appFS := afero.NewMemMapFs()
		appFS.MkdirAll("/dev", 0755)
		data, err := yaml.Marshal(stubConfigs[0])
		require.Nil(t, err)
		require.Nil(t, afero.WriteFile(appFS, "/dev/null", data, 0644))
		for _, w := range a.w {
			w.MockFS(appFS)
		}

		// call the handler
		q := "?notifications=" + url.QueryEscape(`[{"notificationId":1,"namespaceName":"ns"}]`)
		req := httptest.NewRequest("GET", "/notifications/v2"+q, nil)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		req = req.Clone(ctx)
		w := httptest.NewRecorder()
		ps := httprouter.Params{
			httprouter.Param{Key: "appId", Value: "app"},
			httprouter.Param{Key: "cluster", Value: "cluster"},
			httprouter.Param{Key: "namespace", Value: "ns"},
		}
		go func() {
			// trigger config update in the background
			time.Sleep(5 * time.Millisecond)
			for _, w := range a.w {
				w.TriggerEvent()
			}
		}()
		a.longPolling(w, req, ps)
		rsp := w.Result()
		require.Equal(t, 200, rsp.StatusCode)
		b, err := ioutil.ReadAll(rsp.Body)
		require.Nil(t, err)
		require.JSONEq(
			t,
			`[{"notificationId":1,"namespaceName":"ns"}]`,
			string(b),
			string(b),
		)
	})

	t.Run("no change", func(t *testing.T) {
		// mock fs
		appFS := afero.NewMemMapFs()
		appFS.MkdirAll("/dev", 0755)

		// setup apollo
		filepaths := []string{"/dev/null"}
		a, err := New(context.Background(), Config{
			ConfigPath:  filepaths,
			PollTimeout: time.Second,
		})
		require.Error(t, err)
		for _, w := range a.w {
			w.MockFS(appFS)
		}

		// call the handler
		q := "?notifications=" + url.QueryEscape(`[{"notificationId":1,"namespaceName":"ns"}]`)
		req := httptest.NewRequest("GET", "/notifications/v2"+q, nil)
		w := httptest.NewRecorder()
		ps := httprouter.Params{
			httprouter.Param{Key: "appId", Value: "app"},
			httprouter.Param{Key: "cluster", Value: "cluster"},
			httprouter.Param{Key: "namespace", Value: "ns"},
		}
		a.longPolling(w, req, ps)
		rsp := w.Result()
		require.Equal(t, 304, rsp.StatusCode)
		b, err := ioutil.ReadAll(rsp.Body)
		require.Nil(t, err)
		require.Equal(t, "", string(b))
	})

	t.Run("context cancelled", func(t *testing.T) {
		// mock fs
		appFS := afero.NewMemMapFs()
		appFS.MkdirAll("/dev", 0755)

		// setup apollo
		filepaths := []string{"/dev/null"}
		a, err := New(context.Background(), Config{ConfigPath: filepaths})
		require.Error(t, err)
		for _, w := range a.w {
			w.MockFS(appFS)
		}

		// call the handler
		q := "?notifications=" + url.QueryEscape(`[{"notificationId":1,"namespaceName":"ns"}]`)
		req := httptest.NewRequest("GET", "/notifications/v2"+q, nil)
		ctx, cancel := context.WithCancel(context.Background())
		req = req.Clone(ctx)
		w := httptest.NewRecorder()
		ps := httprouter.Params{
			httprouter.Param{Key: "appId", Value: "app"},
			httprouter.Param{Key: "cluster", Value: "cluster"},
			httprouter.Param{Key: "namespace", Value: "ns"},
		}
		go func() {
			// trigger context cancel
			time.Sleep(5 * time.Millisecond)
			cancel()
		}()
		a.longPolling(w, req, ps)
		rsp := w.Result()
		require.Equal(t, 304, rsp.StatusCode)
		b, err := ioutil.ReadAll(rsp.Body)
		require.Nil(t, err)
		require.Equal(t, "", string(b))
	})
}
