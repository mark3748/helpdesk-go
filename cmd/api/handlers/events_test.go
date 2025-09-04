package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type dummyUser struct{ roles []string }

func (d dummyUser) GetRoles() []string { return d.roles }

func TestEvents_HeartbeatAndBacklog(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	handler := events(rdb, 5*time.Millisecond, 1)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Writer = &slowWriter{ResponseWriter: c.Writer, delay: 20 * time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/events", nil)
	c.Request = req
	c.Set("user", dummyUser{roles: []string{"agent"}})

	go handler(c)
	time.Sleep(25 * time.Millisecond)
	for i := 0; i < 5; i++ {
		PublishEvent(ctx, rdb, Event{Type: fmt.Sprintf("t%d", i)})
	}
	<-ctx.Done()
	time.Sleep(5 * time.Millisecond)

	out := w.Body.String()
	if !strings.Contains(out, ":hb") {
		t.Fatalf("missing heartbeat: %q", out)
	}
	if cnt := strings.Count(out, "event:"); cnt > 2 {
		t.Fatalf("backlog not limited, got %d events", cnt)
	}
}

type slowWriter struct {
	gin.ResponseWriter
	delay time.Duration
}

func (s *slowWriter) Write(b []byte) (int, error) {
	time.Sleep(s.delay)
	return s.ResponseWriter.Write(b)
}
