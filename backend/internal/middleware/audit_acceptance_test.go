package middleware

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"fermentation-kinetics-deviation-analysis/backend/internal/model"
	"fermentation-kinetics-deviation-analysis/backend/internal/repository"
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func auditTestActor() util.Actor {
	return util.Actor{UserID: 7, Username: "scientist", Role: "process_scientist", RequestID: "req-audit-006"}
}

func TestAuditWriteWithCancelledContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newAuditTestDB(t)
	engine := gin.New()
	engine.Use(Audit(repository.NewAuditRepository(db)))
	engine.POST("/api/v1/writes/:id", func(c *gin.Context) {
		c.Set("actor", auditTestActor())
		ctx, cancel := context.WithCancel(c.Request.Context())
		cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/writes/42", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("handler status = %d, want 200", recorder.Code)
	}
	var total int64
	if err := db.Model(&model.AuditLog{}).Count(&total).Error; err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("audit rows = %d, want 1: the write audit must survive a cancelled client context", total)
	}
}

type failingAuditRepo struct{}

func (failingAuditRepo) Record(context.Context, model.AuditLog) error {
	return errors.New("audit storage is down")
}
func (failingAuditRepo) List(context.Context, repository.AuditQuery) ([]model.AuditLog, int64, error) {
	return nil, 0, nil
}

func TestAuditWriteFailureIsLogged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	engine := gin.New()
	engine.Use(ErrorHandler(logger))
	engine.Use(Audit(failingAuditRepo{}))
	engine.POST("/api/v1/writes/:id", func(c *gin.Context) {
		c.Set("actor", auditTestActor())
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/writes/42", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if !strings.Contains(buf.String(), "HTTP write audit failed") {
		t.Fatalf("expected audit failure to surface in the error log, got: %s", buf.String())
	}
}
