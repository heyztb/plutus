package routes

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/awnumar/memguard"
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcutil"
	"github.com/heyztb/plutus/plutus"
	"github.com/monero-ecosystem/go-monero-rpc-client/wallet"
)

// BitcoinRPCClient is a reference to the rpc client used by this service to talk to btcd/btcwallet
var BitcoinRPCClient *rpcclient.Client

// MoneroRPCClient is a pointer to the rpc client used by this service to talk to monero-wallet-rpc
var MoneroRPCClient wallet.Client
var netParams *chaincfg.Params

func init() {

	// based on the value of this environment variable, we want to use different net paramaters for encoding and decoding information sent/received to/by btcd/btcwallet
	switch strings.ToLower(os.Getenv("BITCOIN_CHAIN")) {
	case "main":
		netParams = &chaincfg.MainNetParams
	case "test":
		netParams = &chaincfg.TestNet3Params
	case "reg":
		netParams = &chaincfg.RegressionNetParams
	case "sim":
		netParams = &chaincfg.SimNetParams
	}

	// this client uses websockets to connect and keep the connection alive so that multiple requests can be made without the overhead found in opening and closing connections to a server, and this also affords us the ability to receive notifications for certain events in real time
	notificationHandlers := &rpcclient.NotificationHandlers{
		// all transactions will go through this handler method
		OnTxAccepted: func(hash *chainhash.Hash, amount btcutil.Amount) {
			tx, err := getTransaction(hash)
			if err != nil {
				plutus.Logger.Error(err)
			}

			err = saveTxToDatabase(tx)
			if err != nil {
				plutus.Logger.Error(err)
			}
		},
		OnAccountBalance: func(account string, amount btcutil.Amount, confirmed bool) {
			// we only want to use confirmed transactions with values above 540 bits in balance updates our end of things. any transactions below this value will be recorded to the database as a possible dust transaction for further inspection
			if confirmed && amount.ToBTC() > amount.ToUnit(btcutil.AmountMicroBTC*540) {
				wallet, err := plutus.FindBitcoinWalletByAccount(account)
				if err != nil {
					plutus.Logger.Error(err)
				}
				err = wallet.UpdateBalance(amount.ToBTC())
				if err != nil {
					plutus.Logger.Errorf("Balance update failed for user: %s", account)
				}
				plutus.Logger.Infof("Balance updated for user: %s - New balance is %f", amount.ToBTC())
			}
		},
		OnWalletLockState: func(locked bool) {
			if locked {
				// log this message 5 times. It should be very apparent whenever the wallet is locked. At no time during normal operation should the entire wallet become locked. We should take great care to analyze incoming transactions for dust, and lock those outputs instead.
				for range []interface{}{nil, nil, nil, nil, nil} {
					plutus.Logger.Warnf("Wallet has locked. Please unlock it by calling WalletPassphrase")
				}
			} else {
				plutus.Logger.Info("Wallet unlocked")
			}
		},
		OnClientConnected: func() {
			plutus.Logger.Info("bitcoinRPCClient conncted to btcwallet rpc server")
		},
		OnBtcdConnected: func(connected bool) {
			if connected {
				plutus.Logger.Info("btcwallet connected to btcd")
			} else {
				plutus.Logger.Warn("btcwallet NOT connected to btcd")
			}
		},
	}

	// upon first running btcd/btcwallet, this directory will be created. seeing as that software is a prerequisite to this software, we should find this certificate with no issues
	appDataDir := btcutil.AppDataDir("Btcwallet", false)
	cert, err := ioutil.ReadFile(filepath.Join(appDataDir, "rpc.cert"))
	if err != nil {
		// but if we do have issues finding the certificate, we need to alert the operator and then kill the process
		plutus.Logger.Error("Could not find certificate used for communication to btcwallet")
		memguard.SafeExit(1)
	}

	clientConf := &rpcclient.ConnConfig{
		Host:         os.Getenv("BTCWALLET_HOST"),
		Endpoint:     "ws",
		User:         os.Getenv("RPC_USER"),
		Pass:         os.Getenv("RPC_PASS"),
		Certificates: cert,
	}

	BitcoinRPCClient, err = rpcclient.New(clientConf, notificationHandlers)
	if err != nil {
		plutus.Logger.Error(err)
	}

	// and also setup the monero rpc client. this is a much simpler, rest api based client.
	MoneroRPCClient = wallet.New(wallet.Config{
		Address: os.Getenv("MONERO_WALLET_RPC_URL"),
	})
}

// the documentation for the bitcoin rpc client provided to us by btcd says that we cannot call any blocking calls on the rpc instance directly inside of any of the notification handlers.
// that being said, this should be just fine, as we're not calling a blocking method on the rpc client instance. instead, we're using the asynchronous version of the GetTransaction method to return a future response type that we can receive the data from.
// the call to receive blocks until the data is ready, but because this is not blocking on the rpc client instance, it should be okay.
func getTransaction(hash *chainhash.Hash) (*btcjson.GetTransactionResult, error) {
	response := BitcoinRPCClient.GetTransactionAsync(hash)

	result, err := response.Receive()
	if err != nil {
		return nil, err
	}

	return result, nil
}

func saveTxToDatabase(transaction *btcjson.GetTransactionResult) error {
	for _, txInfo := range transaction.Details {
		switch strings.ToLower(txInfo.Category) {
		case "send":
			account, address, amount, fee, err := decodeTransactionDetails(&txInfo)
			if err != nil {
				return err
			}

			// save the withdrawl to our database
			err = plutus.NewBTCWithdrawl(transaction.TxID, account, address, amount, fee)
			if err != nil {
				return err
			}

		case "receive":
			account, address, amount, fee, err := decodeTransactionDetails(&txInfo)
			if err != nil {
				return err
			}

			// if we detect a transaction <540 bits in value then we want to alert the operator. this transaction could be dust, or it could simply be a user sending too little bitcoin.
			if checkForDust(&amount) {
				plutus.Logger.Warnf("Transaction below 540 bits detected. %f uBTC received by %s, TxID: %s", amount.ToUnit(btcutil.AmountMicroBTC), address.EncodeAddress(), transaction.TxID)
			}

			// save the deposit to our database
			err = plutus.NewBTCDeposit(transaction.TxID, account, address, amount, fee)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func decodeTransactionDetails(details *btcjson.GetTransactionDetailsResult) (account string, address btcutil.Address, amount, fee btcutil.Amount, err error) {

	account = details.Account

	address, err = btcutil.DecodeAddress(details.Address, netParams)
	if err != nil {
		return "", nil, 0, 0, err
	}

	amount, err = btcutil.NewAmount(details.Amount)
	if err != nil {
		return "", nil, 0, 0, err
	}

	fee, err = btcutil.NewAmount(*details.Fee)
	if err != nil {
		return "", nil, 0, 0, err
	}

	return
}

// wrapper method to avoid namespace conflicts
func xmrToFloat64(xmr uint64) float64 {
	return wallet.XMRToFloat64(xmr)
}

func checkForDust(amount *btcutil.Amount) bool {
	if amount.ToBTC() <= 0.00054000 {
		return true
	}
	return false
}
