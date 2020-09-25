package watcher

import (
	"context"
	"testing"
	"time"

	"github.com/radovskyb/watcher"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

var stubConfigs = []ConfigMap{
	map[string]map[string]map[string]Namespace{
		"myApp": {
			"myCluster": {
				"myNamespace": {
					ReleaseKey: "abc",
					Configurations: map[string]string{
						"mysql.uri":     "mysql://root@localhost/mysql",
						"snowflake.uri": "http://192.168.0.1/snowflake",
					},
				},
			},
		},
	},
	map[string]map[string]map[string]Namespace{
		"myApp": {
			"myCluster": {
				"myNamespace": {
					ReleaseKey: "abc",
					Configurations: map[string]string{
						"mysql.uri":     "mysql://root@localhost/mysql2",
						"snowflake.uri": "http://192.168.0.1/snowflake",
					},
				},
			},
		},
	},
}

func triggerWriteEvent(ctx context.Context, t *testing.T, w *Watcher) {
	w.fw.TriggerEvent(watcher.Write, nil)
	select {
	case <-ctx.Done():
		require.Fail(t, "context cancelled")
	case <-w.UpdateEvent:
	}
}
func TestWatcher(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	// mock fs
	appFS := afero.NewMemMapFs()
	appFS.MkdirAll("/dev", 0755)

	// write initial config
	data, err := yaml.Marshal(stubConfigs[0])
	require.Nil(t, err)
	require.Nil(t, afero.WriteFile(appFS, "/dev/null", data, 0644))

	// mock fs and load the stubbed config
	w, _ := New(ctx, Config{File: "/dev/null"})
	MockFS(w, appFS)
	triggerWriteEvent(ctx, t, w)
	// verify config values
	require.EqualValues(t, stubConfigs[0], w.Config())

	// update the initial config
	data, err = yaml.Marshal(stubConfigs[1])
	require.Nil(t, err)
	require.Nil(t, afero.WriteFile(appFS, "/dev/null", data, 0644))
	triggerWriteEvent(ctx, t, w)
	// verify config values
	require.EqualValues(t, stubConfigs[1], w.Config())
}

func TestReadConfigMap(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	// mock fs
	appFS := afero.NewMemMapFs()
	appFS.MkdirAll("/dev", 0755)

	w, err := New(ctx, Config{File: "/dev/null"})
	require.EqualError(t, err, "invalid config file")
	MockFS(w, appFS)

	t.Run("file not exist", func(t *testing.T) {
		require.EqualError(t, w.readConfigMap(), "open /dev/null: file does not exist")
	})
	t.Run("empty config", func(t *testing.T) {
		require.Nil(t, afero.WriteFile(appFS, "/dev/null", []byte(""), 0644))
		require.EqualError(t, err, "invalid config file")
		require.EqualError(t, w.readConfigMap(), "invalid config file")
	})
	var testMatrix = []struct {
		name        string
		expectedErr string
		configMap   string
	}{
		{
			name:        "empty app ID",
			expectedErr: "invalid app name ''",
			configMap: `"":
            myCluster:
              myNamespace:
                releaseKey: "20200309212653-7fec91b6d277b5ab"
                configurations:
                  snowflake.uri: "http://192.168.0.1/snowflake"`,
		},
		{
			name:        "empty app ID",
			expectedErr: "invalid app 'myApp'",
			configMap:   `myApp: {}`,
		},
		{
			name:        "empty cluster name",
			expectedErr: "invalid cluster name '' in myApp",
			configMap: `myApp:
            "":
              myNamespace:
                releaseKey: "20200309212653-7fec91b6d277b5ab"
                configurations:
                  snowflake.uri: "http://192.168.0.1/snowflake"`,
		},
		{
			name:        "empty cluster name",
			expectedErr: "invalid cluster 'myCluster' in myApp",
			configMap: `myApp:
            myCluster: {}`,
		},
		{
			name:        "empty namespaceName",
			expectedErr: "invalid namespace name '' in myApp/myCluster",
			configMap: `myApp:
            myCluster:
              "":
                releaseKey: "20200309212653-7fec91b6d277b5ab"
                configurations:
                  snowflake.uri: "http://192.168.0.1/snowflake"`,
		},
		{
			name:        "empty namespaceName",
			expectedErr: "invalid namespace 'myNamespace' in myApp/myCluster",
			configMap: `myApp:
            myCluster:
              myNamespace: {}`,
		},
		{
			name:        "empty releaseKey",
			expectedErr: "invalid releaseKey '' in myApp/myCluster/myNamespace",
			configMap: `myApp:
            myCluster:
              myNamespace:
                releaseKey: ""
                configurations:
                  snowflake.uri: "http://192.168.0.1/snowflake"`,
		},
		{
			name:        "empty config key",
			expectedErr: "invalid config key '' in myApp/myCluster/myNamespace",
			configMap: `myApp:
            myCluster:
              myNamespace:
                releaseKey: "20200309212653-7fec91b6d277b5ab"
                configurations:
                  "": "mysql://root@localhost/mysql"
                  snowflake.uri: "http://192.168.0.1/snowflake"`,
		},
	}

	for _, test := range testMatrix {
		t.Run(test.name, func(t *testing.T) {
			require.Nil(t, afero.WriteFile(appFS, "/dev/null", []byte(test.configMap), 0644))
			require.EqualError(t, w.readConfigMap(), test.expectedErr)
		})
	}
}
