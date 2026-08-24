package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"fermentation-kinetics-deviation-analysis/backend/internal/constants"
	"fermentation-kinetics-deviation-analysis/backend/internal/middleware"
	"fermentation-kinetics-deviation-analysis/backend/internal/model"
	"fermentation-kinetics-deviation-analysis/backend/internal/repository"
	"fermentation-kinetics-deviation-analysis/backend/internal/service"
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newVesselHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.FermentationVessel{}, &model.CultureRecipe{},
		&model.SensorSeries{}, &model.DeviationAnalysis{}, &model.AuditLog{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestUpdateVesselHTTPRejectsInactive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newVesselHandlerDB(t)
	vesselRepo := repository.NewFermentationVesselRepository(db)
	svc := service.NewFermentationVesselService(vesselRepo, repository.NewAuditRepository(db))
	actor := util.Actor{UserID: 1, Username: "scientist", Role: string(constants.RoleProcessScientist), RequestID: "req-007h"}
	now := time.Now().UTC()
	vessel := model.FermentationVessel{
		VesselCode: "FV-HTTP", Name: "HTTP vessel", WorkingVolumeL: 100,
		SensorChannels: `["ph"]`, Location: "Lab", OwnerTeam: "Process",
		VesselState: "active", CommissionedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := vesselRepo.Create(context.Background(), &vessel); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Deactivate(context.Background(), vessel.ID, actor); err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("actor", actor); c.Next() })
	engine.PUT("/api/v1/vessels/:id", middleware.RequirePermission(constants.PermissionVesselWrite),
		NewFermentationVesselHandler(svc).Update)
	body := strings.NewReader(`{"name":"edited-after-inactive"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/vessels/"+strconv.FormatUint(uint64(vessel.ID), 10), body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("update of an inactive vessel over HTTP status = %d, want 409", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "inactive") {
		t.Fatalf("expected a conflict message about the inactive vessel, body: %s", recorder.Body.String())
	}
}
