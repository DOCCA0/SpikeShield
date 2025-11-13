# Visualization Scripts

## K-Line Visualization

Visualize candlestick (K-line) chart from PostgreSQL price data.

### Installation

```bash
pip install -r requirements.txt
```

### Usage

```bash
# Visualize latest 100 candles for BTCUSDT
python visualize_kline.py

# Specify symbol and limit
python visualize_kline.py BTCUSDT 50

# Different symbol
python visualize_kline.py ETHUSDT 200
```

### Features

- Displays candlestick chart with OHLC data
- Shows volume bars below the price chart
- Marks detected spikes with red triangles
- Customizable symbol and time range

### Requirements

- PostgreSQL running with `spikeshield` database
- Price data in `prices` table
- Python 3.8 or higher
