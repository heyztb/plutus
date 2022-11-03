package main

import (
	"os"
	"strings"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/heyztb/plutus/routes"
)

func init() {
	if strings.ToLower(os.Getenv("DEBUG")) == "true" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
}

func configureRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	if strings.ToLower(os.Getenv("DEBUG")) == "true" {
		router.Use(gin.Logger())
		pprof.Register(router)
	}

	// routes

	xmr := router.Group("/xmr")
	{
		xmr.GET("/notify/:txid", routes.CheckAPIKey, routes.XMRNotify)
		xmr.POST("/createaccount", routes.CheckAPIKey, routes.CreateXMRAccount)
		xmr.POST("/createaddress", routes.CheckAPIKey, routes.CreateXMRAddress)
		xmr.POST("/getbalance", routes.CheckAPIKey, routes.GetXMRBalance)
		xmr.POST("/transfer", routes.CheckAPIKey, routes.TransferXMR)
	}

	btc := router.Group("/btc")
	{
		btc.GET("/balance", routes.CheckAPIKey, routes.GetBalance)
		btc.POST("/createaccount", routes.CheckAPIKey, routes.CreateAccount)
		btc.POST("/listtransactions", routes.CheckAPIKey, routes.ListTransactions)
		btc.POST("/sendtoaddress", routes.CheckAPIKey, routes.SendToAddress)
		btc.POST("/sendfrom", routes.CheckAPIKey, routes.SendFrom)
		btc.POST("/sendmany", routes.CheckAPIKey, routes.SendMany)
		btc.POST("/move", routes.CheckAPIKey, routes.Move)
		btc.POST("/getaccountaddress", routes.CheckAPIKey, routes.GetAccountAddress)
		btc.POST("/addmultisig", routes.CheckAPIKey, routes.AddMultisigAddress)
		btc.POST("/submit_pubkey", routes.CheckAPIKey, routes.SubmitPublicKey)
		btc.POST("/unlock", routes.CheckAPIKey, routes.UnlockWallet)
	}

	api := router.Group("/api")
	{
		api.POST("/revoke", routes.CheckAPIKey, routes.RevokeAPIKey)
	}

	return router
}
