import React, { useState, useEffect } from 'react';
import './App.css';
import { useContract } from './hooks/useContract';
import { apiService } from './services/api';

function App() {
  const {
    account,
    loading,
    error,
    connectWallet,
    disconnectWallet,
    buyInsurance,
    mintTestUSDT,
    getBalance,
    getUserPolicies,
    hasActivePolicy,
    isConnected
  } = useContract();

  const [balance, setBalance] = useState('0');
  const [policies, setPolicies] = useState([]);
  const [hasActive, setHasActive] = useState(false);
  const [successMsg, setSuccessMsg] = useState('');
  const [refreshing, setRefreshing] = useState(false);
  
  // Backend API data
  const [spikes, setSpikes] = useState([]);
  const [prices, setPrices] = useState([]);
  const [payouts, setPayouts] = useState([]);
  const [stats, setStats] = useState(null);
  const [apiStatus, setApiStatus] = useState('checking');

  // Check API health
  useEffect(() => {
    const checkAPI = async () => {
      try {
        await apiService.healthCheck();
        setApiStatus('online');
        loadBackendData();
      } catch (err) {
        setApiStatus('offline');
        console.error('API health check failed:', err);
      }
    };
    checkAPI();
    const interval = setInterval(checkAPI, 5000); // Check every 5s
    return () => clearInterval(interval);
  }, []);

  // Load backend data
  const loadBackendData = async () => {
    try {
      const [spikesData, pricesData, payoutsData, statsData] = await Promise.all([
        apiService.getSpikes(10),
        apiService.getPrices('BTCUSDT', 20),
        apiService.getPayouts(10),
        apiService.getStats()
      ]);
      setSpikes(spikesData.spikes || []);
      setPrices(pricesData.prices || []);
      setPayouts(payoutsData.payouts || []);
      setStats(statsData.stats || null);
    } catch (err) {
      console.error('Failed to load backend data:', err);
    }
  };

  // Refresh data
  const refreshData = async () => {
    if (!isConnected) return;
    
    setRefreshing(true);
    try {
      const [bal, pols, active] = await Promise.all([
        getBalance(),
        getUserPolicies(),
        hasActivePolicy()
      ]);
      setBalance(bal);
      setPolicies(pols);
      setHasActive(active);
      
      // Also refresh backend data
      if (apiStatus === 'online') {
        await loadBackendData();
      }
    } catch (err) {
      console.error("Refresh error:", err);
    } finally {
      setRefreshing(false);
    }
  };

  // Auto refresh when connected
  useEffect(() => {
    if (isConnected) {
      refreshData();
    }
  }, [isConnected, account]);

  // Handle buy insurance
  const handleBuyInsurance = async () => {
    try {
      setSuccessMsg('');
      const txHash = await buyInsurance();
      setSuccessMsg(`Insurance purchased successfully! Tx: ${txHash.substring(0, 10)}...`);
      setTimeout(() => refreshData(), 2000);
    } catch (err) {
      console.error("Purchase failed:", err);
    }
  };

  // Handle mint USDT
  const handleMintUSDT = async () => {
    try {
      setSuccessMsg('');
      await mintTestUSDT(100);
      setSuccessMsg('Minted 100 test USDT successfully!');
      setTimeout(() => refreshData(), 2000);
    } catch (err) {
      console.error("Mint failed:", err);
    }
  };

  // Format date
  const formatDate = (date) => {
    return date.toLocaleString();
  };

  // Get policy status
  const getPolicyStatus = (policy) => {
    if (policy.claimed) return 'claimed';
    if (policy.expiryTime < new Date()) return 'expired';
    if (policy.active) return 'active';
    return 'inactive';
  };

  return (
    <div className="App">
      <div className="container">
        {/* Header */}
        <div className="header">
          <h1>🛡️ SpikeShield</h1>
          <p>Decentralized Spike Insurance Protocol</p>
        </div>

        {/* Wallet Section */}
        <div className="wallet-section">
          {!isConnected ? (
            <div>
              <h2>Connect Your Wallet</h2>
              <p>Connect your wallet to purchase spike insurance</p>
              <button 
                className="button" 
                onClick={connectWallet}
                disabled={loading}
              >
                {loading ? <span className="loading"></span> : 'Connect Wallet'}
              </button>
            </div>
          ) : (
            <div>
              <h2>Wallet Connected</h2>
              <div className="account-info">
                <div className="account-address">
                  {account.substring(0, 6)}...{account.substring(account.length - 4)}
                </div>
                <div className="balance-display">
                  {balance} USDT
                </div>
              </div>
              <button 
                className="button button-secondary" 
                onClick={handleMintUSDT}
                disabled={loading}
              >
                {loading ? <span className="loading"></span> : 'Mint 100 Test USDT'}
              </button>
              <button 
                className="button button-danger" 
                onClick={disconnectWallet}
              >
                Disconnect
              </button>
              <button 
                className="button" 
                onClick={refreshData}
                disabled={refreshing}
              >
                {refreshing ? <span className="loading"></span> : 'Refresh Data'}
              </button>
            </div>
          )}
        </div>

        {/* Error/Success Messages */}
        {error && (
          <div className="error-message">
            ❌ {error}
          </div>
        )}
        {successMsg && (
          <div className="success-message">
            ✅ {successMsg}
          </div>
        )}

        {/* Insurance Section */}
        {isConnected && (
          <div className="insurance-section">
            <h2>📋 Insurance Coverage</h2>
            <div className="insurance-info">
              <div className="info-card">
                <h3>Premium</h3>
                <div className="value">10 USDT</div>
              </div>
              <div className="info-card">
                <h3>Coverage Amount</h3>
                <div className="value">100 USDT</div>
              </div>
              <div className="info-card">
                <h3>Duration</h3>
                <div className="value">24 Hours</div>
              </div>
              <div className="info-card">
                <h3>Threshold</h3>
                <div className="value">10% Drop</div>
              </div>
            </div>

            <div style={{ margin: '30px 0' }}>
              <h3>How It Works</h3>
              <ul className="feature-list">
                <li>💰 Pay 10 USDT premium to get 100 USDT coverage</li>
                <li>⏱️ Protection valid for 24 hours</li>
                <li>📉 If BTC price drops ≥10% within 5 minutes, automatic payout</li>
                <li>⚡ Backend monitors price in real-time or replay mode</li>
                <li>🤖 Smart contract executes payout automatically</li>
              </ul>
            </div>

            <button 
              className="button" 
              onClick={handleBuyInsurance}
              disabled={loading || parseFloat(balance) < 10}
              style={{ fontSize: '1.3em', padding: '20px 40px' }}
            >
              {loading ? (
                <span className="loading"></span>
              ) : hasActive ? (
                '✅ Active Policy Exists'
              ) : (
                '🛡️ Buy Insurance (10 USDT)'
              )}
            </button>
            
            {parseFloat(balance) < 10 && (
              <p style={{ color: '#dc3545', marginTop: '10px' }}>
                ⚠️ Insufficient balance. Mint test USDT first.
              </p>
            )}
          </div>
        )}

        {/* Policies Section */}
        {isConnected && policies.length > 0 && (
          <div className="policies-section">
            <h2>📜 Your Insurance Policies</h2>
            <div className="policies-list">
              {policies.map((policy) => {
                const status = getPolicyStatus(policy);
                return (
                  <div key={policy.id} className={`policy-card ${status}`}>
                    <h4>
                      Policy #{policy.id}
                      <span 
                        className={`status-badge status-${status}`}
                        style={{ marginLeft: '10px' }}
                      >
                        {status}
                      </span>
                    </h4>
                    <div className="policy-detail">
                      <span className="policy-label">Premium:</span>
                      <span className="policy-value">{policy.premium} USDT</span>
                    </div>
                    <div className="policy-detail">
                      <span className="policy-label">Coverage:</span>
                      <span className="policy-value">{policy.coverage} USDT</span>
                    </div>
                    <div className="policy-detail">
                      <span className="policy-label">Purchased:</span>
                      <span className="policy-value">{formatDate(policy.purchaseTime)}</span>
                    </div>
                    <div className="policy-detail">
                      <span className="policy-label">Expires:</span>
                      <span className="policy-value">{formatDate(policy.expiryTime)}</span>
                    </div>
                    {policy.claimed && (
                      <div style={{ 
                        marginTop: '15px', 
                        padding: '10px', 
                        background: '#667eea', 
                        color: 'white', 
                        borderRadius: '5px',
                        textAlign: 'center',
                        fontWeight: 'bold'
                      }}>
                        🎉 Payout Executed!
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* Footer Info */}
        {!isConnected && (
          <div style={{ 
            background: 'rgba(255, 255, 255, 0.95)', 
            borderRadius: '20px', 
            padding: '30px',
            marginTop: '20px',
            textAlign: 'left'
          }}>
            <h2>🚀 About SpikeShield</h2>
            <p>
              SpikeShield is a decentralized insurance protocol that protects you against 
              sudden price drops (wicks) in cryptocurrency markets.
            </p>
            <h3>Features:</h3>
            <ul className="feature-list">
              <li>🔄 <strong>Replay Mode:</strong> Test with historical data</li>
              <li>⚡ <strong>Live Mode:</strong> Real-time monitoring using Chainlink Oracle</li>
              <li>🤖 <strong>Automatic Payout:</strong> Smart contract executes payout when wick detected</li>
              <li>🧪 <strong>Testnet Ready:</strong> Deploy on Sepolia or BSC Testnet</li>
            </ul>
          </div>
        )}

        {/* Backend Status and Stats */}
        <div style={{ 
          background: 'rgba(255, 255, 255, 0.95)', 
          borderRadius: '20px', 
          padding: '30px',
          marginTop: '20px'
        }}>
          <h2>
            📊 System Status 
            <span style={{ 
              marginLeft: '15px', 
              fontSize: '0.8em',
              padding: '5px 15px',
              borderRadius: '20px',
              background: apiStatus === 'online' ? '#48bb78' : '#f56565',
              color: 'white'
            }}>
              {apiStatus === 'online' ? '🟢 Online' : '🔴 Offline'}
            </span>
          </h2>
          
          {stats && (
            <div className="stats-grid" style={{ 
              display: 'grid', 
              gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
              gap: '15px',
              marginTop: '20px'
            }}>
              <div className="stat-card" style={{
                background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                color: 'white',
                padding: '20px',
                borderRadius: '15px'
              }}>
                <div style={{ fontSize: '2em', fontWeight: 'bold' }}>{stats.total_spikes}</div>
                <div>Total Wicks Detected</div>
              </div>
              <div className="stat-card" style={{
                background: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
                color: 'white',
                padding: '20px',
                borderRadius: '15px'
              }}>
                <div style={{ fontSize: '2em', fontWeight: 'bold' }}>{stats.total_payouts}</div>
                <div>Total Payouts</div>
              </div>
              <div className="stat-card" style={{
                background: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
                color: 'white',
                padding: '20px',
                borderRadius: '15px'
              }}>
                <div style={{ fontSize: '2em', fontWeight: 'bold' }}>{stats.active_policies}</div>
                <div>Active Policies</div>
              </div>
              <div className="stat-card" style={{
                background: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)',
                color: 'white',
                padding: '20px',
                borderRadius: '15px'
              }}>
                <div style={{ fontSize: '2em', fontWeight: 'bold' }}>{stats.total_prices}</div>
                <div>Price Records</div>
              </div>
            </div>
          )}
        </div>

        {/* Recent Wicks */}
        {apiStatus === 'online' && spikes.length > 0 && (
          <div style={{ 
            background: 'rgba(255, 255, 255, 0.95)', 
            borderRadius: '20px', 
            padding: '30px',
            marginTop: '20px'
          }}>
            <h2>⚡ Recent Wick Detections</h2>
            <div style={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', marginTop: '15px' }}>
                <thead>
                  <tr style={{ borderBottom: '2px solid #e2e8f0' }}>
                    <th style={{ padding: '12px', textAlign: 'left' }}>Time</th>
                    <th style={{ padding: '12px', textAlign: 'left' }}>Symbol</th>
                    <th style={{ padding: '12px', textAlign: 'right' }}>Before</th>
                    <th style={{ padding: '12px', textAlign: 'right' }}>After</th>
                    <th style={{ padding: '12px', textAlign: 'right' }}>Drop %</th>
                  </tr>
                </thead>
                <tbody>
                  {spikes.slice(0, 5).map((spike, idx) => (
                    <tr key={idx} style={{ borderBottom: '1px solid #e2e8f0' }}>
                      <td style={{ padding: '12px' }}>
                        {new Date(spike.timestamp).toLocaleString()}
                      </td>
                      <td style={{ padding: '12px' }}>{spike.symbol}</td>
                      <td style={{ padding: '12px', textAlign: 'right' }}>
                        ${spike.price_before?.toFixed(2)}
                      </td>
                      <td style={{ padding: '12px', textAlign: 'right' }}>
                        ${spike.price_after?.toFixed(2)}
                      </td>
                      <td style={{ padding: '12px', textAlign: 'right', color: '#f56565', fontWeight: 'bold' }}>
                        -{spike.drop_percent?.toFixed(2)}%
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Recent Prices */}
        {apiStatus === 'online' && prices.length > 0 && (
          <div style={{ 
            background: 'rgba(255, 255, 255, 0.95)', 
            borderRadius: '20px', 
            padding: '30px',
            marginTop: '20px'
          }}>
            <h2>📈 Recent Price Data</h2>
            <div style={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', marginTop: '15px' }}>
                <thead>
                  <tr style={{ borderBottom: '2px solid #e2e8f0' }}>
                    <th style={{ padding: '12px', textAlign: 'left' }}>Time</th>
                    <th style={{ padding: '12px', textAlign: 'right' }}>Open</th>
                    <th style={{ padding: '12px', textAlign: 'right' }}>High</th>
                    <th style={{ padding: '12px', textAlign: 'right' }}>Low</th>
                    <th style={{ padding: '12px', textAlign: 'right' }}>Close</th>
                  </tr>
                </thead>
                <tbody>
                  {prices.slice(0, 10).map((price, idx) => (
                    <tr key={idx} style={{ borderBottom: '1px solid #e2e8f0' }}>
                      <td style={{ padding: '12px' }}>
                        {new Date(price.timestamp).toLocaleString()}
                      </td>
                      <td style={{ padding: '12px', textAlign: 'right' }}>${price.open?.toFixed(2)}</td>
                      <td style={{ padding: '12px', textAlign: 'right' }}>${price.high?.toFixed(2)}</td>
                      <td style={{ padding: '12px', textAlign: 'right' }}>${price.low?.toFixed(2)}</td>
                      <td style={{ padding: '12px', textAlign: 'right' }}>${price.close?.toFixed(2)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Payout History */}
        {apiStatus === 'online' && payouts.length > 0 && (
          <div style={{ 
            background: 'rgba(255, 255, 255, 0.95)', 
            borderRadius: '20px', 
            padding: '30px',
            marginTop: '20px'
          }}>
            <h2>💰 Payout History</h2>
            <div style={{ overflowX: 'auto' }}>
              <table style={{ width: '100%', borderCollapse: 'collapse', marginTop: '15px' }}>
                <thead>
                  <tr style={{ borderBottom: '2px solid #e2e8f0' }}>
                    <th style={{ padding: '12px', textAlign: 'left' }}>Time</th>
                    <th style={{ padding: '12px', textAlign: 'left' }}>User</th>
                    <th style={{ padding: '12px', textAlign: 'right' }}>Amount</th>
                    <th style={{ padding: '12px', textAlign: 'left' }}>Tx Hash</th>
                  </tr>
                </thead>
                <tbody>
                  {payouts.slice(0, 5).map((payout, idx) => (
                    <tr key={idx} style={{ borderBottom: '1px solid #e2e8f0' }}>
                      <td style={{ padding: '12px' }}>
                        {new Date(payout.executed_at).toLocaleString()}
                      </td>
                      <td style={{ padding: '12px' }}>
                        {payout.user_address.substring(0, 6)}...{payout.user_address.substring(38)}
                      </td>
                      <td style={{ padding: '12px', textAlign: 'right', color: '#48bb78', fontWeight: 'bold' }}>
                        ${payout.amount?.toFixed(2)}
                      </td>
                      <td style={{ padding: '12px' }}>
                        {payout.tx_hash ? (
                          <a href={`https://sepolia.etherscan.io/tx/${payout.tx_hash}`} 
                             target="_blank" 
                             rel="noopener noreferrer"
                             style={{ color: '#667eea' }}>
                            {payout.tx_hash.substring(0, 10)}...
                          </a>
                        ) : 'Pending'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default App;
