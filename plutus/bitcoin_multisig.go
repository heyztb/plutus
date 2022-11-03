package plutus

import (
	"encoding/hex"
	"errors"
	"fmt"

	uuid "github.com/gofrs/uuid"
	"gopkg.in/go-playground/validator.v9"
)

// BTCMultisigWallet holds information regarding a multisig wallet
type BTCMultisigWallet struct {
	ID             uuid.UUID `json:"id" validate:"required,uuid" gorm:"primary" sql:"type:uuid"`
	Address        string    `json:"address" validate:"btc_addr_bech32"`
	RedeemScript   string    `json:"redeem_script"`
	VendorPubKey   []byte    `json:"vendor_pub_key" validate:"required" sql:"type:text"`
	CustomerPubKey []byte    `json:"customer_pub_key" validate:"required" sql:"type:text"`
	OperatorPubKey []byte    `json:"operator_pub_key" validate:"required" sql:"type:text" gorm:"unique"`
}

var (
	errCouldNotSaveMultisigWallet = errors.New("Failed to save multisig wallet to database")
)

// NewMultisigWallet accepts a public key for both the merchant and customer, and then uses that information alongside a public key that we control to create a new 2-of-3 multisig wallet.
func NewMultisigWallet(vendorKey, customerKey, operatorKey []byte, address, redeemScript string) error {

	// first we have to check the public keys provided to this method to ensure that they are at least some what valid looking and that we aren't just using random data
	err := checkPublicKey(vendorKey)
	if err != nil {
		return err
	}

	// do the same thing for the buyer key
	err = checkPublicKey(customerKey)
	if err != nil {
		return err
	}

	err = checkPublicKey(operatorKey)
	if err != nil {
		return err
	}

	msw := &BTCMultisigWallet{
		ID:             uuid.Must(uuid.NewV4()),
		Address:        address,
		RedeemScript:   redeemScript,
		CustomerPubKey: customerKey,
		VendorPubKey:   vendorKey,
		OperatorPubKey: operatorKey,
	}

	err = msw.Save()
	if err != nil {
		return errCouldNotSaveMultisigWallet
	}

	return nil
}

// Queries

// FindMultisigWalletByAddress exists so that we can at a later time, find the pubkeys used to create a multisig address.
func FindMultisigWalletByAddress(address string) (*BTCMultisigWallet, error) {
	var msw *BTCMultisigWallet
	err := Database.First(&msw).Where("script_address=?", address).Error
	if err != nil {
		return nil, err
	}

	return msw, nil
}

func checkPublicKey(key []byte) error {
	errMessage := ""

	switch {
	case key == nil:
		errMessage += "Public key cannot be empty.\n"
	case len(key) != 65:
		errMessage += fmt.Sprintf("Public key should be 65 bytes long. Provided public key is %d bytes long.", len(key))
	case key[0] != byte(4):
		errMessage += fmt.Sprintf("Public key first byte should be 0x04. Provided public key first byte is 0x%v.", hex.EncodeToString([]byte{key[0]}))
	}

	if errMessage != "" {
		errMessage += "Invalid public key:\n"
		errMessage += hex.EncodeToString(key)
		return errors.New(errMessage)
	}

	return nil
}

// Save validates and saves our multisig wallet to the database
func (msw *BTCMultisigWallet) Save() error {
	err := msw.validate()
	if err != nil {
		return err
	}

	return msw.saveToDatabase()
}

func (msw *BTCMultisigWallet) saveToDatabase() error {
	return Database.Save(msw).Error
}

func (msw *BTCMultisigWallet) validate() error {
	validate = validator.New()

	err := validate.Struct(msw)
	if err != nil {
		return err
	}

	return nil
}
