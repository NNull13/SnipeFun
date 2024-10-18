package bot

import (
	"fmt"

	"GoSnipeFun/config"
)

type TradeEvent struct {
	Signature             string  `json:"signature"`
	Mint                  string  `json:"mint"`
	TraderPublicKey       string  `json:"traderPublicKey"`
	TxType                string  `json:"txType"`
	TokenAmount           float64 `json:"tokenAmount"`
	NewTokenBalance       float64 `json:"newTokenBalance"`
	BondingCurveKey       string  `json:"bondingCurveKey"`
	VTokensInBondingCurve float64 `json:"vTokensInBondingCurve"`
	VSolInBondingCurve    float64 `json:"vSolInBondingCurve"`
	MarketCapSol          float64 `json:"marketCapSol"`
	Name                  string  `json:"name"`
	Symbol                string  `json:"symbol"`
	MetaData              string  `json:"uri"`
	Description           string  `json:"description"`
	CreatedOn             string  `json:"createdOn"`
	Twitter               string  `json:"twitter"`
	Telegram              string  `json:"telegram"`
	Website               string  `json:"website"`
	ShowName              bool    `json:"showName"`
}

func (e TradeEvent) ToString() (msg string) {
	riskSymbol := getSymbol(e.shouldExecute())
	switch e.TxType {
	case createEvent:
		msg = fmt.Sprintf("**--- Token Event %s ---**\n"+
			"**Token:** `%s`\n"+
			"**Transaction Type:** `%s`\n"+
			"**Trader Public Key:** `%s`\n"+
			"**VTokens In Bonding Curve:** `%.2f`\n"+
			"**VSol In Bonding Curve:** `%.2f`\n"+
			"**Market Cap (Sol):** `%.2f`",
			riskSymbol, e.Mint, e.TxType, e.TraderPublicKey,
			e.VTokensInBondingCurve, e.VSolInBondingCurve, e.MarketCapSol)
	case sellEvent, buyEvent:
		msg = fmt.Sprintf("**--- Trader Event %s ---**\n"+
			"**Token:** `%s`\n"+
			"**Transaction Type:** `%s`\n"+
			"**Trader Public Key:** `%s`\n"+
			"**Token Amount:** `%.2f`\n"+
			"**New Token Balance:** `%.2f`\n"+
			"**VTokens In Bonding Curve:** `%.2f`\n"+
			"**VSol In Bonding Curve:** `%.2f`\n"+
			"**Market Cap (Sol):** `%.2f`"+
			"**Name:** `%s`\n"+
			"**Symbol:** `%s`\n"+
			"**Description:** `%s`\n",
			riskSymbol, e.Mint, e.TxType, e.TraderPublicKey, e.TokenAmount, e.NewTokenBalance,
			e.VTokensInBondingCurve, e.VSolInBondingCurve, e.MarketCapSol)
	}
	return msg
}

func (e TradeEvent) shouldExecute() bool {
	ratioIncrement := e.MarketCapSol / e.VSolInBondingCurve
	return ratioIncrement > config.ThresholdSell || ratioIncrement < config.ThresholdBuy
}

type Trade struct {
	Token string `json:"token"`
	Type  string `json:"type"`
}

const (
	sellType = "sell"
	buyType  = "buy"
	tpType   = "takeProfit"
)

func (t Trade) ToString() (msg string) {
	msg = fmt.Sprintf("**--- Trade ---**\n"+
		"**Token:** `%s`\n"+
		"**Type:** `%s`", t.Token, t.Type)
	return msg
}

type Record struct {
	Address   string
	Timestamp int64
	MarketCap float64
}

type Notification struct {
	Message string           `json:"message"`
	Type    NotificationType `json:"type"`
}

type DiscordMessage struct {
	Content string `json:"content"`
}

type NotificationType int

const (
	eventNotification NotificationType = iota
	infoNotification
	errorNotification
)

type TokenInfo struct {
	Name        string `json:"name" validate:"required"`
	Symbol      string `json:"symbol" validate:"required"`
	Description string `json:"description" validate:"required"`
	Image       string `json:"image" validate:"required,url"`
	CreatedOn   string `json:"createdOn" validate:"required,url"`
	Twitter     string `json:"twitter"`
	Telegram    string `json:"telegram"`
	Website     string `json:"website"`
	ShowName    bool   `json:"showName"`
}

func (t TokenInfo) ToString() string {
	return fmt.Sprintf(
		"**Name:** `%s`\n"+
			"**Symbol:** `%s`\n"+
			"**Description:** `%s`\n"+
			"**Twitter:** `%s`\n"+
			"**Telegram:** `%s`\n"+
			"**Website:** `%s`\n"+
			"**Image:** `%s`\n"+
			"**Created On:** `%s`",
		t.Name, t.Symbol, t.Description, t.Website,
		t.Twitter, t.Telegram, t.Image, t.CreatedOn)
}
