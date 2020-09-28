package longpoll

import (
	"context"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPoll(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx := context.Background()
		poll, err := New(ctx, Config{Notifications: []Notification{{1, "test"}}, Timeout: time.Second}, recorder)
		require.Nil(t, err)
		require.Nil(t, poll.Update())

		poll.Wait()

		res := recorder.Result()
		b, err := ioutil.ReadAll(res.Body)
		require.Nil(t, err)
		require.Equal(t, 200, res.StatusCode)
		require.JSONEq(t, `[{"namespaceName": "test","notificationId": 1}]`, string(b))

		// no further updates should be accepted now
		require.Error(t, poll.Update())
	})

	t.Run("no change", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx := context.Background()
		poll, err := New(ctx, Config{Notifications: []Notification{{1, "test"}}, Timeout: time.Millisecond}, recorder)
		require.Nil(t, err)

		poll.Wait()

		res := recorder.Result()
		b, err := ioutil.ReadAll(res.Body)
		require.Nil(t, err)
		require.Equal(t, 304, res.StatusCode)
		require.Equal(t, "", string(b))
	})

	t.Run("client canceled", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		ctx, cancel := context.WithCancel(context.Background())
		poll, err := New(ctx, Config{Notifications: []Notification{{1, "test"}}, Timeout: time.Second}, recorder)
		require.Nil(t, err)
		// mock cancel from the client
		cancel()

		poll.Wait()

		res := recorder.Result()
		b, err := ioutil.ReadAll(res.Body)
		require.Nil(t, err)
		require.Equal(t, 304, res.StatusCode)
		require.Equal(t, "", string(b))
	})
}
