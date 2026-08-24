package service

import (
	"context"
	"testing"
	"time"

	"fermentation-kinetics-deviation-analysis/backend/internal/dto"
	"fermentation-kinetics-deviation-analysis/backend/internal/model"
	"fermentation-kinetics-deviation-analysis/backend/internal/repository"
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
)

func ptrString(v string) *string        { return &v }
func ptrFloat(v float64) *float64       { return &v }
func ptrChannels(v []string) *[]string  { return &v }
func ptrTime(v time.Time) *time.Time    { return &v }

func updateActor() util.Actor {
	return util.Actor{UserID: 1, Username: "admin", Role: "admin", RequestID: "req-upd"}
}

func newVesselUpdateEnv(t *testing.T) (*FermentationVesselService, uint) {
	t.Helper()
	db := newTestDB(t)
	vesselRepo := repository.NewFermentationVesselRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	vessel := model.FermentationVessel{
		VesselCode: "FV-UPD", Name: "Update vessel", WorkingVolumeL: 100,
		SensorChannels: `["ph"]`, Location: "Lab", OwnerTeam: "Process",
		VesselState: "active", CommissionedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := vesselRepo.Create(context.Background(), &vessel); err != nil {
		t.Fatal(err)
	}
	return NewFermentationVesselService(vesselRepo, auditRepo), vessel.ID
}

func newRecipeUpdateEnv(t *testing.T) (*CultureRecipeService, uint) {
	t.Helper()
	db := newTestDB(t)
	vesselRepo := repository.NewFermentationVesselRepository(db)
	recipeRepo := repository.NewCultureRecipeRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	vessel := model.FermentationVessel{
		VesselCode: "FV-REC", Name: "Recipe vessel", WorkingVolumeL: 100,
		SensorChannels: `["ph"]`, Location: "Lab", OwnerTeam: "Process",
		VesselState: "active", CommissionedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := vesselRepo.Create(context.Background(), &vessel); err != nil {
		t.Fatal(err)
	}
	boundaries, references, tolerances := testRecipeConfig(t)
	recipe := model.CultureRecipe{
		VesselID: vessel.ID, RecipeCode: "UPDATE-A", Version: 1, Organism: "Test organism",
		TargetDurationH: 8, PhaseBoundariesJSON: string(boundaries), ReferenceCurvesJSON: string(references),
		ToleranceProfileJSON: string(tolerances), RecipeState: "draft",
		CreatedBy: 8, CreatedByName: "scientist", CreatedAt: now, UpdatedAt: now,
	}
	if err := recipeRepo.Create(context.Background(), &recipe); err != nil {
		t.Fatal(err)
	}
	return NewCultureRecipeService(recipeRepo, vesselRepo, auditRepo), recipe.ID
}

func TestUpdateVesselWithoutNameNoPanic(t *testing.T) {
	svc, id := newVesselUpdateEnv(t)
	now := time.Now().UTC()
	_, err := svc.Update(context.Background(), id, dto.UpdateFermentationVesselRequest{
		WorkingVolumeL: ptrFloat(150), Location: ptrString("Lab2"), OwnerTeam: ptrString("Team"),
		SensorChannels: ptrChannels([]string{"ph", "do"}), CommissionedAt: ptrTime(now),
	}, updateActor())
	if err != nil {
		t.Fatalf("update without name should not panic: %v", err)
	}
}

func TestUpdateVesselWithoutChannelsNoPanic(t *testing.T) {
	svc, id := newVesselUpdateEnv(t)
	now := time.Now().UTC()
	_, err := svc.Update(context.Background(), id, dto.UpdateFermentationVesselRequest{
		Name: ptrString("Renamed"), WorkingVolumeL: ptrFloat(150), Location: ptrString("Lab2"),
		OwnerTeam: ptrString("Team"), CommissionedAt: ptrTime(now),
	}, updateActor())
	if err != nil {
		t.Fatalf("update without channels should not panic: %v", err)
	}
}

func TestUpdateVesselWithoutCommissionedNoPanic(t *testing.T) {
	svc, id := newVesselUpdateEnv(t)
	_, err := svc.Update(context.Background(), id, dto.UpdateFermentationVesselRequest{
		Name: ptrString("Renamed"), WorkingVolumeL: ptrFloat(150), Location: ptrString("Lab2"),
		OwnerTeam: ptrString("Team"), SensorChannels: ptrChannels([]string{"ph"}),
	}, updateActor())
	if err != nil {
		t.Fatalf("update without commissioned should not panic: %v", err)
	}
}

func TestUpdateRecipeWithoutOrganismNoPanic(t *testing.T) {
	svc, id := newRecipeUpdateEnv(t)
	_, err := svc.Update(context.Background(), id, dto.UpdateCultureRecipeRequest{Version: 1}, updateActor())
	if err != nil {
		t.Fatalf("update without organism should not panic: %v", err)
	}
}

func TestUpdateRecipeWithoutDurationNoPanic(t *testing.T) {
	svc, id := newRecipeUpdateEnv(t)
	_, err := svc.Update(context.Background(), id, dto.UpdateCultureRecipeRequest{Version: 1, Organism: ptrString("New strain")}, updateActor())
	if err != nil {
		t.Fatalf("update without duration should not panic: %v", err)
	}
}
