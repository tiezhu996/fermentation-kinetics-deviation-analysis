package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fermentation-kinetics-deviation-analysis/backend/internal/config"
	"fermentation-kinetics-deviation-analysis/backend/internal/constants"
	"fermentation-kinetics-deviation-analysis/backend/internal/middleware"
	"fermentation-kinetics-deviation-analysis/backend/internal/model"
	"fermentation-kinetics-deviation-analysis/backend/internal/repository"
	"fermentation-kinetics-deviation-analysis/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newHandlerTestDB(t *testing.T) *gorm.DB {
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

func TestRecipeGetMissingIs404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := newHandlerTestDB(t)
	recipeRepo := repository.NewCultureRecipeRepository(db)
	vesselRepo := repository.NewFermentationVesselRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	h := NewCultureRecipeHandler(service.NewCultureRecipeService(recipeRepo, vesselRepo, auditRepo))
	cfg := config.Config{JWTSecret: "handler-test-secret-1234567890", JWTIssuer: "test-issuer", JWTExpiry: time.Hour}
	auth := middleware.NewAuthenticator(repository.NewUserRepository(db), cfg)
	engine := gin.New()
	api := engine.Group("/api/v1")
	api.Use(auth.RequireAuth())
	group := api.Group("/culture-recipes")
	group.GET("/:id", middleware.RequirePermission(constants.PermissionRead), h.Get)

	now := time.Now().UTC()
	claims := middleware.Claims{
		UserID: 1, Username: "admin", DisplayName: "Admin", Role: "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: cfg.JWTIssuer, Subject: "admin",
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(cfg.JWTExpiry)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/culture-recipes/999999", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", recorder.Code, recorder.Body.String())
	}
}
