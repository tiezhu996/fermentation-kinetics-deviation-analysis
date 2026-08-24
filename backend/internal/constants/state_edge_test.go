package constants

import "testing"

func TestSeriesReadyEdgeAllowed(t *testing.T) {
	if !CanTransitionSeries(SeriesNormalized, SeriesReady) {
		t.Fatal("normalized series should be able to reach ready")
	}
}

func TestRecipeDraftReentryAllowed(t *testing.T) {
	if !CanTransitionRecipe(RecipeValidated, RecipeDraft) {
		t.Fatal("validated recipe should be able to return to draft")
	}
}

func TestAnalysisQueuedToAnalyzingAllowed(t *testing.T) {
	if !CanTransitionAnalysis(AnalysisQueued, AnalysisAnalyzing) {
		t.Fatal("queued analysis should be able to enter analyzing")
	}
}

func TestDeviationThresholdWatchBoundary(t *testing.T) {
	if DeviationLevelForScore(0.20) != DeviationWatch {
		t.Fatal("score 0.20 should be classified as watch")
	}
}

func TestSeriesImportedRejectedAllowed(t *testing.T) {
	if !CanTransitionSeries(SeriesImported, SeriesRejected) {
		t.Fatal("imported series should be able to be rejected")
	}
}

func TestFailedAnalysisVoidable(t *testing.T) {
	if !CanTransitionAnalysis(AnalysisFailed, AnalysisVoided) {
		t.Fatal("failed analysis should be able to be voided")
	}
}
