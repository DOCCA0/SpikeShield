package api

import (
	"encoding/csv"
	"net/http"
	"os"
	"strconv"
	"time"

	"spikeshield/db"
	"spikeshield/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Server handles HTTP API requests
type Server struct {
	addr   string
	router *gin.Engine
}

// NewServer creates a new API server with Gin
func NewServer(addr string) *Server {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())

	// CORS middleware
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	config.AllowHeaders = []string{"Content-Type"}
	router.Use(cors.New(config))

	s := &Server{
		addr:   addr,
		router: router,
	}

	// Register routes
	s.setupRoutes()

	return s
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	api := s.router.Group("/api")
	{
		api.GET("/health", s.handleHealth)
		api.GET("/spikes", s.handleSpikes)
		api.GET("/prices", s.handlePrices)
		api.GET("/payouts", s.handlePayouts)
		api.GET("/stats", s.handleStats)
		api.POST("/insert_fake_kline", s.handleInsertFakeKline)
	}
}

// Start begins serving HTTP requests
func (s *Server) Start() error {
	utils.LogInfo("🌐 API Server starting on %s", s.addr)
	return s.router.Run(s.addr)
}

// handleHealth returns server health status
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleSpikes returns recent wick detection events
func (s *Server) handleSpikes(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	spikes, err := db.GetRecentSpikes(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch spikes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(spikes),
		"spikes": spikes,
	})
}

// handlePrices returns recent price data
func (s *Server) handlePrices(c *gin.Context) {
	symbol := c.DefaultQuery("symbol", "BTCUSDT")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))

	prices, err := db.GetRecentPrices(symbol, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch prices"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"symbol": symbol,
		"count":  len(prices),
		"prices": prices,
	})
}

// handlePayouts returns payout history
func (s *Server) handlePayouts(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	payouts, err := db.GetRecentPayouts(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payouts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(payouts),
		"payouts": payouts,
	})
}

// handleStats returns system statistics
func (s *Server) handleStats(c *gin.Context) {
	stats, err := db.GetSystemStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	latestPrice, _ := db.GetLatestPrice("BTCUSDT")

	c.JSON(http.StatusOK, gin.H{
		"stats":        stats,
		"latest_price": latestPrice,
		"status":       "monitoring",
	})
}

// handleInsertFakeKline inserts CSV data line by line with 1 second delay
func (s *Server) handleInsertFakeKline(c *gin.Context) {
	utils.LogInfo("📝 Starting fake kline insertion from CSV...")

	// Run in background to avoid blocking the HTTP response
	go func() {
		// First, delete all existing prices
		utils.LogInfo("🗑️  Deleting all existing prices...")
		if err := db.DeleteAllPrices(); err != nil {
			utils.LogError("Failed to delete prices: %v", err)
			return
		}
		utils.LogInfo("✅ All prices deleted")

		csvPath := "../data/btcusdt_wick_test.csv"
		file, err := os.Open(csvPath)
		if err != nil {
			utils.LogError("Failed to open CSV file: %v", err)
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)

		// Read header
		header, err := reader.Read()
		if err != nil {
			utils.LogError("Failed to read CSV header: %v", err)
			return
		}

		utils.LogInfo("CSV header: %v", header)

		lineCount := 0
		for {
			record, err := reader.Read()
			if err != nil {
				// End of file
				utils.LogInfo("✅ Finished inserting %d lines from CSV", lineCount)
				break
			}

			// Parse CSV record (timestamp, open, high, low, close, volume)
			if len(record) < 6 {
				utils.LogError("Invalid CSV record: %v", record)
				continue
			}

			timestamp, err := strconv.ParseInt(record[0], 10, 64)
			if err != nil {
				utils.LogError("Failed to parse timestamp: %v", err)
				continue
			}

			open, _ := strconv.ParseFloat(record[1], 64)
			high, _ := strconv.ParseFloat(record[2], 64)
			low, _ := strconv.ParseFloat(record[3], 64)
			close, _ := strconv.ParseFloat(record[4], 64)
			volume, _ := strconv.ParseFloat(record[5], 64)

			// Create price data
			priceData := &db.PriceData{
				Timestamp: time.Unix(timestamp/1000, 0),
				Symbol:    "BTCUSDT",
				Open:      open,
				High:      high,
				Low:       low,
				Close:     close,
				Volume:    volume,
			}

			// Insert into database (this will trigger the channel notification)
			if err := db.InsertPrice(priceData); err != nil {
				utils.LogError("Failed to insert price: %v", err)
				continue
			}

			lineCount++
			utils.LogInfo("Inserted line %d: %s @ $%.2f", lineCount, priceData.Timestamp.Format(time.RFC3339), priceData.Close)

			// Sleep 1 second before next insert
			time.Sleep(1 * time.Second)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"status":  "started",
		"message": "Fake kline insertion started in background",
	})
}
