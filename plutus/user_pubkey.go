package plutus

import (
	"crypto/subtle"
	"errors"
	"time"

	"github.com/btcsuite/btcd/btcec"
	uuid "github.com/gofrs/uuid"
	"gopkg.in/go-playground/validator.v9"
)

// UserPublicKey defines the structure of our public_keys table. This table exists as a means for storing customer and vendor extended public keys for easier look up for multisignature transactions.
type UserPublicKey struct {
	ID      uuid.UUID `json:"id" sql:"type:uuid" validate:"required,uuid" gorm:"primary"`
	PubKey  string    `json:"key" sql:"type:varchar(1024)" validate:"required" gorm:"unique"`
	Account string    `json:"account" validate:"required" gorm:"unique"`

	CreatedAt *time.Time `json:"created_at,omitempty" gorm:"index"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" gorm:"index"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

var (
	errPubKeyAlreadyOnFile = errors.New("This key is already associated with this username")
	errInvalidPublicKey    = errors.New("Invalid public key submitted")
	errCouldNotSavePubKey  = errors.New("Failed to save public key to database")
)

// NewPubKey will accept a account and public key and insert it into the database
func NewPubKey(account, publicKey string) error {

	// ensure that we can actually parse this public key
	key, err := btcec.ParsePubKey([]byte(publicKey), btcec.S256())
	if err != nil {
		return errInvalidPublicKey
	}

	// we want to check the database to see if we have a key for this user already, and if we do we want to compare that key with the key being sent to us by the client. if the keys match, we return early with an error because it doesn't make sense to continue any further
	keyFromDB, err := FindPublicKeyByAccount(account)
	if err == nil {
		if subtle.ConstantTimeCompare([]byte(keyFromDB.PubKey), key.SerializeCompressed()) == 1 {
			return errPubKeyAlreadyOnFile
		}
	}

	now := time.Now()
	pubKey := &UserPublicKey{
		ID:      uuid.Must(uuid.NewV4()),
		PubKey:  publicKey,
		Account: account,

		CreatedAt: &now,
		UpdatedAt: &now,
	}

	err = pubKey.Save()
	if err != nil {
		return errCouldNotSavePubKey
	}

	return nil
}

// Queries

// FindPublicKeyByAccount will do a lookup in our database for a public key for a given account. This can either be a merchant or customer. That bit of information really doesn't matter to us.
func FindPublicKeyByAccount(account string) (*UserPublicKey, error) {
	var p UserPublicKey
	err := Database.Find(&p, "account=?", account).Error
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// Save will save our public key to the database after validating that it contains all necessary pieces
func (p *UserPublicKey) Save() error {
	err := p.validate()
	if err != nil {
		return err
	}
	return p.saveToDatabase()
}

func (p *UserPublicKey) saveToDatabase() error {
	return Database.Save(p).Error
}

func (p *UserPublicKey) validate() error {
	validate = validator.New()

	err := validate.Struct(p)
	if err != nil {
		return err
	}
	return nil
}
