package bot

import (
	"encoding/json"
	"log"

	"GoSnipeFun/config"
)

var (
	searching bool
	managing  = (config.StopLoss || config.TakeProfit)
)

const (
	createEvent = "create"
	sellEvent   = "sell"
	buyEvent    = "buy"
)

func getWatchedTokenKeys() []string {
	keys := make([]string, 0, len(watchedTokens))
	for k := range watchedTokens {
		keys = append(keys, k)
	}
	return keys
}

func getWatchedWalletsKeys() []string {
	keys := make([]string, 0, len(watchedTokens))
	for k, _ := range config.WalletsToWatch {
		keys = append(keys, k)
	}
	return keys
}

func subscribeToEvents(onlyManage bool) {
	watchedTokensKeys := getWatchedTokenKeys()
	watchedWalletsKey := getWatchedWalletsKeys()
	subscriptions := []map[string]any{
		{"method": "subscribeTokenTrade", "keys": watchedTokensKeys},
		{"method": "subscribeAccountTrade ", "keys": watchedWalletsKey},
	}
	if !onlyManage {
		subscriptions = append(subscriptions, map[string]any{"method": "subscribeNewToken"})
	}

	for _, sub := range subscriptions {
		if err := conn.WriteJSON(sub); err != nil {
			log.Printf("subscribe error: %v", err)
		}
	}

	messageChan := make(chan []byte, 5)

	go processMessage(messageChan)

	readMessages(messageChan)

}

func readMessages(messageChan chan []byte) {
	var err error
	for {
		var message []byte
		if _, message, err = conn.ReadMessage(); err != nil {
			log.Printf("read error: %v", err)
			continue
		}
		messageChan <- message
	}
}

func processMessage(messageChan chan []byte) {
	for message := range messageChan {
		var event TradeEvent
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("unmarshal error: %v", err)
			continue
		}

		if event.Mint == "" {
			log.Printf("message is not a trade event: %s", message)
			continue
		}

		log.Printf("event: %+v", event)
		sendNotification(Notification{Message: event.ToString(), Type: eventNotification})

		tokenData, err := fetchTokenInfoRetry(event.Mint, config.Retries)
		if tokenData == nil || err != nil {
			continue
		}

		msg, isHealthy := tokenQualityCheck(*tokenData)
		if isHealthy {
			sendNotification(Notification{
				Message: "**--- Token Quality Check " + msg + " ---**\n" +
					"**Token:** `" + event.Mint + "`\n" +
					tokenData.ToString(), Type: infoNotification})
			handleEvent(event)
		} else {
			sendNotification(Notification{
				Message: "**--- Token Quality Check " + msg + " ---**\n" +
					"**Token:** `" + event.Mint + "`\n", Type: infoNotification})
		}
	}
}

func subscribeTokenEvents(address string) {
	subscription := map[string]any{"method": "subscribeTokenTrade", "keys": []string{address}}

	if err := conn.WriteJSON(subscription); err != nil {
		log.Printf("subscribe error: %v", err)
	}
}

func unsubscribeToken(mint string) {
	unsub := map[string]any{
		"method": "unsubscribeTokenTrade",
		"keys":   []string{mint},
	}

	if err := conn.WriteJSON(unsub); err != nil {
		log.Printf("unsubscribe error: %v", err)
	}

	delete(watchedTokens, mint)
	if len(watchedTokens) < config.MaxTrades {
		searching = true
	}
}
