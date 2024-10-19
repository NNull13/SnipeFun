package bot

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"

	"GoSnipeFun/config"
)

var validate = validator.New()

func handleEvent(event TradeEvent) {
	var (
		err           error
		isHealthy     = true
		walletWatched = config.WalletsToWatch[event.TraderPublicKey]
		avoidFilters  = walletWatched != nil && walletWatched.AvoidFilters
	)

	isCreate := event.TxType == createEvent || event.TxType == initialBuyEvent

	// Perform Token Quality Check
	if isCreate {
		if isHealthy, err = performTokenQualityCheck(event, avoidFilters); err != nil {
			return
		}
	}

	// Determine Trade Type
	trade := Trade{Token: event.Mint}
	switch {
	case walletWatched != nil:
		switch {
		case event.TxType == sellEvent && walletWatched.Sell:
			trade.Type = sellType
		case event.TxType == buyEvent && walletWatched.Buy:
			trade.Type = buyType
		case isHealthy && isCreate && walletWatched.NewTokens:
			trade.Type = buyType
		}
	default:
		trade.Type = determineTradeType(event, isHealthy)
	}

	// Execute Trade
	if err = executeTradeRetry(trade, config.Retries); err != nil {
		return
	}

	// Post-Trade Processing
	processTrade(trade, event)
}

// performTokenQualityCheck checks the token's quality and sends notifications accordingly.
func performTokenQualityCheck(event TradeEvent, avoidFilters bool) (bool, error) {
	if healthChecking && !avoidFilters && event.MetaData != "" {
		tokenInfo, err := getTokenInfo(event.MetaData)
		if err != nil {
			log.Printf("getTokenInfo error: %v", err)
			return false, err
		}
		msg, isHealthy := tokenQualityCheck(tokenInfo)
		if isHealthy {
			sendNotification(Notification{
				Message: fmt.Sprintf(
					"**--- Token Quality Check %s ---**\n**Token:** `%s`\n%s",
					msg, event.Mint, tokenInfo.ToString(),
				),
				Type: infoNotification,
			})
		}
		return isHealthy, nil
	} else {
		sendNotification(Notification{
			Message: fmt.Sprintf(
				"**--- Token Quality Check Avoided ---**\n**Token:** `%s`\n",
				event.Mint,
			),
			Type: infoNotification,
		})
		return true, nil
	}
}

// determineTradeType decides the type of trade based on the event and current state.
func determineTradeType(event TradeEvent, isHealthy bool) string {
	switch event.TxType {
	case createEvent:
		if isHealthy && searching && event.shouldExecute() {
			return buyType
		}
	case sellEvent, buyEvent:
		record := watchedTokens[event.Mint]
		if managing {
			switch {
			case event.shouldExecute():
				return sellType
			case config.TakeProfit && event.MarketCapSol > record.MarketCap*config.TakeProfitPcg/100:
				return tpType
			case config.StopLoss && event.MarketCapSol < record.MarketCap*config.StopLossPcg/100:
				return sellType
			}
		}
	}
	return ""
}

// processTrade performs post-trade actions such as updating records and subscriptions.
func processTrade(trade Trade, event TradeEvent) {
	switch trade.Type {
	case buyType:
		if managing {
			record := Record{
				Address:   event.Mint,
				Timestamp: time.Now().Unix(),
				MarketCap: event.MarketCapSol,
			}
			watchedTokens[event.Mint] = record
			addLineToFile(record)
			subscribeTokenEvents(event.Mint)
		}
		if len(watchedTokens) >= config.MaxTrades {
			searching = false
		}
	case sellType:
		unsubscribeToken(event.Mint)
		go saveRecordsToFile(watchedTokens)
	case tpType:
		// Keep watching the token
	}
}

func liquidateAll() {
	for token := range watchedTokens {
		executeTradeRetry(Trade{Type: sellType, Token: token}, config.Retries)
	}
}

func tokenQualityCheck(t TokenInfo) (string, bool) {
	err := validate.Struct(t)
	if err != nil {
		return getSymbol(false), false
	}

	if config.ImageFilter {
		resp, err := http.Head(t.Image)
		if err != nil {
			return getSymbol(false) + " ImageFilter", false
		}
		defer resp.Body.Close()

		isImage := false
		switch resp.Header.Get("Content-Type") {
		case "image/jpeg", "image/png", "image/gif", "image/webp":
			isImage = true
		}
		if !isImage {
			return getSymbol(false) + " ImageFilter", false
		}
	}

	if config.TwitterFilter {
		isEmpty := strings.TrimSpace(t.Twitter) == ""
		isTwitter := strings.Contains(t.Twitter, "twitter.com") || strings.Contains(t.Twitter, "x.com")
		if isEmpty || !isTwitter {
			return getSymbol(false) + " TwitterFilter", false
		}
	}

	if config.TelegramFilter {
		isEmpty := strings.TrimSpace(t.Telegram) == ""
		isTelegram := strings.Contains(t.Telegram, "telegram") || strings.Contains(t.Telegram, "t.me")
		if isEmpty || !isTelegram {
			return getSymbol(false) + " TelegramFilter", false
		}
	}

	if config.WebsiteFilter && strings.TrimSpace(t.Website) == "" && validate.Var(t.Website, "url") != nil {
		return getSymbol(false) + " WebsiteFilter", false
	}

	if config.DescriptionFilter != 0 && len(t.Description) <= config.DescriptionFilter {
		return getSymbol(false) + " DescriptionFilter", false
	}

	return getSymbol(true), true
}
