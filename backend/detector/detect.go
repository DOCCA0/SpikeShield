package detector

import (
	"fmt"
	"time"

	"spikeshield/db"
	"spikeshield/utils"
)

// Detector monitors price changes and detects spikes (long wicks)
type Detector struct {
	ThresholdPercent float64 // Minimum range percentage for spike detection
	BodyRatioMax     float64 // Maximum body/range ratio (smaller = longer wick)
	Symbol           string
}

// NewDetector creates a new detector instance
func NewDetector(symbol string, thresholdPercent float64, bodyRatioMax float64) *Detector {
	return &Detector{
		ThresholdPercent: thresholdPercent,
		BodyRatioMax:     bodyRatioMax,
		Symbol:           symbol,
	}
}

// CheckForSpike analyzes recent prices and detects if a spike (long wick) occurred
// A spike is characterized by:
// 1. Small body: abs(open-close)/(high-low) <= body_ratio_max (default 0.3)
// 2. Large range: (high-low)/close >= threshold_percent (default 0.1 = 10%)
func (d *Detector) CheckForSpike() (*db.Spike, error) {
	// Get latest price
	latest, err := db.GetLatestPrice(d.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest price: %w", err)
	}

	// Calculate body size relative to total range
	bodySize := abs(latest.Open - latest.Close)
	totalRange := latest.High - latest.Low

	// Avoid division by zero
	if totalRange == 0 || latest.Close == 0 {
		return nil, nil
	}

	bodyRatio := bodySize / totalRange      // Small body means long wick
	rangeRatio := totalRange / latest.Close // Range as ratio of close

	utils.LogDebug("Spike check: open=$%.2f, high=$%.2f, low=$%.2f, close=$%.2f, bodyRatio=%.4f, rangeRatio=%.4f",
		latest.Open, latest.High, latest.Low, latest.Close, bodyRatio, rangeRatio)

	// Detect spike: small body (< bodyRatioMax) AND large range (>= threshold)
	if bodyRatio <= d.BodyRatioMax && rangeRatio >= d.ThresholdPercent {
		spike := &db.Spike{
			Timestamp:   latest.Timestamp,
			Symbol:      d.Symbol,
			PriceBefore: latest.Open,
			PriceAfter:  latest.High, // Record the spike high
			DropPercent: rangeRatio * 100,
		}

		// Save spike to database
		if err := db.InsertSpike(spike); err != nil {
			return nil, fmt.Errorf("failed to insert spike: %w", err)
		}

		utils.LogInfo("🚨 SPIKE DETECTED! %s had %.2f%% range (body ratio: %.2f%%) - High: $%.2f, Low: $%.2f",
			d.Symbol, rangeRatio*100, bodyRatio*100, latest.High, latest.Low)

		return spike, nil
	}

	return nil, nil
}

// abs returns absolute value of float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// ContinuousMonitor runs spike detection in a loop
func (d *Detector) ContinuousMonitor(checkInterval time.Duration, onSpike func(*db.Spike)) {
	utils.LogInfo("Starting continuous spike monitoring for %s (threshold: %.1f%%)", d.Symbol, d.ThresholdPercent*100)

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		spike, err := d.CheckForSpike()
		if err != nil {
			utils.LogError("Detection error: %v", err)
			continue
		}

		if spike != nil && onSpike != nil {
			// Trigger callback when spike is detected
			onSpike(spike)
		}
	}
}

// DetectAllInRange analyzes all price data in a time range (for replay mode)
// Detects spikes: candles with long wicks (small body, large range)
func (d *Detector) DetectAllInRange() ([]*db.Spike, error) {
	utils.LogInfo("Analyzing price data for spikes for symbol %s", d.Symbol)

	prices, err := db.GetAllPrices(d.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get prices: %w", err)
	}

	if len(prices) < 1 {
		return nil, fmt.Errorf("insufficient price data")
	}

	var spikes []*db.Spike

	// Scan through all candles looking for spikes
	for _, candle := range prices {
		bodySize := abs(candle.Open - candle.Close)
		totalRange := candle.High - candle.Low

		// Skip if no range or invalid data
		if totalRange == 0 || candle.Close == 0 {
			continue
		}

		// Calculate body ratio and range ratio
		bodyRatio := bodySize / totalRange
		rangeRatio := totalRange / candle.Close

		// Detect spike: small body AND large range
		if bodyRatio <= d.BodyRatioMax && rangeRatio >= d.ThresholdPercent {
			spike := &db.Spike{
				Timestamp:   candle.Timestamp,
				Symbol:      d.Symbol,
				PriceBefore: candle.Open,
				PriceAfter:  candle.High,
				DropPercent: rangeRatio * 100,
			}

			if err := db.InsertSpike(spike); err != nil {
				utils.LogError("Failed to insert spike: %v", err)
				continue
			}

			spikes = append(spikes, spike)
			utils.LogInfo("Spike detected at %s: %.2f%% range (body: %.2f%%) - High: $%.2f, Low: $%.2f, Close: $%.2f",
				spike.Timestamp.Format(time.RFC3339), rangeRatio*100, bodyRatio*100, candle.High, candle.Low, candle.Close)
		}
	}

	utils.LogInfo("Analysis complete: found %d spike(s)", len(spikes))
	return spikes, nil
}
