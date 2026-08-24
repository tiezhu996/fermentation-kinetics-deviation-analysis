package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"fermentation-kinetics-deviation-analysis/backend/internal/algorithm"
	"fermentation-kinetics-deviation-analysis/backend/internal/constants"
	"fermentation-kinetics-deviation-analysis/backend/internal/dto"
	"fermentation-kinetics-deviation-analysis/backend/internal/middleware"
	"fermentation-kinetics-deviation-analysis/backend/internal/model"
	"fermentation-kinetics-deviation-analysis/backend/internal/repository"
	"fermentation-kinetics-deviation-analysis/backend/internal/service"
	"fermentation-kinetics-deviation-analysis/backend/internal/timeseries"
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPermissionTestDB(t *testing.T) *gorm.DB {
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

type permissionFixture struct {
	handler    *DeviationAnalysisHandler
	analysisID uint
}

func newReviewedAnalysis(t *testing.T) *permissionFixture {
	t.Helper()
	db := newPermissionTestDB(t)
	vesselRepo := repository.NewFermentationVesselRepository(db)
	recipeRepo := repository.NewCultureRecipeRepository(db)
	seriesRepo := repository.NewSensorSeriesRepository(db)
	analysisRepo := repository.NewDeviationAnalysisRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	vessel := model.FermentationVessel{
		VesselCode: "FV-P1", Name: "Permission vessel", WorkingVolumeL: 100,
		SensorChannels: `["ph"]`, Location: "Lab", OwnerTeam: "Process",
		VesselState: "active", CommissionedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := vesselRepo.Create(context.Background(), &vessel); err != nil {
		t.Fatal(err)
	}
	recipe := model.CultureRecipe{
		VesselID: vessel.ID, RecipeCode: "PERM-A", Version: 1, Organism: "Test organism",
		TargetDurationH: 8, PhaseBoundariesJSON: `[{"phase":"lag","start_hour":0,"end_hour":2},{"phase":"growth","start_hour":2,"end_hour":4},{"phase":"production","start_hour":4,"end_hour":6},{"phase":"harvest","start_hour":6,"end_hour":8}]`,
		ReferenceCurvesJSON: `{"ph":[{"elapsed_h":0,"value":7},{"elapsed_h":2,"value":6.9},{"elapsed_h":4,"value":6.8},{"elapsed_h":6,"value":6.7},{"elapsed_h":8,"value":6.6}]}`,
		ToleranceProfileJSON: `{"ph":{"weight":1,"max_distance":1}}`, RecipeState: "published",
		CreatedBy: 8, CreatedByName: "scientist", CreatedAt: now, UpdatedAt: now,
	}
	if err := recipeRepo.Create(context.Background(), &recipe); err != nil {
		t.Fatal(err)
	}
	points := make([]timeseries.Point, 0, 9)
	for hour := 0; hour <= 8; hour++ {
		value := 7 - float64(hour)*0.05
		valueCopy := value
		points = append(points, timeseries.Point{
			Timestamp: now.Add(time.Duration(hour) * time.Hour), Values: map[string]*float64{"ph": &valueCopy},
		})
	}
	pointsJSON, err := timeseries.EncodePoints(points)
	if err != nil {
		t.Fatal(err)
	}
	series := model.SensorSeries{
		VesselID: vessel.ID, RecipeID: recipe.ID, RunCode: "RUN-P1", Channel: "ph",
		SampleIntervalS: 3600, PointsJSON: pointsJSON, StartedAt: now, EndedAt: now.Add(8 * time.Hour),
		SourceChecksum: util.HashString(pointsJSON), SeriesState: "ready", QualitySummary: `{"valid":true}`,
		NormalizationJSON: `{"method":"median_iqr"}`, ImportedBy: 9, ImportedByName: "analyst",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := seriesRepo.Create(context.Background(), &series); err != nil {
		t.Fatal(err)
	}
	svc := service.NewDeviationAnalysisService(analysisRepo, recipeRepo, seriesRepo, auditRepo, algorithm.NewEvaluator())
	initiator := util.Actor{UserID: 9, Username: "analyst", Role: "data_analyst", RequestID: "req-008-init"}
	created, reused, err := svc.Run(context.Background(), dto.RunDeviationAnalysisRequest{SensorSeriesID: series.ID}, "idem-008", initiator)
	if err != nil || reused {
		t.Fatalf("run analysis: reused=%v err=%v", reused, err)
	}
	reviewer := util.Actor{UserID: 10, Username: "reviewer", Role: "reviewer", RequestID: "req-008-rev"}
	if _, err := svc.Transition(context.Background(), created.ID, dto.DeviationAnalysisTransitionRequest{
		ToState: "reviewed", Comment: "Reviewed.",
	}, reviewer); err != nil {
		t.Fatalf("review transition: %v", err)
	}
	return &permissionFixture{handler: NewDeviationAnalysisHandler(svc), analysisID: created.ID}
}

func confirmRequest(t *testing.T, engine *gin.Engine, analysisID uint) *httptest.ResponseRecorder {
	t.Helper()
	body := strings.NewReader(`{"to_state":"confirmed","comment":"confirm"}`)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/deviation-analyses/%d/transition", analysisID), body)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

func TestAnalystConfirmForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fx := newReviewedAnalysis(t)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("actor", util.Actor{UserID: 11, Username: "analyst", Role: "data_analyst", RequestID: "req-008-analyst"})
		c.Next()
	})
	engine.POST("/api/v1/deviation-analyses/:id/transition",
		middleware.RequirePermission(constants.PermissionAnalysisReview), fx.handler.Transition)
	recorder := confirmRequest(t, engine, fx.analysisID)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("analyst confirm status = %d, want 403", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "role does not have permission for this action") {
		t.Fatalf("analyst confirm must be rejected by the permission layer, body: %s", recorder.Body.String())
	}
}

func TestScientistConfirmBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fx := newReviewedAnalysis(t)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("actor", util.Actor{UserID: 12, Username: "scientist", Role: "process_scientist", RequestID: "req-008-scientist"})
		c.Next()
	})
	engine.POST("/api/v1/deviation-analyses/:id/transition",
		middleware.RequirePermission(constants.PermissionAnalysisReview), fx.handler.Transition)
	recorder := confirmRequest(t, engine, fx.analysisID)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("scientist confirm status = %d, want 403", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "role cannot confirm analysis results") {
		t.Fatalf("scientist confirm must be blocked by the handler check, body: %s", recorder.Body.String())
	}
}

func TestRoleRestrictedEndpointForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		c.Set("actor", util.Actor{UserID: 11, Username: "analyst", Role: "data_analyst", RequestID: "req-008-role"})
		c.Next()
	})
	engine.GET("/api/v1/restricted", middleware.RequireRoles(constants.RoleReviewer), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/restricted", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("role-restricted endpoint status = %d, want 403", recorder.Code)
	}
}
