package detector

import (
	"fmt"
	"time"

	"spikeshield/db"
	"spikeshield/utils"
)

// Detector monitors price changes and detects wicks (flash crashes)
type Detector struct {
	ThresholdPercent float64 // Minimum wick depth percentage
	WindowMinutes    int     // Time window to check for recovery
	Symbol           string
}

// NewDetector creates a new detector instance
func NewDetector(symbol string, thresholdPercent float64, windowMinutes int) *Detector {
	return &Detector{
		ThresholdPercent: thresholdPercent,
		WindowMinutes:    windowMinutes,
		Symbol:           symbol,
	}
}

// CheckForSpike analyzes recent prices and detects if a wick (flash crash) occurred
// A wick is characterized by:
// 1. Rapid price drop (low is significantly below open/close)
// 2. Price recovers quickly (close is near the open)
func (d *Detector) CheckForSpike() (*db.Spike, error) {
	// Get latest price
	latest, err := db.GetLatestPrice(d.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest price: %w", err)
	}

	// Check if this candle has a significant lower wick
	// Wick is the difference between the low and the higher of open/close
	closeOrOpen := latest.Close
	if latest.Open > latest.Close {
		closeOrOpen = latest.Open // Use open if it's higher (red candle)
	}

	// Calculate wick depth as percentage
	wickDepth := ((closeOrOpen - latest.Low) / closeOrOpen) * 100

	// Also check if price recovered (close is near open, within 2%)
	bodySize := abs(latest.Close - latest.Open)
	priceRecovered := (bodySize / latest.Open) < 0.02 // Body is less than 2% of open price

	utils.LogDebug("Wick check: open=$%.2f, high=$%.2f, low=$%.2f, close=$%.2f, wick=%.2f%%, recovered=%v",
		latest.Open, latest.High, latest.Low, latest.Close, wickDepth, priceRecovered)

	// Detect wick: significant depth AND price recovered
	if wickDepth >= d.ThresholdPercent && priceRecovered {
		spike := &db.Spike{
			Timestamp:   latest.Timestamp,
			Symbol:      d.Symbol,
			PriceBefore: closeOrOpen,
			PriceAfter:  latest.Low,
			DropPercent: wickDepth,
		}

		// Save spike to database
		if err := db.InsertSpike(spike); err != nil {
			return nil, fmt.Errorf("failed to insert spike: %w", err)
		}

		utils.LogInfo("🚨 WICK DETECTED! %s had %.2f%% wick from $%.2f to $%.2f (recovered to $%.2f)",
			d.Symbol, wickDepth, closeOrOpen, latest.Low, latest.Close)

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

// ContinuousMonitor runs wick detection in a loop
func (d *Detector) ContinuousMonitor(checkInterval time.Duration, onSpike func(*db.Spike)) {
	utils.LogInfo("Starting continuous wick monitoring for %s (threshold: %.2f%%)", d.Symbol, d.ThresholdPercent)

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
// Detects wicks: candles with long lower shadows where price recovered
func (d *Detector) DetectAllInRange(start, end time.Time) ([]*db.Spike, error) {
	utils.LogInfo("Analyzing price data for wicks from %s to %s", start.Format(time.RFC3339), end.Format(time.RFC3339))

	prices, err := db.GetPricesBetween(d.Symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get prices: %w", err)
	}

	if len(prices) < 1 {
		return nil, fmt.Errorf("insufficient price data")
	}

	var spikes []*db.Spike

	// Scan through all candles looking for wicks
	for _, candle := range prices {
		// Use the higher of open or close as reference
		closeOrOpen := candle.Close
		if candle.Open > candle.Close {
			closeOrOpen = candle.Open
		}

		// Calculate wick depth
		wickDepth := ((closeOrOpen - candle.Low) / closeOrOpen) * 100

		// Check if price recovered (small body)
		bodySize := abs(candle.Close - candle.Open)
		priceRecovered := (bodySize / candle.Open) < 0.02

		// Detect wick
		if wickDepth >= d.ThresholdPercent && priceRecovered {
			spike := &db.Spike{
				Timestamp:   candle.Timestamp,
				Symbol:      d.Symbol,
				PriceBefore: closeOrOpen,
				PriceAfter:  candle.Low,
				DropPercent: wickDepth,
			}

			if err := db.InsertSpike(spike); err != nil {
				utils.LogError("Failed to insert wick: %v", err)
				continue
			}

			spikes = append(spikes, spike)
			utils.LogInfo("Wick detected at %s: %.2f%% wick from $%.2f to $%.2f (recovered to $%.2f)",
				spike.Timestamp.Format(time.RFC3339), wickDepth, closeOrOpen, candle.Low, candle.Close)
		}
	}

	utils.LogInfo("Analysis complete: found %d wick(s)", len(spikes))
	return spikes, nil
}
