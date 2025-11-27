import React, { useEffect, useState } from 'react';
import { useContract } from '../hooks/useContract';
import { apiService } from '../services/api';

/**
 * PayoutNotification component
 * Listens for payout events and displays notifications when user receives payouts
 */
const PayoutNotification = () => {
  const { account, isConnected } = useContract();
  const [payouts, setPayouts] = useState([]);

  useEffect(() => {
    if (!isConnected || !account) return;

    let mounted = true;

    const fetchPayouts = async () => {
      try {
        const resp = await apiService.getPayouts(10, account);
        const items = resp?.payouts || [];
        if (!mounted) return;

        // Prepend new items that are not in the current list
        setPayouts(prev => {
          const existing = new Set(prev.map(p => p.tx_hash || p.TxHash));
          const newOnes = items.filter(i => !existing.has(i.tx_hash || i.TxHash));
          const normalized = newOnes.map(i => ({
            txHash: i.tx_hash || i.TxHash || '',
            amount: i.amount || i.Amount || 0,
            policyId: i.policy_id || i.PolicyID || 0,
            blockNumber: i.block_number || i.executed_at || ''
          }));
          // Show notifications for new items
          normalized.forEach(payoutData => {
            if ('Notification' in window && Notification.permission === 'granted') {
              new Notification('💰 Insurance Payout Received!', {
                body: `You received ${payoutData.amount} USDT for policy #${payoutData.policyId}`,
                icon: '/logo192.png'
              });
            }
          });
          return [...normalized, ...prev];
        });
      } catch (err) {
        console.error('Failed to fetch payouts for user:', err);
      }
    };

    // Initial fetch and polling
    fetchPayouts();
    const iv = setInterval(fetchPayouts, 10000);

    // Request notification permission (once)
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission();
    }

    return () => {
      mounted = false;
      clearInterval(iv);
    };
  }, [isConnected, account]);

  if (!isConnected || payouts.length === 0) return null;

  return (
    <div style={{
      position: 'fixed',
      top: '20px',
      right: '20px',
      zIndex: 1000,
      maxWidth: '400px'
    }}>
      {payouts.map((payout, index) => (
        <div
          key={`${payout.txHash}-${index}`}
          style={{
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            color: 'white',
            padding: '15px',
            borderRadius: '10px',
            marginBottom: '10px',
            boxShadow: '0 4px 6px rgba(0,0,0,0.1)',
            animation: 'slideIn 0.3s ease-out'
          }}
        >
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: '8px' }}>
            <span style={{ fontSize: '24px', marginRight: '10px' }}>🎉</span>
            <strong>Payout Received!</strong>
          </div>
          <div style={{ fontSize: '14px', opacity: 0.9 }}>
            Amount: <strong>{payout.amount} USDT</strong>
          </div>
          <div style={{ fontSize: '12px', opacity: 0.8, marginTop: '5px' }}>
            Policy #{payout.policyId} | Block #{payout.blockNumber}
          </div>
        </div>
      ))}
    </div>
  );
};

export default PayoutNotification;
