package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type tradeMsg struct {
	Price string `json:"p"`
}

type stats struct {
	sync.Mutex
	n    int64
	mean float64
	m2   float64
}

func (s *stats) update(x float64) float64 {
	s.Lock()
	defer s.Unlock()
	s.n++
	if s.n == 1 {
		s.mean = x
		return 0
	}
	delta := x - s.mean
	s.mean += delta / float64(s.n)
	delta2 := x - s.mean
	s.m2 += delta * delta2
	return math.Sqrt(s.m2 / float64(s.n-1))
}

// main starts a Prometheus metrics exporter that connects to the Binance BTC/USDC trade WebSocket stream, processes real-time trade prices, and exposes the latest price, trade count, and price volatility as metrics on port 2112.
func main() {
	datapoints := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "btc_usdc_datapoints",
			Help: "number of trade messages",
		},
	)
	priceGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "btc_usdc_price",
			Help: "latest BTC/USDC price",
		},
	)
	volGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "btc_usdc_volatility",
			Help: "sample standard deviation of price",
		},
	)
	prometheus.MustRegister(datapoints, priceGauge, volGauge)
	go http.ListenAndServe(":2112", promhttp.Handler())

	conn, _, err := websocket.DefaultDialer.Dial(
		"wss://stream.binance.com:9443/ws/btcusdc@trade",
		nil,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	var st stats
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			continue
		}
		var t tradeMsg
		if json.Unmarshal(msg, &t) != nil {
			continue
		}
		p, err := strconv.ParseFloat(t.Price, 64)
		if err != nil {
			continue
		}
		datapoints.Inc()
		priceGauge.Set(p)
		volGauge.Set(st.update(p))
	}
}
