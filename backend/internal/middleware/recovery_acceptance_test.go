package middleware

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fermentation-kinetics-deviation-analysis/backend/internal/util"
	"github.com/gin-gonic/gin"
)

func captureLogger() (*bytes.Buffer, *slog.Logger) {
	var buf bytes.Buffer
	return &buf, slog.New(slog.NewTextHandler(&buf, nil))
}

func TestPanicLoggedWithStack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	buf, logger := captureLogger()
	engine := gin.New()
	engine.Use(ErrorHandler(logger))
	engine.Use(Recovery(logger))
	engine.GET("/api/v1/crash", func(c *gin.Context) {
		panic("sensor read failed")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/crash", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	out := buf.String()
	if !strings.Contains(out, "panic recovered") {
		t.Fatalf("expected panic recovery to be logged, got: %s", out)
	}
	if !strings.Contains(out, "sensor read failed") {
		t.Fatalf("expected the panic detail in the log, got: %s", out)
	}
	if !strings.Contains(out, "goroutine") {
		t.Fatalf("expected the panic stack trace in the log, got: %s", out)
	}
}

func TestPanicAttachedToErrorChain(t *testing.T) {
	gin.SetMode(gin.TestMode)
	buf, logger := captureLogger()
	engine := gin.New()
	engine.Use(ErrorHandler(logger))
	engine.Use(Recovery(logger))
	engine.GET("/api/v1/crash", func(c *gin.Context) {
		panic("sensor read failed")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/crash", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	out := buf.String()
	if !strings.Contains(out, "unclassified request error") {
		t.Fatalf("the panic must surface in the error handler chain log, got: %s", out)
	}
}

func TestErrorHandlerLogsMultipleErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	buf, logger := captureLogger()
	engine := gin.New()
	engine.Use(ErrorHandler(logger))
	engine.GET("/api/v1/multi", func(c *gin.Context) {
		c.Error(errors.New("first failure"))
		c.Error(errors.New("second failure"))
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/multi", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	out := buf.String()
	if !strings.Contains(out, "first failure") || !strings.Contains(out, "second failure") {
		t.Fatalf("expected every error in the chain to be logged, got: %s", out)
	}
}

func TestErrorHandlerSummarizesMultipleErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	buf, logger := captureLogger()
	engine := gin.New()
	engine.Use(ErrorHandler(logger))
	engine.GET("/api/v1/multi", func(c *gin.Context) {
		c.Error(util.NewError(http.StatusBadRequest, util.CodeBadRequest, "first failure"))
		c.Error(util.NewError(http.StatusConflict, util.CodeConflict, "second failure"))
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/multi", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	out := buf.String()
	if !strings.Contains(out, "request accumulated multiple errors") {
		t.Fatalf("expected a summary entry for the accumulated error chain, got: %s", out)
	}
}
