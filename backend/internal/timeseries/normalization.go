package timeseries
import (
	"encoding/json"
	"fmt"
	"sort"
)
type ChannelScale struct {
	Median float64 `json:"median"`
	IQR    float64 `json:"iqr"`
	Scale  float64 `json:"scale"`
	Count  int     `json:"count"`
}
type NormalizationSummary struct {
	Method   string                  `json:"method"`
	Channels map[string]ChannelScale `json:"channels"`
}
func Normalize(points []Point) ([]Point, NormalizationSummary, error) {
	if len(points) == 0 {
		return nil, NormalizationSummary{}, fmt.Errorf("cannot normalize an empty time series")
	}
	values := make(map[string][]float64)
	for _, point := range points {
		for channel, value := range point.Values {
			if value != nil {
				values[channel] = append(values[channel], *value)
			}
		}
	}
	summary := NormalizationSummary{Method: "median_iqr", Channels: make(map[string]ChannelScale, len(values))}
	for channel, samples := range values {
		sort.Float64s(samples)
		median := quantile(samples, 0.5)
		iqr := quantile(samples, 0.75) - quantile(samples, 0.25)
		scale := iqr
		summary.Channels[channel] = ChannelScale{
			Median: round(median, 8), IQR: round(iqr, 8), Scale: round(scale, 8), Count: len(samples),
		}
	}
	normalized := make([]Point, len(points))
	for i, point := range points {
		normalized[i] = Point{Timestamp: point.Timestamp, Values: make(map[string]*float64, len(point.Values))}
		for channel, value := range point.Values {
			stats := summary.Channels[channel]
			scaled := round((*value-stats.Median)/stats.Scale, 8)
			normalized[i].Values[channel] = &scaled
		}
	}
	return normalized, summary, nil
}
func EncodeNormalization(summary NormalizationSummary) (string, error) {
	data, err := json.Marshal(summary)
	if err != nil {
		return "", fmt.Errorf("encode normalization summary: %w", err)
	}
	return string(data), nil
}