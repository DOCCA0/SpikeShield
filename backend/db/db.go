package db

import (
	"database/sql"
	"fmt"
	"time"

	"spikeshield/utils"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// PriceData represents a price record
type PriceData struct {
	ID        int
	Timestamp time.Time
	Symbol    string
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// Spike represents a detected spike event
type Spike struct {
	ID          int
	Timestamp   time.Time
	Symbol      string
	PriceBefore float64
	PriceAfter  float64
	DropPercent float64
	DetectedAt  time.Time
}

// Policy represents an insurance policy
type Policy struct {
	ID             int
	UserAddress    string
	Premium        float64
	CoverageAmount float64
	PurchaseTime   time.Time
	ExpiryTime     time.Time
	Status         string
	TxHash         string
}

// Payout represents a payout record
type Payout struct {
	ID          int
	PolicyID    int
	UserAddress string
	Amount      float64
	SpikeID     int
	TxHash      string
	ExecutedAt  time.Time
}

// Connect establishes database connection
func Connect(cfg *utils.Config) error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
	)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	utils.LogInfo("Database connected successfully")
	return nil
}

// InsertPrice inserts a price record
func InsertPrice(p *PriceData) error {
	query := `INSERT INTO prices (timestamp, symbol, open, high, low, close, volume) 
			  VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	return DB.QueryRow(query, p.Timestamp, p.Symbol, p.Open, p.High, p.Low, p.Close, p.Volume).Scan(&p.ID)
}

// InsertSpike inserts a spike detection record
func InsertSpike(s *Spike) error {
	query := `INSERT INTO spikes (timestamp, symbol, price_before, price_after, drop_percent) 
			  VALUES ($1, $2, $3, $4, $5) RETURNING id`
	return DB.QueryRow(query, s.Timestamp, s.Symbol, s.PriceBefore, s.PriceAfter, s.DropPercent).Scan(&s.ID)
}

// GetLatestPrice retrieves the most recent price for a symbol
func GetLatestPrice(symbol string) (*PriceData, error) {
	query := `SELECT id, timestamp, symbol, open, high, low, close, volume 
			  FROM prices WHERE symbol = $1 ORDER BY timestamp DESC LIMIT 1`

	p := &PriceData{}
	err := DB.QueryRow(query, symbol).Scan(&p.ID, &p.Timestamp, &p.Symbol, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// GetPricesBetween retrieves prices within a time range
func GetPricesBetween(symbol string, start, end time.Time) ([]*PriceData, error) {
	query := `SELECT id, timestamp, symbol, open, high, low, close, volume 
			  FROM prices WHERE symbol = $1 AND timestamp BETWEEN $2 AND $3 ORDER BY timestamp`

	rows, err := DB.Query(query, symbol, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []*PriceData
	for rows.Next() {
		p := &PriceData{}
		if err := rows.Scan(&p.ID, &p.Timestamp, &p.Symbol, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume); err != nil {
			return nil, err
		}
		prices = append(prices, p)
	}
	return prices, nil
}

// GetActivePolicies retrieves all active policies
func GetActivePolicies() ([]*Policy, error) {
	query := `SELECT id, user_address, premium, coverage_amount, purchase_time, expiry_time, status, COALESCE(tx_hash, '') 
			  FROM policies WHERE status = 'active' AND expiry_time > NOW()`

	rows, err := DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []*Policy
	for rows.Next() {
		p := &Policy{}
		if err := rows.Scan(&p.ID, &p.UserAddress, &p.Premium, &p.CoverageAmount, &p.PurchaseTime, &p.ExpiryTime, &p.Status, &p.TxHash); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}
	return policies, nil
}

// InsertPayout records a payout execution
func InsertPayout(p *Payout) error {
	query := `INSERT INTO payouts (policy_id, user_address, amount, spike_id, tx_hash) 
			  VALUES ($1, $2, $3, $4, $5) RETURNING id`
	return DB.QueryRow(query, p.PolicyID, p.UserAddress, p.Amount, p.SpikeID, p.TxHash).Scan(&p.ID)
}

// UpdatePolicyStatus updates policy status
func UpdatePolicyStatus(policyID int, status string) error {
	query := `UPDATE policies SET status = $1 WHERE id = $2`
	_, err := DB.Exec(query, status, policyID)
	return err
}

// Close closes database connection
func Close() {
	if DB != nil {
		DB.Close()
	}
}

// GetRecentSpikes retrieves recent spike detection events
func GetRecentSpikes(limit int) ([]*Spike, error) {
	query := `SELECT id, timestamp, symbol, price_before, price_after, drop_percent, detected_at 
			  FROM spikes ORDER BY detected_at DESC LIMIT $1`

	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spikes []*Spike
	for rows.Next() {
		s := &Spike{}
		if err := rows.Scan(&s.ID, &s.Timestamp, &s.Symbol, &s.PriceBefore, &s.PriceAfter, &s.DropPercent, &s.DetectedAt); err != nil {
			return nil, err
		}
		spikes = append(spikes, s)
	}
	return spikes, nil
}

// GetRecentPrices retrieves recent price data
func GetRecentPrices(symbol string, limit int) ([]*PriceData, error) {
	query := `SELECT id, timestamp, symbol, open, high, low, close, volume 
			  FROM prices WHERE symbol = $1 ORDER BY timestamp DESC LIMIT $2`

	rows, err := DB.Query(query, symbol, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []*PriceData
	for rows.Next() {
		p := &PriceData{}
		if err := rows.Scan(&p.ID, &p.Timestamp, &p.Symbol, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume); err != nil {
			return nil, err
		}
		prices = append(prices, p)
	}
	return prices, nil
}

// GetRecentPayouts retrieves recent payout records
func GetRecentPayouts(limit int) ([]*Payout, error) {
	query := `SELECT id, policy_id, user_address, amount, spike_id, COALESCE(tx_hash, ''), executed_at 
			  FROM payouts ORDER BY executed_at DESC LIMIT $1`

	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payouts []*Payout
	for rows.Next() {
		p := &Payout{}
		if err := rows.Scan(&p.ID, &p.PolicyID, &p.UserAddress, &p.Amount, &p.SpikeID, &p.TxHash, &p.ExecutedAt); err != nil {
			return nil, err
		}
		payouts = append(payouts, p)
	}
	return payouts, nil
}

// SystemStats represents system statistics
type SystemStats struct {
	TotalSpikes    int `json:"total_spikes"`
	TotalPayouts   int `json:"total_payouts"`
	TotalPolicies  int `json:"total_policies"`
	ActivePolicies int `json:"active_policies"`
	TotalPrices    int `json:"total_prices"`
}

// GetSystemStats retrieves system statistics
func GetSystemStats() (*SystemStats, error) {
	stats := &SystemStats{}

	// Count spikes
	DB.QueryRow("SELECT COUNT(*) FROM spikes").Scan(&stats.TotalSpikes)

	// Count payouts
	DB.QueryRow("SELECT COUNT(*) FROM payouts").Scan(&stats.TotalPayouts)

	// Count total policies
	DB.QueryRow("SELECT COUNT(*) FROM policies").Scan(&stats.TotalPolicies)

	// Count active policies
	DB.QueryRow("SELECT COUNT(*) FROM policies WHERE status = 'active' AND expiry_time > NOW()").Scan(&stats.ActivePolicies)

	// Count price records
	DB.QueryRow("SELECT COUNT(*) FROM prices").Scan(&stats.TotalPrices)

	return stats, nil
}
