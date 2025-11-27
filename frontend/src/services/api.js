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
  async getPayouts(limit = 50, user) {
    const userParam = user ? `&user=${user}` : '';
    const response = await fetch(`${API_BASE_URL}/api/payouts?limit=${limit}${userParam}`);
    if (!response.ok) throw new Error('Failed to fetch payouts');
    return response.json();
  },

  // Get user policies by address (server-side read)
  async getUserPolicies(address) {
    if (!address) return { count: 0, policies: [] };
    const response = await fetch(`${API_BASE_URL}/api/policies?address=${address}`);
    if (!response.ok) throw new Error('Failed to fetch user policies');
    return response.json();
  },

  // Get user token balance (server-side read of ERC20)
  async getUserBalance(address) {
    if (!address) return { address: '', balance: '0', raw: '0', decimals: 6 };
    const response = await fetch(`${API_BASE_URL}/api/balance?address=${address}`);
    if (!response.ok) throw new Error('Failed to fetch user balance');
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
  },

  // Link wallet to backend (triggers balance/policy sync)
  async linkWallet(address, token = null) {
    const body = { address };
    if (token) body.token = token;
    const response = await fetch(`${API_BASE_URL}/api/wallet/link`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    if (!response.ok) throw new Error(`Failed to link wallet: ${response.statusText}`);
    return response.json();
  }
};
