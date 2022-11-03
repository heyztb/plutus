package routes

import (
	"errors"
	"fmt"
	"os"

	"github.com/btcsuite/btcutil"
	"github.com/gin-gonic/gin"
	"github.com/heyztb/plutus/plutus"
	"go.uber.org/ratelimit"
)

var (
	errInvalidRequestBody      = errors.New("Invalid request body")
	errInvalidBitcoinAddress   = errors.New("Invalid bitcoin address detected")
	errInvalidBitcoinValue     = errors.New("Invalid bitcoin value detected")
	errSendingTransaction      = errors.New("Error sending transaction")
	errRequestFailed           = errors.New("The request failed, please try again")
	errBalanceCheck            = errors.New("Could not obtain platform balance")
	errFailedToFindPublicKey   = errors.New("Failed to find a public key")
	errInvalidPublicKey        = errors.New("Public key on file is invalid")
	errInvalidWalletPassphrase = errors.New("Invalid wallet passphrase")
)

// GetBalance will be used to return the total observable balance on the platform across all users. GET request, no body.
// TODO: Lock this behind administrative permission check middleware
func GetBalance(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	// this will grab the balance from all accounts and return it as a single value :)
	response, err := BitcoinRPCClient.GetBalance("*")
	if err != nil {
		c.JSON(400, gin.H{
			"error": errBalanceCheck.Error(),
		})
	}

	// amount in BTC
	c.JSON(200, gin.H{
		"balance": response.String(),
	})
}

// {
//  "label": "account",
// }

// ListTransactions will be used to gather transaction information for our users
func ListTransactions(c *gin.Context) {
	limiter := ratelimit.New(5)
	limiter.Take()

	body := make(map[string]string)

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	response, err := BitcoinRPCClient.ListTransactions(body["label"])
	if err != nil {
		c.JSON(400, gin.H{
			"error": errRequestFailed.Error(),
		})
		return
	}

	for k := range body {
		delete(body, k)
	}

	// please refer to the above response type to see the response structure.
	c.JSON(200, gin.H{
		"transactions": response,
	})
}

// {
// 	"destination": "destinationAddress",
// 	"amount": "amount in BTC, can be string or number"
// }

// SendToAddress creates signs and sends a transaction to the given address
func SendToAddress(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	body := make(map[string]interface{})

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	destination, err := btcutil.DecodeAddress(body["destination"].(string), netParams)
	if err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidBitcoinAddress.Error(),
		})
		return
	}
	amount, err := btcutil.NewAmount(body["amount"].(float64))
	if err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidBitcoinValue.Error(),
		})
		return
	}

	response, err := BitcoinRPCClient.SendToAddress(destination, amount)
	if err != nil {
		c.JSON(400, gin.H{
			"error": errSendingTransaction.Error(),
		})
	}

	for k := range body {
		delete(body, k)
	}

	c.JSON(200, gin.H{
		"txid": response.String(),
	})
}

// SendFrom will use a specific account as the source of funds in a transaction
// This can likely be used for withdrawls on our side of things (as operators)
func SendFrom(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	body := make(map[string]interface{})

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	destination, err := btcutil.DecodeAddress(body["destination"].(string), netParams)
	if err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidBitcoinAddress.Error(),
		})
		return
	}

	amount, err := btcutil.NewAmount(body["amount"].(float64))
	if err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidBitcoinValue.Error(),
		})
		return
	}

	response, err := BitcoinRPCClient.SendFrom(body["source"].(string), destination, amount)
	if err != nil {
		c.JSON(400, gin.H{
			"error": errSendingTransaction.Error(),
		})
		return
	}

	for k := range body {
		delete(body, k)
	}

	c.JSON(200, gin.H{
		"txid": response.String(),
	})
}

// SendMany sends multiple amounts to multiple addresses using the provided account as a source of funds in a single transaction. Useful for taking commission.
func SendMany(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	body := make(map[string]interface{})

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	destinationMap := make(map[btcutil.Address]btcutil.Amount)

	for address, amount := range body["destinations"].(map[string]float64) {
		addr, err := btcutil.DecodeAddress(address, netParams)
		if err != nil {
			c.JSON(400, gin.H{
				"error": fmt.Sprintf(errInvalidBitcoinAddress.Error()+": %s", address),
			})
			return
		}

		amnt, err := btcutil.NewAmount(amount)
		if err != nil {
			c.JSON(400, gin.H{
				"error": fmt.Sprintf(errInvalidBitcoinValue.Error()+" in tx to %s", address),
			})
			return
		}

		destinationMap[addr] = amnt
	}

	response, err := BitcoinRPCClient.SendMany(body["source"].(string), destinationMap)
	if err != nil {
		c.JSON(400, gin.H{
			"error": errSendingTransaction.Error(),
		})
		return
	}

	for k := range body {
		delete(body, k)
	}

	c.JSON(200, gin.H{
		"txid": response.String(),
	})

}

// Move will move funds from one account to another
func Move(c *gin.Context) {
	limiter := ratelimit.New(5)
	limiter.Take()

	body := make(map[string]interface{})
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	amount, err := btcutil.NewAmount(body["amount"].(float64))
	if err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidBitcoinValue.Error(),
		})
		return
	}

	response, err := BitcoinRPCClient.Move(body["source"].(string), body["destination"].(string), amount)
	if err != nil {
		c.JSON(400, gin.H{
			"error": errSendingTransaction.Error(),
		})
		return
	}

	for k := range body {
		delete(body, k)
	}

	// true or false
	c.JSON(200, response)
}

// CreateAccount will create an account for the supplied label on our btcwallet server
func CreateAccount(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	body := make(map[string]string)

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	err := BitcoinRPCClient.CreateNewAccount(body["label"])
	if err != nil {
		c.JSON(400, gin.H{
			"error": errRequestFailed.Error(),
		})
		return
	}

	for k := range body {
		delete(body, k)
	}

	c.JSON(200, "OK")
}

// GetAccountAddress will return a new address for the supplied account
func GetAccountAddress(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	body := make(map[string]string)

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	response, err := BitcoinRPCClient.GetAccountAddress(body["label"])
	if err != nil {
		c.JSON(400, gin.H{
			"error": errRequestFailed.Error(),
		})
		return
	}

	err = plutus.NewBitcoinWallet(body["label"], response.EncodeAddress())
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	for k := range body {
		delete(body, k)
	}

	c.JSON(200, gin.H{
		"address": response.EncodeAddress(),
	})
}

// AddMultisigAddress will take public keys from both the customer and merchant involved in the transaction, as well as a public key in our control (abstracted away, you won't see it here), and use those to generate a native segwit multisig address to be used in a transaction. /btc/new_multisig
// Post body should look something like this
// {
// 	"vendor": username,
//	"customer": username,
// }
func AddMultisigAddress(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	body := make(map[string]string)

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	customer, err := plutus.FindPublicKeyByAccount(body["customer"])
	if err != nil {
		c.JSON(400, gin.H{
			"error": errFailedToFindPublicKey.Error(),
			"field": "customer",
		})
	}

	customerKey, err := btcutil.NewAddressPubKey([]byte(customer.PubKey), netParams)
	if err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidPublicKey.Error(),
			"field": "customer",
		})
		return
	}

	vendor, err := plutus.FindPublicKeyByAccount(body["vendor"])
	if err != nil {
		c.JSON(400, gin.H{
			"error": errFailedToFindPublicKey.Error(),
			"field": "vendor",
		})
		return
	}

	for k := range body {
		delete(body, k)
	}

	vendorKey, err := btcutil.NewAddressPubKey([]byte(vendor.PubKey), netParams)
	if err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidPublicKey.Error(),
			"field": "vendor",
		})
		return
	}

	operator, err := plutus.GetOperatorKey()
	if err != nil {
		c.JSON(400, gin.H{
			"error": errFailedToFindPublicKey.Error(),
			"field": "operator",
		})
		return
	}

	operatorKey, err := btcutil.NewAddressPubKey(operator, netParams)
	if err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidPublicKey.Error(),
			"field": "operator",
		})
		return
	}

	addresses := []btcutil.Address{
		customerKey,
		vendorKey,
		operatorKey,
	}

	response, err := BitcoinRPCClient.CreateMultisig(2, addresses)
	if err != nil {
		c.JSON(400, gin.H{
			"error": errRequestFailed.Error(),
		})
		return
	}

	err = plutus.NewMultisigWallet(customerKey.PubKey().SerializeCompressed(), operatorKey.PubKey().SerializeCompressed(), vendorKey.PubKey().SerializeCompressed(), response.Address, response.RedeemScript)
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, response)
}

// {
// 	"username": "username",
// 	"public_key": "xpub_key"
// }

// SubmitPublicKey accepts a username and a public key
func SubmitPublicKey(c *gin.Context) {

	body := make(map[string]string)

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	err := plutus.NewPubKey(body["username"], body["pubKey"])
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	for k := range body {
		delete(body, k)
	}

	c.JSON(200, gin.H{
		"message": "Key successfully inserted or updated",
	})

}

// UnlockWallet will unlock the wallet
func UnlockWallet(c *gin.Context) {

	// Unlock the wallet for ~4 months
	err := BitcoinRPCClient.WalletPassphrase(os.Getenv("WALLET_PASS"), 86400*30*4)
	if err != nil {
		c.JSON(400, gin.H{
			"err": errInvalidWalletPassphrase,
		})
		return
	}

	c.JSON(200, "OK")

}
