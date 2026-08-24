package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fermentation-kinetics-deviation-analysis/backend/internal/util"
	"github.com/gin-gonic/gin"
)

func TestRateLimiterConcurrentAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewRateLimiter(50)
	middleware := limiter.Middleware("login")
	start := make(chan struct{})
	var wg sync.WaitGroup
	var allowed atomic.Int32
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for j := 0; j < 300; j++ {
				recorder := httptest.NewRecorder()
				context, _ := gin.CreateTestContext(recorder)
				context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
				context.Request.RemoteAddr = fmt.Sprintf("10.0.0.%d:1234", worker)
				context.Set("actor", util.Actor{
					Username: "shared-user", Role: "data_analyst",
					RequestID: fmt.Sprintf("req-%d-%d", worker, j),
				})
				middleware(context)
				if recorder.Code == http.StatusOK {
					allowed.Add(1)
				}
			}
		}(i)
	}
	close(start)
	wg.Wait()
	if allowed.Load() > 50 {
		t.Fatalf("rate limit bypassed: %d allowed, cap 50", allowed.Load())
	}
}

func TestRateLimiterSweepConcurrent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limiter := NewRateLimiter(100)
	base := time.Now()
	for i := 0; i < 10100; i++ {
		limiter.windows[fmt.Sprintf("stale-%d", i)] = rateWindow{start: base.Add(-3 * time.Minute)}
	}
	middleware := limiter.Middleware("series-import")
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for j := 0; j < 60; j++ {
				recorder := httptest.NewRecorder()
				context, _ := gin.CreateTestContext(recorder)
				context.Request = httptest.NewRequest(http.MethodGet, "/", nil)
				context.Request.RemoteAddr = fmt.Sprintf("10.0.1.%d:1", worker)
				context.Set("actor", util.Actor{
					Username: fmt.Sprintf("worker-%d-%d", worker, j), Role: "data_analyst",
					RequestID: fmt.Sprintf("sweep-%d-%d", worker, j),
				})
				middleware(context)
			}
		}(i)
	}
	close(start)
	wg.Wait()
}
