package config

const (
	//Wallet & API keys
	ApiKey      string = "" // api key to get the data from the website
	PrivateKey  string = "" // private key to sign the transactions
	PublicKey   string = "" // public key to get the data from the website
	RpcEndpoint string = "" // rpc endpoint to connect to the solana

	//Discord webhooks
	NotifWebhookURL string = "" // webhook to send notifications
	EventWebhookURL string = "" // webhook to send events
	ErrorWebhookURL string = "" // webhook to send errors

	//Trade configs
	IsLocalRpc        bool    = false  // use local rpc or pump fun jto rpc
	Retries           int     = 3      // number of retries on sell/buy (only if tx fail)
	GasFee            float64 = 0.001  // gas fee to take priority
	TradeSize         float64 = 0.005  // amount of sol on each buy
	MaxTrades         int     = 10     // max trades simultaneously
	Slippage          int     = 15     // slippage for the trades
	StopLoss          bool    = true   // true to activate stop loss mechanism
	TakeProfit        bool    = true   // true to activate take profit mechanism
	TakeProfitPcg     float64 = 200    // percentage to take profit
	StopLossPcg       float64 = 80     // percentage to sell on loss
	StopLossSellPcg   string  = "100%" // percentage of the position to liquidate on stop loss
	TakeProfitSellPcg string  = "80%"  // percentage of position to liquidate on take profit
	ThresholdBuy      float64 = 1.15   // threshold to buy
	ThresholdSell     float64 = 1.6    // threshold to sell

	//Filters
	TwitterFilter     bool = false // filter to check on x (twitter)
	TelegramFilter    bool = false // filter to check on telegram
	WebsiteFilter     bool = false // filter to check on website
	ImageFilter       bool = false // filter to check on image
	DescriptionFilter int  = 0     // filter min length of description (0 to disable)

	//Bot configs
	FileName     string = "records.txt" // path to persist the data (only if stop loss or take profit true)
	StartingMode string = "bot"         // "bot" or "cleaner" or "manager"
	// Bot starts the bot with the configs
	// Cleaner liquidates all the tokens on the wallet
	// Manager only manages the positions (no buy, but sells)

)

var WalletsToWatch = map[string]*struct {
	NewTokens bool
	Buy       bool
	Sell      bool
}{
	"samp13": { // address to watch
		NewTokens: false, // snipe new tokens
		Buy:       false, // copy trade to buy the same tokens
		Sell:      false, // copy trade to sell the same tokens
	},
	//...
}
