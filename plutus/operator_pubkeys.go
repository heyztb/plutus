package plutus

import (
	"math/rand"
	"time"

	uuid "github.com/gofrs/uuid"
	"gopkg.in/go-playground/validator.v9"
)

// OperatorPublicKey is a struct that defines the strucutre of our operator_public_keys table. This table will hold the keys under our control that can be used in multisignature transactions.
type OperatorPublicKey struct {
	ID     uuid.UUID `json:"id" sql:"type:uuid" validate:"required,uuid" gorm:"primary"`
	PubKey []byte    `json:"public_key" sql:"type:text" validate:"required" gorm:"unique"`

	CreatedAt *time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

// GetOperatorKey will search the database for all of our public keys, randomly select one, and return it to us.
func GetOperatorKey() ([]byte, error) {
	var keys []*OperatorPublicKey
	err := Database.Find(&keys).Error
	if err != nil {
		return nil, err
	}

	// non cryptographically secure rand in use here, mainly because I don't really see the need, especially given that the set of pub keys will only ever get so large
	rand.Seed(time.Now().UTC().UnixNano())
	key := keys[rand.Intn(len(keys))].PubKey

	return key, nil
}

// Save will save this operator public key to our database
func (p *OperatorPublicKey) Save() error {
	err := p.validate()
	if err != nil {
		return err
	}
	return p.saveToDatabase()
}

func (p *OperatorPublicKey) saveToDatabase() error {
	return Database.Save(p).Error
}

func (p *OperatorPublicKey) validate() error {
	validate = validator.New()

	err := validate.Struct(p)
	if err != nil {
		return err
	}
	return nil
}
