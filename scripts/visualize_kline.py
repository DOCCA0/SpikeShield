#!/usr/bin/env python3
"""
Visualize K-line (Candlestick) data from PostgreSQL database
"""

import psycopg2
import pandas as pd
import matplotlib.pyplot as plt
import mplfinance as mpf
from datetime import datetime
import sys

# Database configuration
DB_CONFIG = {
    'host': 'localhost',
    'port': 5432,
    'user': 'postgres',
    'password': 'postgres',
    'database': 'spikeshield'
}

def fetch_price_data(symbol='BTCUSDT', limit=100):
    """Fetch price data from PostgreSQL"""
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        cursor = conn.cursor()
        
        query = """
            SELECT timestamp, open, high, low, close, volume
            FROM prices
            WHERE symbol = %s
            ORDER BY timestamp DESC
            LIMIT %s
        """
        
        cursor.execute(query, (symbol, limit))
        rows = cursor.fetchall()
        
        if not rows:
            print(f"No data found for symbol {symbol}")
            return None
        
        # Create DataFrame
        df = pd.DataFrame(rows, columns=['Date', 'Open', 'High', 'Low', 'Close', 'Volume'])
        df['Date'] = pd.to_datetime(df['Date'])
        df = df.set_index('Date')
        df = df.sort_index()  # Sort by date ascending for proper visualization
        
        # Convert to float
        df['Open'] = df['Open'].astype(float)
        df['High'] = df['High'].astype(float)
        df['Low'] = df['Low'].astype(float)
        df['Close'] = df['Close'].astype(float)
        df['Volume'] = df['Volume'].astype(float)
        
        cursor.close()
        conn.close()
        
        return df
        
    except Exception as e:
        print(f"Error fetching data: {e}")
        return None

def fetch_spikes(symbol='BTCUSDT'):
    """Fetch spike data to overlay on chart"""
    try:
        conn = psycopg2.connect(**DB_CONFIG)
        cursor = conn.cursor()
        
        query = """
            SELECT timestamp, price_before, price_after, drop_percent
            FROM spikes
            WHERE symbol = %s
            ORDER BY timestamp
        """
        
        cursor.execute(query, (symbol,))
        rows = cursor.fetchall()
        
        cursor.close()
        conn.close()
        
        return rows
        
    except Exception as e:
        print(f"Error fetching spikes: {e}")
        return []

def plot_candlestick(df, symbol='BTCUSDT', spikes=None, save_path='kline_chart.png'):
    """Plot candlestick chart using mplfinance"""
    
    # Create custom style
    mc = mpf.make_marketcolors(up='#26a69a', down='#ef5350', inherit=True)
    s = mpf.make_mpf_style(base_mpf_style='charles', marketcolors=mc)
    
    # Prepare spike markers if available
    apds = []
    if spikes:
        spike_times = [pd.Timestamp(s[0]) for s in spikes]
        spike_prices = [float(s[2]) for s in spikes]  # price_after
        
        # Create a series aligned with the DataFrame index
        spike_series = pd.Series(index=df.index, dtype=float)
        for st, sp in zip(spike_times, spike_prices):
            # Find the closest timestamp in the DataFrame
            if st in df.index:
                spike_series[st] = sp
        
        # Only add plot if we have valid spike data
        if spike_series.notna().any():
            apds.append(mpf.make_addplot(spike_series, type='scatter', markersize=100, 
                                         marker='v', color='red', panel=0))
    
    # Plot
    title = f'{symbol} K-Line Chart (Latest {len(df)} candles)'
    
    if apds:
        mpf.plot(df, type='candle', style=s, title=title, 
                volume=True, ylabel='Price (USD)', 
                ylabel_lower='Volume',
                addplot=apds,
                savefig=save_path,
                warn_too_much_data=len(df)+1)
    else:
        mpf.plot(df, type='candle', style=s, title=title, 
                volume=True, ylabel='Price (USD)', 
                ylabel_lower='Volume',
                savefig=save_path,
                warn_too_much_data=len(df)+1)
    
    return save_path

def main():
    symbol = 'BTCUSDT'
    limit = 100
    
    # Parse command line arguments
    if len(sys.argv) > 1:
        symbol = sys.argv[1]
    if len(sys.argv) > 2:
        limit = int(sys.argv[2])
    
    print(f"Fetching {limit} candles for {symbol}...")
    df = fetch_price_data(symbol, limit)
    
    if df is None or df.empty:
        print("No data to visualize")
        return
    
    print(f"Fetched {len(df)} records")
    print(f"Date range: {df.index[0]} to {df.index[-1]}")
    
    # Fetch spikes
    spikes = fetch_spikes(symbol)
    if spikes:
        print(f"Found {len(spikes)} spike events")
    
    # Plot
    print("Generating candlestick chart...")
    output_file = f'{symbol}_kline.png'
    saved_path = plot_candlestick(df, symbol, spikes, output_file)
    
    print(f"Chart saved to: {saved_path}")

if __name__ == '__main__':
    main()
