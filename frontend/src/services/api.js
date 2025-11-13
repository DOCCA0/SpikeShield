const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

export const apiService = {
  // Get recent wick detection events
  async getSpikes(limit = 50) {
    const response = await fetch(`${API_BASE_URL}/api/spikes?limit=${limit}`);
    if (!response.ok) throw new Error('Failed to fetch spikes');
    return response.json();
  },

  // Get recent price data
  async getPrices(symbol = 'BTCUSDT', limit = 100) {
    const response = await fetch(`${API_BASE_URL}/api/prices?symbol=${symbol}&limit=${limit}`);
    if (!response.ok) throw new Error('Failed to fetch prices');
    return response.json();
  },

  // Get payout history
  async getPayouts(limit = 50) {
    const response = await fetch(`${API_BASE_URL}/api/payouts?limit=${limit}`);
    if (!response.ok) throw new Error('Failed to fetch payouts');
    return response.json();
  },

  // Get system statistics
  async getStats() {
    const response = await fetch(`${API_BASE_URL}/api/stats`);
    if (!response.ok) throw new Error('Failed to fetch stats');
    return response.json();
  },

  // Health check
  async healthCheck() {
    const response = await fetch(`${API_BASE_URL}/api/health`);
    if (!response.ok) throw new Error('API server is down');
    return response.json();
  }
};
