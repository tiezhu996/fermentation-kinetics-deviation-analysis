package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"fermentation-kinetics-deviation-analysis/backend/internal/constants"
	"fermentation-kinetics-deviation-analysis/backend/internal/dto"
	"fermentation-kinetics-deviation-analysis/backend/internal/model"
	"fermentation-kinetics-deviation-analysis/backend/internal/repository"
	"fermentation-kinetics-deviation-analysis/backend/internal/timeseries"
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
)

func newImportReadyService(t *testing.T) (*SensorSeriesService, uint, uint) {
	t.Helper()
	db := newTestDB(t)
	vesselRepo := repository.NewFermentationVesselRepository(db)
	recipeRepo := repository.NewCultureRecipeRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	now := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	vessel := model.FermentationVessel{
		VesselCode: "FV-ALIAS", Name: "Alias vessel", WorkingVolumeL: 100,
		SensorChannels: `["ph","do"]`, Location: "Lab", OwnerTeam: "Process",
		VesselState: "active", CommissionedAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if err := vesselRepo.Create(context.Background(), &vessel); err != nil {
		t.Fatal(err)
	}
	boundaries, references, tolerances := testRecipeConfig(t)
	recipe := model.CultureRecipe{
		VesselID: vessel.ID, RecipeCode: "ALIAS-A", Version: 1, Organism: "Test organism",
		TargetDurationH: 8, PhaseBoundariesJSON: string(boundaries), ReferenceCurvesJSON: string(references),
		ToleranceProfileJSON: string(tolerances), RecipeState: string(constants.RecipePublished),
		CreatedBy: 8, CreatedByName: "scientist", CreatedAt: now, UpdatedAt: now,
	}
	if err := recipeRepo.Create(context.Background(), &recipe); err != nil {
		t.Fatal(err)
	}
	svc := NewSensorSeriesService(repository.NewSensorSeriesRepository(db), recipeRepo, vesselRepo, auditRepo)
	return svc, vessel.ID, recipe.ID
}

func distinctAliasPoints(t *testing.T) json.RawMessage {
	t.Helper()
	ph := []float64{1, 2, 3, 100}
	do := []float64{50, 51, 52, 53}
	points := make([]timeseries.Point, 0, 4)
	for i := 0; i < 4; i++ {
		p, d := ph[i], do[i]
		points = append(points, timeseries.Point{
			Timestamp: time.Date(2026, 8, 20, 0, i, 0, 0, time.UTC),
			Values:    map[string]*float64{"ph": &p, "do": &d},
		})
	}
	raw, err := timeseries.EncodePoints(points)
	if err != nil {
		t.Fatal(err)
	}
	return json.RawMessage(raw)
}

func aliasActor() util.Actor {
	return util.Actor{UserID: 9, Username: "analyst", Role: string(constants.RoleDataAnalyst), RequestID: "req-alias"}
}

func TestImportPreservesPerPointValues(t *testing.T) {
	svc, vesselID, recipeID := newImportReadyService(t)
	imported, err := svc.Import(context.Background(), dto.ImportSensorSeriesRequest{
		VesselID: vesselID, RecipeID: recipeID, RunCode: "RUN-ALIAS-1", Channel: "multichannel",
		SampleIntervalS: 60, PointsJSON: distinctAliasPoints(t),
	}, aliasActor())
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	stored, err := timeseries.DecodePoints(string(imported.PointsJSON))
	if err != nil {
		t.Fatalf("decode stored points: %v", err)
	}
	if len(stored) != 4 {
		t.Fatalf("stored %d points, want 4", len(stored))
	}
	if *stored[0].Values["ph"] == *stored[3].Values["ph"] {
		t.Fatal("stored points collapsed to one value")
	}
	if *stored[0].Values["ph"] != 1 || *stored[3].Values["ph"] != 100 {
		t.Fatalf("stored ph leaked: first=%v last=%v", *stored[0].Values["ph"], *stored[3].Values["ph"])
	}
	if *stored[1].Values["do"] != 51 {
		t.Fatalf("stored do point1 = %v, want 51", *stored[1].Values["do"])
	}
}

func TestValidateTransitionPreservesPoints(t *testing.T) {
	svc, vesselID, recipeID := newImportReadyService(t)
	imported, err := svc.Import(context.Background(), dto.ImportSensorSeriesRequest{
		VesselID: vesselID, RecipeID: recipeID, RunCode: "RUN-ALIAS-2", Channel: "multichannel",
		SampleIntervalS: 60, PointsJSON: distinctAliasPoints(t),
	}, aliasActor())
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	validated, err := svc.Transition(context.Background(), imported.ID, dto.SensorSeriesTransitionRequest{ToState: "validated"}, aliasActor())
	if err != nil {
		t.Fatalf("transition to validated: %v", err)
	}
	stored, err := timeseries.DecodePoints(string(validated.PointsJSON))
	if err != nil {
		t.Fatalf("decode validated points: %v", err)
	}
	if *stored[0].Values["ph"] == *stored[3].Values["ph"] {
		t.Fatal("validated points collapsed to one value")
	}
	if *stored[0].Values["ph"] != 1 || *stored[3].Values["ph"] != 100 {
		t.Fatalf("validated ph leaked: first=%v last=%v", *stored[0].Values["ph"], *stored[3].Values["ph"])
	}
}

func TestNormalizeTransitionUsesDistinctValues(t *testing.T) {
	svc, vesselID, recipeID := newImportReadyService(t)
	imported, err := svc.Import(context.Background(), dto.ImportSensorSeriesRequest{
		VesselID: vesselID, RecipeID: recipeID, RunCode: "RUN-ALIAS-3", Channel: "multichannel",
		SampleIntervalS: 60, PointsJSON: distinctAliasPoints(t),
	}, aliasActor())
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if _, err := svc.Transition(context.Background(), imported.ID, dto.SensorSeriesTransitionRequest{ToState: "validated"}, aliasActor()); err != nil {
		t.Fatalf("transition to validated: %v", err)
	}
	normalized, err := svc.Transition(context.Background(), imported.ID, dto.SensorSeriesTransitionRequest{ToState: "normalized"}, aliasActor())
	if err != nil {
		t.Fatalf("transition to normalized: %v", err)
	}
	var summary timeseries.NormalizationSummary
	if err := json.Unmarshal(normalized.NormalizationJSON, &summary); err != nil {
		t.Fatalf("decode normalization summary: %v", err)
	}
	stats := summary.Channels["ph"]
	if stats.Median != 2.5 || stats.IQR != 25.5 {
		t.Fatalf("ph stats = %+v, want median 2.5 IQR 25.5 computed from distinct values", stats)
	}
}
