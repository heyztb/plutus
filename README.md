## Plutus

### Why?

This project was created for educational purposes. It provides an API to interact with a Bitcoin and Monero daemon simultaneously, particularly [Btcwallet](https://github.com/btcsuite/btcwallet) and [Monero-wallet-rpc](https://getmonero.org/). You may want to use this software if you are building an ecommerce application and would like to accept these cryptocurrencies as payment methods.

### Features

[x] Compiles down to a single binary, can theoretically be run anywhere that Bitcoin and Monero can  
[x] Communications between client software and this application secured using TLS 
[x] Secure API key system to ensure only authenticated clients may make requests to this software (keys are generated on the command line by the operator and distributed to clients as necessary)  
[x] Uses [memguard](https://github.com/awnumar/memguard) to secure sensitive information in memory, protecting against cold-boot and side-channel attacks.  
[x] Simple API system to interface with the most important e-commerce related functions of the Bitcoin and Monero daemons  
[x] Rudimentary Bitcoin dust attack detection (we don't really do anything about it, however)  
[x] Automatically tracks all incoming and outgoing bitcoin transactions, updates user balances accordingly  
[x] API endpoint exposed to allow the same behavior for Monero (use monero-wallet-rpc --tx-notify + curl for this)  
[x] Bitcoin multisig support  

### Usage

```
  -> clone this repository
  -> populate .env or the actual OS environment variables
  -> go run . || go build -o plu; ./plu
```

### Development & Testing

In order to get started working on this codebase, you will need a few things first.

- [PostgreSQL](https://postgresql.org/)
- [Btcwallet](https://github.com/btcsuite/btcwallet)
- [Monero-wallet-rpc](https://getmonero.org)
- [Golang](https://golang.org)

This repository still needs unit tests. If you want to contribute these, feel free.

I have made my best effort at leaving useful comments throughout the codebase, to ensure that any new readers or maintainers can get familiar with the codebase with minimal effort.