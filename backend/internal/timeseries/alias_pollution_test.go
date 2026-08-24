package timeseries

import "testing"

func TestValidatePointsRemainDistinct(t *testing.T) {
	raw := []byte(`[
		{"timestamp":"2026-08-20T00:00:00Z","values":{"ph":1,"do":50}},
		{"timestamp":"2026-08-20T00:01:00Z","values":{"ph":2,"do":51}},
		{"timestamp":"2026-08-20T00:02:00Z","values":{"ph":3,"do":52}},
		{"timestamp":"2026-08-20T00:03:00Z","values":{"ph":100,"do":53}}
	]`)
	points, _, err := Validate(raw, "multichannel", 60)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(points) != 4 {
		t.Fatalf("got %d points, want 4", len(points))
	}
	if got := *points[0].Values["ph"]; got != 1 {
		t.Fatalf("point0 ph = %v, want 1 (later point leaked)", got)
	}
	if got := *points[0].Values["do"]; got != 50 {
		t.Fatalf("point0 do = %v, want 50 (later point leaked)", got)
	}
	if *points[0].Values["ph"] == *points[3].Values["ph"] {
		t.Fatal("all points collapsed to the last point value")
	}
	if *points[1].Values["do"] != 51 || *points[2].Values["do"] != 52 {
		t.Fatalf("middle point values leaked: point1 do=%v point2 do=%v", *points[1].Values["do"], *points[2].Values["do"])
	}
	changed := 9.9
	points[0].Values["ph"] = &changed
	if *points[1].Values["ph"] == 9.9 {
		t.Fatal("points share the same values map: mutating point0 changed point1")
	}
}
