package api

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"spikeshield/db"
	"spikeshield/utils"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
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
		api.GET("/policies", s.handlePolicies)
		api.GET("/balance", s.handleBalance)
		api.POST("/balance/refresh", s.handleBalanceRefresh)
		api.POST("/insert_fake_kline", s.handleInsertFakeKline)
		api.POST("/wallet/link", s.handleWalletLink)
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

	user := c.Query("user")
	var (
		payouts []*db.Payout
		err     error
	)

	// Use unified GetRecentPayouts which accepts an optional user filter
	payouts, err = db.GetRecentPayouts(limit, user)

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
		// First, delete all existing spikes
		utils.LogInfo("🗑️  Deleting all existing spikes...")
		if err := db.DeleteAllSpikes(); err != nil {
			utils.LogError("Failed to delete spikes: %v", err)
			return
		}
		utils.LogInfo("✅ All spikes deleted")

		// Then delete all existing prices
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

			// Parse timestamp (ISO 8601 format)
			timestamp, err := time.Parse(time.RFC3339, record[0])
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
				Timestamp: timestamp,
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

// handleBalance returns token balance from DB (indexer). Defaults to USDT if token not provided.
func (s *Server) handleBalance(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address query parameter required"})
		return
	}

	token := c.DefaultQuery("token", utils.AppConfig.RPC.UsdtAddress)

	// Read from DB cache
	bal, err := db.GetBalanceForUser(token, address)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{
				"address": address,
				"token":   token,
				"found":   false,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch balance"})
		return
	}

	// Format using configured decimals if available
	decimals := utils.AppConfig.RPC.UsdtDecimals
	if decimals <= 0 {
		decimals = 6
	}
	format := fmt.Sprintf("%%.%df", decimals)
	formatted := fmt.Sprintf(format, bal.Balance)

	c.JSON(http.StatusOK, gin.H{
		"address":      bal.UserAddress,
		"token":        bal.TokenAddress,
		"balance":      formatted,
		"last_updated": bal.LastUpdated,
		"found":        true,
	})
}

// handleBalanceRefresh forces an on-chain read of the token balance and upserts to DB
func (s *Server) handleBalanceRefresh(c *gin.Context) {
	address := c.PostForm("address")
	if address == "" {
		// allow query param too
		address = c.Query("address")
	}
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address required"})
		return
	}

	token := c.DefaultQuery("token", utils.AppConfig.RPC.UsdtAddress)
	cfg := utils.AppConfig
	if cfg == nil || cfg.RPC.URL == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "RPC not configured"})
		return
	}

	client, err := ethclient.Dial(cfg.RPC.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect RPC"})
		return
	}
	defer client.Close()

	// Use minimal ERC20 ABI
	const erc20ABI = `[{"constant":true,"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"}]`
	parsed, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse ABI"})
		return
	}

	tokenAddr := common.HexToAddress(token)
	contract := bind.NewBoundContract(tokenAddr, parsed, client, client, client)

	var out []interface{}
	if err := contract.Call(&bind.CallOpts{Context: c.Request.Context()}, &out, "balanceOf", common.HexToAddress(address)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call balanceOf"})
		return
	}
	balanceRaw := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	// Convert to human float using configured decimals
	decimals := cfg.RPC.UsdtDecimals
	balf := new(big.Float).SetInt(balanceRaw)
	denom := new(big.Float).SetFloat64(math.Pow10(int(decimals)))
	human := new(big.Float).Quo(balf, denom)
	humanVal, _ := human.Float64()

	// Upsert into DB
	if err := db.UpsertBalance(token, address, humanVal); err != nil {
		utils.LogError("Failed to upsert balance: %v", err)
	}

	// Format according to configured decimals
	if cfg.RPC.UsdtDecimals <= 0 {
		cfg.RPC.UsdtDecimals = 6
	}
	format := fmt.Sprintf("%%.%df", cfg.RPC.UsdtDecimals)
	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"token":   token,
		"balance": fmt.Sprintf(format, humanVal),
	})
}

// handlePolicies returns policies for a given user address
func (s *Server) handlePolicies(c *gin.Context) {
	address := c.Query("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address query parameter required"})
		return
	}

	policies, err := db.GetPoliciesForUser(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch policies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":    len(policies),
		"policies": policies,
	})
}

// handleWalletLink triggers a background upsert of balances and policies for the linked wallet
func (s *Server) handleWalletLink(c *gin.Context) {
	var req struct {
		Address string `json:"address"`
		Token   string `json:"token"`
	}
	utils.LogInfo("wallet/link called address=%s token=%s", req.Address, req.Token)
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if req.Address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "address required"})
		return
	}

	// Run upsert in background to avoid blocking client
	go func(addr string) {
		if err := db.UpsertForUser(utils.AppConfig, addr); err != nil {
			utils.LogError("UpsertForUser failed for %s: %v", addr, err)
		} else {
			utils.LogInfo("UpsertForUser succeeded for %s", addr)
		}
	}(req.Address)

	c.JSON(http.StatusAccepted, gin.H{"status": "accepted"})
}
