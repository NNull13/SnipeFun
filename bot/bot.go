package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"GoSnipeFun/config"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func handleEvent(event TradeEvent) {
	walletWatched := config.WalletsToWatch[event.TraderPublicKey]

	switch event.TxType {
	case createEvent:
		if searching && ((walletWatched != nil && walletWatched.NewTokens) || event.shouldExecute()) {
			err := executeTradeRetry(Trade{Type: buyType, Token: event.Mint}, config.Retries)
			if managing && err == nil {
				record := Record{
					Address:   event.Mint,
					Timestamp: time.Now().Unix(),
					MarketCap: event.MarketCapSol,
				}
				watchedTokens[event.Mint] = record
				addLineToFile(record)
				subscribeTokenEvents(event.Mint)
			}
		}
		if len(watchedTokens) >= config.MaxTrades {
			searching = false
		}
	case sellEvent, buyEvent:
		record := watchedTokens[event.Mint]
		if managing {
			typeTrade := ""
			switch {
			case walletWatched != nil:
				switch {
				case walletWatched.Sell && event.TxType == sellEvent:
					typeTrade = sellType
				case walletWatched.Buy && event.TxType == buyEvent:
					typeTrade = buyType
				}
			case event.shouldExecute():
				typeTrade = sellType
			case config.TakeProfit && event.MarketCapSol > record.MarketCap*config.TakeProfitPcg/100:
				typeTrade = tpType
			case config.StopLoss && event.MarketCapSol < record.MarketCap*config.StopLossPcg/100:
				typeTrade = sellType
			}

			if typeTrade != "" {
				err := executeTradeRetry(Trade{Type: sellType, Token: event.Mint}, config.Retries)
				if err == nil {
					unsubscribeToken(event.Mint)
					go saveRecordsToFile(watchedTokens)
				}
			}
		}
	}
}

func liquidateAll() {
	for token := range watchedTokens {
		executeTradeRetry(Trade{Type: sellType, Token: token}, config.Retries)
	}
}

func fetchTokenInfoRetry(token string, retries int) (*TokenInfo, error) {
	var err error
	var tokenInfo *TokenInfo
	for i := 0; i < retries; i++ {
		tokenInfo, err = fetchTokenInfo(token)
		if err == nil {
			return tokenInfo, nil
		}
	}
	return nil, err
}

func fetchTokenInfo(token string) (*TokenInfo, error) {
	url := fmt.Sprintf("https://pumpportal.fun/api/data/token-info?ca=%s", token)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var tokenInfo TokenInfo
	if err = json.Unmarshal(body, &tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return &tokenInfo, nil
}

func tokenQualityCheck(t TokenInfo) (string, bool) {
	err := validate.Struct(t)
	if err != nil {
		return getSymbol(false), false
	}

	if config.ImageFilter {
		resp, err := http.Head(t.Data.Image)
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
		isEmpty := strings.TrimSpace(t.Data.Twitter) == ""
		isTwitter := strings.Contains(t.Data.Twitter, "twitter.com") || strings.Contains(t.Data.Twitter, "x.com")
		if isEmpty || !isTwitter {
			return getSymbol(false) + " TwitterFilter", false
		}
	}

	if config.TelegramFilter {
		isEmpty := strings.TrimSpace(t.Data.Telegram) == ""
		isTelegram := strings.Contains(t.Data.Telegram, "telegram") || strings.Contains(t.Data.Telegram, "t.me")
		if isEmpty || !isTelegram {
			return getSymbol(false) + " TelegramFilter", false
		}
	}

	if config.WebsiteFilter && strings.TrimSpace(t.Data.Website) == "" && validate.Var(t.Data.Website, "url") != nil {
		return getSymbol(false) + " WebsiteFilter", false
	}

	if config.DescriptionFilter != 0 && len(t.Data.Description) <= config.DescriptionFilter {
		return getSymbol(false) + " DescriptionFilter", false
	}

	return getSymbol(true), true
}
