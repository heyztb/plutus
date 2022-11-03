package plutus

import (
	"errors"
	"time"

	uuid "github.com/gofrs/uuid"
	"gopkg.in/go-playground/validator.v9"
)

// MoneroWallet will hold infomration about a wallet
type MoneroWallet struct {
	ID           uuid.UUID `json:"id" sql:"type:uuid" validate:"required,uuid" gorm:"primary"`
	Account      string    `json:"account" validate:"required" gorm:"unique"`
	AccountIndex uint64    `json:"accountIndex" validate:"required" gorm:"unique"`
	Address      string    `json:"address" validate:"required,btc_addr_bech32" gorm:"unique"`
	Balance      float64   `json:"balance" validate:"required"`

	CreatedAt *time.Time `json:"created_at,omitempty" gorm:"index"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" gorm:"index"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// NewMoneroWallet will create a new monero wallet in our database
func NewMoneroWallet(account string, accountIndex uint64, address string) error {
	_, err := FindMoneroWalletByAccount(account)
	if err == nil {
		return errors.New("Account already exists")
	}

	now := time.Now()
	w := &MoneroWallet{
		ID:           uuid.Must(uuid.NewV4()),
		Account:      account,
		AccountIndex: accountIndex,
		Address:      address,
		Balance:      0.000000000000,

		CreatedAt: &now,
		UpdatedAt: nil,
		DeletedAt: nil,
	}

	return w.Save()
}

// FindMoneroWalletByAccount will return a wallet or an error for a given account
func FindMoneroWalletByAccount(account string) (*MoneroWallet, error) {
	w := new(MoneroWallet)
	err := Database.Find(&w).Where("account=?", account).Error
	if err != nil {
		return nil, err
	}

	return w, nil
}

// FindMoneroWalletByAccountIndex will return a wallet or an error for a given account
func FindMoneroWalletByAccountIndex(accountIndex uint64) (*MoneroWallet, error) {
	w := new(MoneroWallet)
	err := Database.Find(&w).Where("account_index=?", accountIndex).Error
	if err != nil {
		return nil, err
	}

	return w, nil
}

// FindMoneroWalletByAddress will return or a wallet or an error for a given address.
func FindMoneroWalletByAddress(address string) (*MoneroWallet, error) {
	w := new(MoneroWallet)
	err := Database.Find(w).Where("address=?", address).Error
	if err != nil {
		return nil, err
	}

	return w, nil
}

// UpdateBalance will update the wallet's balance and save it in our database
func (w *MoneroWallet) UpdateBalance(newAmount float64) error {
	if newAmount < 0 {
		Logger.Warnf("Balance update for Monero wallet at account index %d denied. Balances cannot be negative", w.AccountIndex)
		return errors.New("Invalid balance update: Balances cannot be less than 0")
	}
	w.Balance = newAmount

	now := time.Now()
	w.UpdatedAt = &now

	// normally we would just return the value being returned from Save() (i.e return w.Save()) but I want to explicitly check for an error here to ensure that things went smoothly before we use our logger for the transaction.
	err := w.Save()
	if err != nil {
		return err
	}

	Logger.Infof("Monero wallet balance update for user: %s - balance updated to %f", w.Account, w.Balance)

	return nil
}

// UpdateAddress will update the wallet's address and save it in our database
func (w *MoneroWallet) UpdateAddress(newAddress string) error {
	Logger.Infof("Monero wallet @ Account Index: %d - Address updated to from %s to %s", w.AccountIndex, w.Address, newAddress)
	w.Address = newAddress

	now := time.Now()
	w.UpdatedAt = &now

	return w.Save()
}

// Save saves the wallet to db
func (w *MoneroWallet) Save() error {
	err := w.validate()
	if err != nil {
		return err
	}
	return w.saveToDatabase()
}

func (w *MoneroWallet) validate() error {
	validate = validator.New()

	err := validate.Struct(w)
	if err != nil {
		return err
	}

	return nil

}

func (w *MoneroWallet) saveToDatabase() error {
	return Database.Save(w).Error
}
