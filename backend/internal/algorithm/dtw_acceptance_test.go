package algorithm

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"fermentation-kinetics-deviation-analysis/backend/internal/constants"
	"fermentation-kinetics-deviation-analysis/backend/internal/model"
	"fermentation-kinetics-deviation-analysis/backend/internal/timeseries"
	"fermentation-kinetics-deviation-analysis/backend/internal/util"
)

func TestDTWPathIsChronological(t *testing.T) {
	actual := []float64{1, 2, 3}
	reference := []float64{1, 2, 3}
	_, path, err := DTW(actual, reference, 2)
	if err != nil {
		t.Fatalf("DTW: %v", err)
	}
	if len(path) == 0 {
		t.Fatal("DTW returned an empty alignment path")
	}
	first := path[0]
	if first.ActualIndex != 0 || first.ReferenceIndex != 0 {
		t.Fatalf("alignment path must start at (0,0), got (%d,%d)", first.ActualIndex, first.ReferenceIndex)
	}
	last := path[len(path)-1]
	if last.ActualIndex != len(actual)-1 || last.ReferenceIndex != len(reference)-1 {
		t.Fatalf("alignment path must end at (%d,%d), got (%d,%d)", len(actual)-1, len(reference)-1, last.ActualIndex, last.ReferenceIndex)
	}
}

func longReferenceSnapshot(t *testing.T) Snapshot {
	t.Helper()
	started := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	points := make([]timeseries.Point, 0, 5)
	reference := map[string][]CurvePoint{"ph": {}}
	for hour := 0; hour <= 4; hour++ {
		value := 7.0 - float64(hour)*0.05
		valueCopy := value
		points = append(points, timeseries.Point{
			Timestamp: started.Add(time.Duration(hour) * time.Hour),
			Values:    map[string]*float64{"ph": &valueCopy},
		})
	}
	for hour := 0; hour <= 8; hour++ {
		reference["ph"] = append(reference["ph"], CurvePoint{ElapsedHour: float64(hour), Value: 7.0 - float64(hour)*0.05})
	}
	pointsJSON, err := timeseries.EncodePoints(points)
	if err != nil {
		t.Fatal(err)
	}
	boundaries, _ := util.CanonicalJSON([]PhaseBoundary{
		{Phase: constants.PhaseLag, StartHour: 0, EndHour: 8},
	})
	references, _ := util.CanonicalJSON(reference)
	tolerances, _ := util.CanonicalJSON(map[string]ChannelTolerance{"ph": {Weight: 1, MaxDistance: 1}})
	series := model.SensorSeries{
		ID: 3, VesselID: 2, RecipeID: 4, RunCode: "LONG-REF", Channel: "ph",
		SampleIntervalS: 3600, PointsJSON: pointsJSON, SourceChecksum: util.HashString(pointsJSON),
		StartedAt: started, EndedAt: started.Add(4 * time.Hour),
	}
	recipe := model.CultureRecipe{
		ID: 4, Version: 2, TargetDurationH: 8, PhaseBoundariesJSON: boundaries,
		ReferenceCurvesJSON: references, ToleranceProfileJSON: tolerances,
	}
	return NewSnapshot(series, recipe)
}

func TestEvaluateReferenceLongerNoPanic(t *testing.T) {
	snapshot := longReferenceSnapshot(t)
	result, err := NewEvaluator().Evaluate(snapshot)
	if err != nil {
		t.Fatalf("evaluate with a longer reference curve: %v", err)
	}
	if result.OverallScore < 0 || result.OverallScore > 1 {
		t.Fatalf("overall score out of range: %v", result.OverallScore)
	}
}

func offsetReferenceSnapshot(t *testing.T) Snapshot {
	t.Helper()
	started := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	points := make([]timeseries.Point, 0, 9)
	reference := map[string][]CurvePoint{"ph": {}}
	for hour := 0; hour <= 8; hour++ {
		value := 8.0
		valueCopy := value
		points = append(points, timeseries.Point{
			Timestamp: started.Add(time.Duration(hour) * time.Hour),
			Values:    map[string]*float64{"ph": &valueCopy},
		})
		reference["ph"] = append(reference["ph"], CurvePoint{ElapsedHour: float64(hour), Value: 7.0})
	}
	pointsJSON, err := timeseries.EncodePoints(points)
	if err != nil {
		t.Fatal(err)
	}
	boundaries, _ := util.CanonicalJSON([]PhaseBoundary{
		{Phase: constants.PhaseLag, StartHour: 0, EndHour: 8},
	})
	references, _ := util.CanonicalJSON(reference)
	tolerances, _ := util.CanonicalJSON(map[string]ChannelTolerance{"ph": {Weight: 1, MaxDistance: 1}})
	series := model.SensorSeries{
		ID: 3, VesselID: 2, RecipeID: 4, RunCode: "OFFSET-REF", Channel: "ph",
		SampleIntervalS: 3600, PointsJSON: pointsJSON, SourceChecksum: util.HashString(pointsJSON),
		StartedAt: started, EndedAt: started.Add(8 * time.Hour),
	}
	recipe := model.CultureRecipe{
		ID: 4, Version: 2, TargetDurationH: 8, PhaseBoundariesJSON: boundaries,
		ReferenceCurvesJSON: references, ToleranceProfileJSON: tolerances,
	}
	return NewSnapshot(series, recipe)
}

func TestEvaluateCauseDirection(t *testing.T) {
	snapshot := offsetReferenceSnapshot(t)
	result, err := NewEvaluator().Evaluate(snapshot)
	if err != nil {
		t.Fatalf("evaluate offset snapshot: %v", err)
	}
	if !strings.Contains(result.SuspectedCausesJSON, "above") {
		t.Fatalf("actual above reference must yield an above-cause, got: %s", result.SuspectedCausesJSON)
	}
}

func twoPhaseSnapshot(t *testing.T) Snapshot {
	t.Helper()
	started := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	points := make([]timeseries.Point, 0, 9)
	reference := map[string][]CurvePoint{"ph": {}}
	for hour := 0; hour <= 8; hour++ {
		value := 7.0 - float64(hour)*0.05
		valueCopy := value
		points = append(points, timeseries.Point{
			Timestamp: started.Add(time.Duration(hour) * time.Hour),
			Values:    map[string]*float64{"ph": &valueCopy},
		})
		reference["ph"] = append(reference["ph"], CurvePoint{ElapsedHour: float64(hour), Value: 7.0 - float64(hour)*0.05})
	}
	pointsJSON, err := timeseries.EncodePoints(points)
	if err != nil {
		t.Fatal(err)
	}
	boundaries, _ := util.CanonicalJSON([]PhaseBoundary{
		{Phase: constants.PhaseLag, StartHour: 0, EndHour: 4},
		{Phase: constants.PhaseGrowth, StartHour: 4, EndHour: 8},
	})
	references, _ := util.CanonicalJSON(reference)
	tolerances, _ := util.CanonicalJSON(map[string]ChannelTolerance{"ph": {Weight: 1, MaxDistance: 1}})
	series := model.SensorSeries{
		ID: 3, VesselID: 2, RecipeID: 4, RunCode: "TWO-PHASE", Channel: "ph",
		SampleIntervalS: 3600, PointsJSON: pointsJSON, SourceChecksum: util.HashString(pointsJSON),
		StartedAt: started, EndedAt: started.Add(8 * time.Hour),
	}
	recipe := model.CultureRecipe{
		ID: 4, Version: 2, TargetDurationH: 8, PhaseBoundariesJSON: boundaries,
		ReferenceCurvesJSON: references, ToleranceProfileJSON: tolerances,
	}
	return NewSnapshot(series, recipe)
}

func TestEvaluateAlignedPhaseOrder(t *testing.T) {
	snapshot := twoPhaseSnapshot(t)
	result, err := NewEvaluator().Evaluate(snapshot)
	if err != nil {
		t.Fatalf("evaluate two-phase snapshot: %v", err)
	}
	var aligned []AlignedPoint
	if err := json.Unmarshal([]byte(result.AlignedCurveJSON), &aligned); err != nil {
		t.Fatalf("decode aligned curve: %v", err)
	}
	if len(aligned) == 0 {
		t.Fatal("aligned curve is empty")
	}
	if aligned[0].Phase != string(constants.PhaseLag) {
		t.Fatalf("aligned curve must start with the lag phase, first phase = %q", aligned[0].Phase)
	}
}
