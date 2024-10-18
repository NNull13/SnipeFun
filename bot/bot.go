package bot

import (
	"GoSnipeFun/config"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func handleEvent(event TradeEvent) {
	var err error
	var tokenInfo TokenInfo
	isHealthy := true
	if healthChecking {
		tokenInfo, err = getTokenInfo(event.MetaData)
		if err != nil {
			log.Printf("getTokenInfo error: %v", err)
			return
		}
		var msg string
		msg, isHealthy = tokenQualityCheck(tokenInfo)
		if isHealthy {
			sendNotification(Notification{
				Message: "**--- Token Quality Check " + msg + " ---**\n" +
					"**Token:** `" + event.Mint + "`\n" +
					tokenInfo.ToString(), Type: infoNotification})
		}
	}

	walletWatched := config.WalletsToWatch[event.TraderPublicKey]
	trade := Trade{Token: event.Mint}

	switch {
	case isHealthy && walletWatched != nil:
		switch {
		case event.TxType == sellEvent && walletWatched.Sell:
			trade.Type = sellType
		case event.TxType == buyEvent && walletWatched.Buy:
			trade.Type = buyType
		case event.TxType == createEvent && walletWatched.NewTokens:
			trade.Type = buyType
		}
	case isHealthy && event.TxType == createEvent:
		if searching && event.shouldExecute() {
			trade.Type = buyType
		}
	case event.TxType == sellEvent, event.TxType == buyEvent:
		record := watchedTokens[event.Mint]
		if managing {
			switch {
			case event.shouldExecute():
				trade.Type = sellType
			case config.TakeProfit && event.MarketCapSol > record.MarketCap*config.TakeProfitPcg/100:
				trade.Type = tpType
			case config.StopLoss && event.MarketCapSol < record.MarketCap*config.StopLossPcg/100:
				trade.Type = sellType
			}
		}
	}

	if trade.Type == "" {
		return
	}

	if err = executeTradeRetry(trade, config.Retries); err == nil {
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
			// keep watching
		}
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
