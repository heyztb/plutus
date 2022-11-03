package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/heyztb/plutus/plutus"
	"github.com/monero-ecosystem/go-monero-rpc-client/wallet"
	"go.uber.org/ratelimit"
)

// CreateXMRAccount will be used to create an account in monerod, labeled with the user ID it is associated with. It will return the index of their account, as well as an address that can be used to receive funds

// XMRNotify will be used as the endpoint queried by monero_wallet_rpc's --tx-notify flag
// URL will take the form of /xmr/notify/:txid
func XMRNotify(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	response, err := MoneroRPCClient.GetTransferByTxID(&wallet.RequestGetTransferByTxID{
		TxID: c.Param("txid"),
	})

	wallet, err := plutus.FindMoneroWalletByAccountIndex(response.Transfer.SubaddrIndex.Major)
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	var newBalance float64
	switch response.Transfer.Type {
	case "in":
		// track the incoming deposit
		err := plutus.NewXMRDeposit(response.Transfer.TxID, wallet.Account, response.Transfer.Address, xmrToFloat64(response.Transfer.Amount), xmrToFloat64(response.Transfer.Fee))
		if err != nil {
			c.JSON(400, gin.H{
				"error": err.Error(),
			})
			return
		}

		// and update the user's balance
		newBalance = wallet.Balance + xmrToFloat64(response.Transfer.Amount)
		err = wallet.UpdateBalance(newBalance)
		if err != nil {
			c.JSON(400, gin.H{
				"error": err.Error(),
			})
			return
		}

	case "out":
		// track the outgoing withdrawl
		err := plutus.NewXMRWithdrawl(response.Transfer.TxID, wallet.Account, response.Transfer.Address, xmrToFloat64(response.Transfer.Amount), xmrToFloat64(response.Transfer.Fee))
		if err != nil {
			c.JSON(400, gin.H{
				"error": err.Error(),
			})
			return
		}

		// and update the user's balance
		newBalance = wallet.Balance - xmrToFloat64(response.Transfer.Amount)
		err = wallet.UpdateBalance(newBalance)
		if err != nil {
			c.JSON(400, gin.H{
				"error": err.Error(),
			})
			return
		}
	}

	c.JSON(200, "OK")
}

// CreateXMRAccount takes a label and returns a new xmr account
// POST
// {
// 	"label": "label",
// }
func CreateXMRAccount(c *gin.Context) {

	limiter := ratelimit.New(3)
	limiter.Take()

	body := make(map[string]string)

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	response, err := MoneroRPCClient.CreateAccount(&wallet.RequestCreateAccount{
		Label: body["label"],
	})
	if err != nil {
		c.JSON(400, gin.H{
			"error": errRequestFailed.Error(),
		})
		return
	}

	err = plutus.NewMoneroWallet(body["label"], response.AccountIndex, response.Address)
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"account_index": response.AccountIndex,
		"address":       response.Address,
	})
}

// CreateXMRAddress takes an account label and searches for it in the database and then updates the user's address with a fresh one
func CreateXMRAddress(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	body := make(map[string]string)

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	w, err := plutus.FindMoneroWalletByAccount(body["label"])
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	response, err := MoneroRPCClient.CreateAddress(&wallet.RequestCreateAddress{
		AccountIndex: w.AccountIndex,
	})
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	err = w.UpdateAddress(response.Address)
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"address": response.Address,
	})
}

// GetXMRBalance takes an account label and searches for in the database, then returns the balance information to us
// POST /xmr/userbalance/
func GetXMRBalance(c *gin.Context) {

	limiter := ratelimit.New(10)
	limiter.Take()

	body := make(map[string]string)
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	w, err := plutus.FindMoneroWalletByAccount(body["label"])
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"balance": w.Balance,
	})
}

// TransferXMR will be used to transfer monero.
// Example request body:
// {
// "label": "source account",
// "destinations": [{"address": <address>, "amount": <float>}, ...],
// }
func TransferXMR(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	body := make(map[string]interface{})
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	w, err := plutus.FindMoneroWalletByAccount(body["label"].(string))
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	var destinations []*wallet.Destination

	for _, destination := range body["destinations"].([]map[string]interface{}) {

		// we're expecting the amount of XMR to be in a readable format, and not in atomic units
		amt := wallet.Float64ToXMR(destination["amount"].(float64))

		dest := &wallet.Destination{
			Amount:  amt,
			Address: destination["address"].(string),
		}
		destinations = append(destinations, dest)
	}

	response, err := MoneroRPCClient.Transfer(&wallet.RequestTransfer{
		Destinations: destinations,
		AccountIndex: w.AccountIndex,
		GetTxKey:     true,
		RingSize:     7,
		Priority:     1,
	})
	if err != nil {
		c.JSON(400, gin.H{
			"error": errRequestFailed.Error(),
		})
	}

	c.JSON(200, gin.H{
		"tx_id":  response.TxHash,
		"tx_key": response.TxKey,
	})
}

// SweepDustXMR will sweep all dust together, making it easier to spend and mix.
func SweepDustXMR(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	response, err := MoneroRPCClient.SweepDust(&wallet.RequestSweepDust{
		GetTxKeys: true,
	})
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, response)
}

// GetTransfers will be an admin utility to show incoming and outgoing transfers for a user wallet under our control
// POST /xmr/gettransfers/
// Example request body:
// {
// 	"label": "label"
// }
func GetTransfers(c *gin.Context) {

	limiter := ratelimit.New(5)
	limiter.Take()

	body := make(map[string]string)
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{
			"error": errInvalidRequestBody.Error(),
		})
		return
	}

	w, err := plutus.FindMoneroWalletByAccount(body["label"])
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		return
	}

	response, err := MoneroRPCClient.GetTransfers(&wallet.RequestGetTransfers{
		In:           true,
		Out:          true,
		AccountIndex: w.AccountIndex,
	})
	if err != nil {
		c.JSON(400, gin.H{
			"error": errRequestFailed.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"incoming": response.In,
		"outgoing": response.Out,
	})

}
