package plutus

import (
	"errors"
	"time"

	uuid "github.com/gofrs/uuid"
	"gopkg.in/go-playground/validator.v9"
)

// BitcoinWallet will hold infomration about a wallet
type BitcoinWallet struct {
	ID      uuid.UUID `json:"id" sql:"type:uuid" validate:"required,uuid" gorm:"primary"`
	Account string    `json:"account" validate:"required" gorm:"unique"`
	Address string    `json:"address" validate:"required,btc_addr_bech32"`
	Balance float64   `json:"balance" validate:"required"`

	CreatedAt *time.Time `json:"created_at,omitempty" gorm:"index"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" gorm:"index"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

var (
	errAddressUpdateNotNeeded = errors.New("Address already on file")
	errInvalidBalanceUpdate   = errors.New("Invalid balanace update")
)

// NewBitcoinWallet will create a new bitcoin wallet in our database
func NewBitcoinWallet(account, address string) error {
	w, err := FindBitcoinWalletByAccount(account)
	if err == nil {
		// if the account already exists, we can take this time to update the address
		return w.UpdateAddress(address)
	}

	now := time.Now()
	w = &BitcoinWallet{
		ID:      uuid.Must(uuid.NewV4()),
		Account: account,
		Address: address,
		Balance: 0.00000000,

		CreatedAt: &now,
		UpdatedAt: nil,
		DeletedAt: nil,
	}

	return w.saveToDatabase()
}

// FindBitcoinWalletByAccount will return a wallet or an error for a given account label.
func FindBitcoinWalletByAccount(account string) (*BitcoinWallet, error) {
	w := new(BitcoinWallet)
	err := Database.Find(&w, "account=?", account).Error
	if err != nil {
		return nil, err
	}

	return w, nil
}

// FindBitcoinWalletByAddress will return or a wallet or an error for a given address.
func FindBitcoinWalletByAddress(address string) (*BitcoinWallet, error) {
	w := new(BitcoinWallet)
	err := Database.Find(w).Where("address=?", address).Error
	if err != nil {
		return nil, err
	}

	return w, nil
}

// UpdateBalance will update the wallet's balance and save it in our database
func (w *BitcoinWallet) UpdateBalance(newAmount float64) error {
	if newAmount < 0 {
		Logger.Warnf("Balance update for Bitcoin wallet belonging to '%s' denied. Balances cannot be negative", w.Account)
		return errInvalidBalanceUpdate
	}
	w.Balance = newAmount

	now := time.Now()
	w.UpdatedAt = &now

	err := w.Save()
	if err != nil {
		return err
	}

	Logger.Infof("Bitcoin wallet balance update for user: %s - balance updated to %f", w.Account, w.Balance)

	return nil
}

// UpdateAddress will update the wallet's address and save it in our database
func (w *BitcoinWallet) UpdateAddress(newAddress string) error {
	if newAddress == w.Address {
		return errAddressUpdateNotNeeded
	}

	Logger.Infof("BTC Address Update: %s - new address: %s", w.Account, newAddress)
	w.Address = newAddress

	now := time.Now()
	w.UpdatedAt = &now

	return w.saveToDatabase()
}

// Save saves the wallet to db
func (w *BitcoinWallet) Save() error {
	err := w.validate()
	if err != nil {
		return err
	}
	return w.saveToDatabase()
}

func (w *BitcoinWallet) validate() error {
	validate = validator.New()

	err := validate.Struct(w)
	if err != nil {
		return err
	}

	return nil
}

func (w *BitcoinWallet) saveToDatabase() error {
	return Database.Save(w).Error
}
