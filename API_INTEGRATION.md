# Backend API Integration Complete

## ✅ What Was Added

### Backend REST API Endpoints

Created `/backend/api/server.go` with the following endpoints:

1. **`GET /api/health`** - Health check
   - Returns: `{"status": "healthy", "time": "..."}`

2. **`GET /api/spikes?limit=50`** - Recent wick detections
   - Returns: List of detected wick events with timestamps, prices, and drop percentages

3. **`GET /api/prices?symbol=BTCUSDT&limit=100`** - Historical price data
   - Returns: OHLCV candlestick data

4. **`GET /api/payouts?limit=50`** - Payout history
   - Returns: List of executed payouts with user addresses and amounts

5. **`GET /api/stats`** - System statistics
   - Returns: Total wicks, payouts, policies, active policies, and price records

### Database Functions

Added to `/backend/db/db.go`:

- `GetRecentSpikes(limit)` - Fetch recent wick detections
- `GetRecentPrices(symbol, limit)` - Fetch recent price data
- `GetRecentPayouts(limit)` - Fetch payout history
- `GetSystemStats()` - Get system statistics

### Frontend Integration

1. **API Service** (`/frontend/src/services/api.js`)
   - Centralized API client with methods for all endpoints
   - Environment-based configuration

2. **Updated App.js**
   - Added API health monitoring
   - Auto-refresh every 30 seconds
   - Displays:
     - ✅ Real-time system status (Online/Offline)
     - 📊 System statistics dashboard
     - ⚡ Recent wick detections table
     - 📈 Recent price data table
     - 💰 Payout history table

## 🚀 How to Run

### Backend
```bash
cd backend
go run main.go -mode live -api-port 8080
```

The API server starts automatically on port 8080 alongside the wick detection system.

### Frontend
```bash
cd frontend
npm start
```

Frontend connects to `http://localhost:8080` by default.

## 📊 API Response Examples

### Stats
```json
{
  "stats": {
    "total_spikes": 2,
    "total_payouts": 0,
    "total_policies": 0,
    "active_policies": 0,
    "total_prices": 34
  },
  "latest_price": {
    "Symbol": "BTCUSDT",
    "Close": 103133.30,
    "Timestamp": "2025-11-13T14:00:00Z"
  },
  "status": "monitoring"
}
```

### Spikes
```json
{
  "count": 2,
  "spikes": [
    {
      "id": 9,
      "timestamp": "2024-01-01T01:25:00Z",
      "symbol": "BTCUSDT",
      "price_before": 45880.00,
      "price_after": 40500.00,
      "drop_percent": 11.73,
      "detected_at": "2025-11-13T14:17:00Z"
    }
  ]
}
```

## 🔧 Configuration

### Backend
API port can be configured with `-api-port` flag (default: 8080)

### Frontend
Create `/frontend/.env`:
```
REACT_APP_API_URL=http://localhost:8080
```

## ✨ Features

- ✅ CORS enabled for cross-origin requests
- ✅ Real-time data updates
- ✅ Health monitoring
- ✅ Responsive UI with statistics dashboard
- ✅ Historical data visualization
- ✅ Payout tracking

## 🔄 Data Flow

```
Backend (Go) ←→ PostgreSQL
     ↓
  REST API (:8080)
     ↓
Frontend (React) ←→ MetaMask/Web3
```

The frontend now integrates both:
1. **Blockchain data** (via Web3/ethers.js)
2. **Backend monitoring data** (via REST API)
