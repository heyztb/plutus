package plutus

import (
	"time"

	"github.com/btcsuite/btcutil"
	uuid "github.com/gofrs/uuid"
	"gopkg.in/go-playground/validator.v9"
)

// This file exists to provide database logic surrounding transactions made through this application. The primary purpose of this table is to serve as one source of information regarding transactions made through our service. The other source will of course be from the RPC interfaces exposed by the respective cryptod. This will allow us to perform reconciliation on all balances to ensure that no users are missing or gaining access to funds they should or should not otherwise have

// BitcoinTransaction is a struct mapping the layout for our bitcoin_transactions table
type BitcoinTransaction struct {
	ID           uuid.UUID `json:"id" sql:"type:uuid" gorm:"primary" validate:"required,uuid"`
	TxID         string    `json:"tx_id" validate:"required"`
	Account      string    `json:"account" validate:"required"`
	Address      string    `json:"address" validate:"required"`
	Amount       float64   `json:"amount" validate:"required"`
	Fee          float64   `json:"fee" validate:"required"`
	Deposit      bool      `json:"deposit" validate:"required"`
	Withdrawl    bool      `json:"withdrawl" validate:"required"`
	PossibleDust bool      `json:"possible_dust" validate:"required"`

	CreatedAt *time.Time `json:"created_at,omitempty" gorm:"index"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// FindBitcoinTransactionByTxID will search the transaction table for a given txID
func FindBitcoinTransactionByTxID(txID string) (*BitcoinTransaction, error) {
	tx := new(BitcoinTransaction)
	err := Database.Find(&tx).Where("tx_id=?", txID).Error
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// FindBitcoinDepositsByAccount will search the database for an account and find all deposits
func FindBitcoinDepositsByAccount(account string) ([]*BitcoinTransaction, error) {
	var txs []*BitcoinTransaction
	err := Database.Find(&txs, "account=?", account, "deposit=?", true).Error
	if err != nil {
		return nil, err
	}

	return txs, nil
}

// FindBitcoinWithdrawlsByAccount will search the database for an account and return all withdrawls
func FindBitcoinWithdrawlsByAccount(account string) ([]*BitcoinTransaction, error) {

	var txs []*BitcoinTransaction
	err := Database.Find(&txs, "account=?", account, "withdrawl=?", true).Error
	if err != nil {
		return nil, err
	}

	return txs, nil
}

// NewBTCDeposit will crete and save a new bitcoin deposit to the database
func NewBTCDeposit(txid, account string, address btcutil.Address, amount, fee btcutil.Amount) error {
	tx := new(BitcoinTransaction)

	return tx.new(txid, account, address, amount, fee, true, false)
}

// NewBTCWithdrawl will create and save a new bitcoin withdrawl to the database
func NewBTCWithdrawl(txid, account string, address btcutil.Address, amount, fee btcutil.Amount) error {
	tx := new(BitcoinTransaction)

	return tx.new(txid, account, address, amount, fee, false, true)
}

func (tx *BitcoinTransaction) new(txid, account string, address btcutil.Address, amount, fee btcutil.Amount, deposit, withdrawl bool) error {
	now := time.Now()

	// we want to track any transactions for dust amounts. if we detect that the value of a transaction falls below 540 bits, then mark it as a possible dust transaction. This is important because if the operator begins to notice or suspects a dust attack is in progress, they can easily find any suspect transactions and lock them so that they won't be spent in a future transaction, thus negating the dust attack
	isDust := false
	if amount.ToBTC() < amount.ToUnit(btcutil.AmountMicroBTC*540) {
		isDust = true
	}

	tx = &BitcoinTransaction{
		TxID:         txid,
		Account:      account,
		Address:      address.EncodeAddress(),
		Amount:       amount.ToBTC(),
		Fee:          fee.ToBTC(),
		Deposit:      deposit,
		Withdrawl:    withdrawl,
		PossibleDust: isDust,
		CreatedAt:    &now,
	}

	err := tx.Save()
	if err != nil {
		return err
	}

	if deposit {
		Logger.Infof("New Bitcoin deposit for [%f] account : [%s] on address :  [%s]", account, address, amount.ToBTC())
	} else {
		Logger.Infof("New Bitcoin withdrawl for [%f] account : [%s] on address :  [%s]", account, address, amount.ToBTC())
	}

	return nil
}

// Save will save the transaction to our database
func (tx *BitcoinTransaction) Save() error {
	err := tx.validate()
	if err != nil {
		return err
	}
	return tx.saveToDatabase()
}

func (tx *BitcoinTransaction) saveToDatabase() error {
	return Database.Save(tx).Error
}

func (tx *BitcoinTransaction) validate() error {
	validate = validator.New()

	err := validate.Struct(tx)
	if err != nil {
		return err
	}
	return nil
}

// MoneroTransaction is a struct mapping the layout for our monero_transactions table
type MoneroTransaction struct {
	ID        uuid.UUID `json:"id" sql:"type:uuid" gorm:"primary" validate:"required,uuid"`
	TxID      string    `json:"tx_id" validate:"required"`
	Account   string    `json:"account" validate:"required"`
	Address   string    `json:"address" validate:"required"`
	Amount    float64   `json:"amount" validate:"required"`
	Fee       float64   `json:"fee" validate:"required"`
	Deposit   bool      `json:"deposit" validate:"required"`
	Withdrawl bool      `json:"withdrawl" validate:"required"`

	CreatedAt *time.Time `json:"created_at,omitempty" gorm:"index"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// FindMoneroTransactionByTxID will search the transaction table for a given txHash
func FindMoneroTransactionByTxID(txID string) (*MoneroTransaction, error) {
	tx := new(MoneroTransaction)
	err := Database.Find(&tx).Where("tx_id=?", txID).Error
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// FindMoneroDepositsByAccount will search the database for an account and return all deposits
func FindMoneroDepositsByAccount(account string) (*[]MoneroTransaction, error) {
	txs := new([]MoneroTransaction)
	err := Database.Find(&txs, "account=?", account, "deposit=?", true).Error
	if err != nil {
		return nil, err
	}

	return txs, nil
}

// FindMoneroWithdrawlsByAccount will search the database for an account and return all withdrawls
func FindMoneroWithdrawlsByAccount(account string) (*[]MoneroTransaction, error) {

	txs := new([]MoneroTransaction)
	err := Database.Find(&txs, "account=?", account, "withdrawl=?", true).Error
	if err != nil {
		return nil, err
	}

	return txs, nil
}

// NewXMRDeposit is used to track a new XMR deposit in our system
func NewXMRDeposit(txid, account, address string, amount, fee float64) error {
	tx := new(MoneroTransaction)

	return tx.new(txid, account, address, amount, fee, true, false)
}

// NewXMRWithdrawl is used to trantck a new XMR withdrawl in our system
func NewXMRWithdrawl(txid, account, address string, amount, fee float64) error {

	tx := new(MoneroTransaction)

	return tx.new(txid, account, address, amount, fee, false, true)
}

func (tx *MoneroTransaction) new(txid, account, address string, amount, fee float64, deposit, withdrawl bool) error {

	now := time.Now()

	tx.TxID = txid
	tx.Account = account
	tx.Address = address
	tx.Amount = amount
	tx.Fee = fee
	tx.Deposit = deposit
	tx.Withdrawl = withdrawl
	tx.CreatedAt = &now

	err := tx.Save()
	if err != nil {
		return err
	}

	if deposit {
		Logger.Infof("New Monero deposit for [%f] account : [%s] on address :  [%s]", account, address, amount)
	} else {
		Logger.Infof("New Monero withdrawl for [%f] account : [%s] on address :  [%s]", account, address, amount)
	}

	return nil
}

// Save will save the transaction to our database
func (tx *MoneroTransaction) Save() error {
	err := tx.validate()
	if err != nil {
		return err
	}
	return tx.saveToDatabase()
}

func (tx *MoneroTransaction) saveToDatabase() error {
	return Database.Save(tx).Error
}

func (tx *MoneroTransaction) validate() error {
	validate = validator.New()

	err := validate.Struct(tx)
	if err != nil {
		return err
	}
	return nil
}
