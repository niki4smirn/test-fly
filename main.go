package main

import (
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
)

type tradeMsg struct {
	Price string `json:"p"`
}

func main() {
	const wsURL = "wss://stream.binance.com:9443/ws/btcusdc@trade"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				log.Printf("read error: %v", err)
				return
			}

			var t tradeMsg
			if err := json.Unmarshal(msg, &t); err != nil {
				log.Printf("json error: %v", err)
				continue
			}

			log.Printf("BTC/USDC price: %s", t.Price)
		}
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			_ = conn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(
					websocket.CloseNormalClosure, "",
				),
			)
			return
		}
	}
}
