package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"fermentation-kinetics-deviation-analysis/backend/internal/dto"
	"fermentation-kinetics-deviation-analysis/backend/internal/repository"
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
)

func TestRecipeMissingReturnsNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewCultureRecipeService(
		repository.NewCultureRecipeRepository(db),
		repository.NewFermentationVesselRepository(db),
		repository.NewAuditRepository(db),
	)
	_, err := svc.Get(context.Background(), 999999)
	var appErr *util.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error=%v, want AppError", err)
	}
	if appErr.Status != http.StatusNotFound || appErr.Code != util.CodeNotFound {
		t.Fatalf("appErr status=%d code=%s, want 404 NOT_FOUND", appErr.Status, appErr.Code)
	}
}

func TestCopyMissingRecipeNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewCultureRecipeService(
		repository.NewCultureRecipeRepository(db),
		repository.NewFermentationVesselRepository(db),
		repository.NewAuditRepository(db),
	)
	_, err := svc.Copy(context.Background(), 999999, dto.CopyCultureRecipeRequest{},
		util.Actor{UserID: 1, Username: "admin", Role: "admin"})
	var appErr *util.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error=%v, want AppError", err)
	}
	if appErr.Status != http.StatusNotFound || appErr.Code != util.CodeNotFound {
		t.Fatalf("appErr status=%d code=%s, want 404 NOT_FOUND", appErr.Status, appErr.Code)
	}
}

func TestUpdateMissingRecipeNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewCultureRecipeService(
		repository.NewCultureRecipeRepository(db),
		repository.NewFermentationVesselRepository(db),
		repository.NewAuditRepository(db),
	)
	_, err := svc.Update(context.Background(), 999999, dto.UpdateCultureRecipeRequest{},
		util.Actor{UserID: 1, Username: "admin", Role: "admin"})
	var appErr *util.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error=%v, want AppError", err)
	}
	if appErr.Status != http.StatusNotFound || appErr.Code != util.CodeNotFound {
		t.Fatalf("appErr status=%d code=%s, want 404 NOT_FOUND", appErr.Status, appErr.Code)
	}
}

func TestTransitionMissingRecipeNotFound(t *testing.T) {
	db := newTestDB(t)
	svc := NewCultureRecipeService(
		repository.NewCultureRecipeRepository(db),
		repository.NewFermentationVesselRepository(db),
		repository.NewAuditRepository(db),
	)
	_, err := svc.Transition(context.Background(), 999999, dto.CultureRecipeTransitionRequest{ToState: "validated", Version: 1},
		util.Actor{UserID: 1, Username: "admin", Role: "admin"})
	var appErr *util.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error=%v, want AppError", err)
	}
	if appErr.Status != http.StatusNotFound || appErr.Code != util.CodeNotFound {
		t.Fatalf("appErr status=%d code=%s, want 404 NOT_FOUND", appErr.Status, appErr.Code)
	}
}
