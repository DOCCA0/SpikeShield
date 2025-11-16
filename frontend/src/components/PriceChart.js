import React, { useEffect, useRef } from 'react';
import { createChart } from 'lightweight-charts';

const PriceChart = ({ prices }) => {
  const chartContainerRef = useRef();
  const chartRef = useRef();
  const candlestickSeriesRef = useRef();

  useEffect(() => {
    if (!chartContainerRef.current) return;

    // Create chart
    const chart = createChart(chartContainerRef.current, {
      width: chartContainerRef.current.clientWidth,
      height: 400,
      layout: {
        backgroundColor: '#ffffff',
        textColor: '#333',
      },
      grid: {
        vertLines: { color: '#f0f0f0' },
        horzLines: { color: '#f0f0f0' },
      },
      timeScale: {
        timeVisible: true,
        secondsVisible: false,
      },
      localization: {
        locale: 'en-US',
      },
    });

    // Create candlestick series
    const candlestickSeries = chart.addCandlestickSeries({
      upColor: '#26a69a',
      downColor: '#ef5350',
      borderVisible: false,
      wickUpColor: '#26a69a',
      wickDownColor: '#ef5350',
    });

    chartRef.current = chart;
    candlestickSeriesRef.current = candlestickSeries;

    // Handle resize
    const handleResize = () => {
      if (chartContainerRef.current && chartRef.current) {
        chartRef.current.applyOptions({
          width: chartContainerRef.current.clientWidth,
        });
      }
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      if (chartRef.current) {
        chartRef.current.remove();
      }
    };
  }, []);

  useEffect(() => {
    if (!candlestickSeriesRef.current || !prices || prices.length === 0) return;

    // Convert prices to candlestick data
    const candleData = prices
      .map((price) => ({
        time: Math.floor(new Date(price.Timestamp).getTime() / 1000),
        open: price.Open,
        high: price.High,
        low: price.Low,
        close: price.Close,
      }))
      .sort((a, b) => a.time - b.time); // Sort by time ascending

    candlestickSeriesRef.current.setData(candleData);
    chartRef.current.timeScale().fitContent();
  }, [prices]);

  return (
    <div
      ref={chartContainerRef}
      style={{
        width: '100%',
        height: '400px',
        marginTop: '15px',
      }}
    />
  );
};

export default PriceChart;
