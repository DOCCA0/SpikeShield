package api

import (
	"net/http"
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
