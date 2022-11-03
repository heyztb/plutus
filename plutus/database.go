package plutus

import (
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres" //postgres dialect
	validator "gopkg.in/go-playground/validator.v9"
)

var (
	// Database is a global variable we will be using to access our database
	Database *gorm.DB
	validate *validator.Validate
)

func init() {
	var err error
	Database, err = gorm.Open("postgres", os.Getenv("PG_URL"))
	if err != nil {
		panic(err)
	}

	if os.Getenv("DEBUG") == "true" {
		Database.LogMode(true)
	}

	Database.DB().SetMaxOpenConns(8)
}

// SyncModels ensures that the database models are up to date
func SyncModels() {
	Database.AutoMigrate(
		BitcoinWallet{},
		MoneroWallet{},
		BTCMultisigWallet{},
		UserPublicKey{},
		OperatorPublicKey{},
		APIKey{},
		BitcoinTransaction{},
		MoneroTransaction{},
	)

	addConstraints()
}

// AddConstraints will add foreign key constraints
func addConstraints() {
	Database.Model(BTCMultisigWallet{}).AddForeignKey("vendor_pub_key", "user_public_keys(pub_key)", "CASCADE", "CASCADE")
	Database.Model(BTCMultisigWallet{}).AddForeignKey("customer_pub_key", "user_public_keys(pub_key)", "CASCADE", "CASCADE")
	Database.Model(BTCMultisigWallet{}).AddForeignKey("operator_pub_key", "operator_public_keys(pub_key)", "CASCADE", "CASCADE")
	Database.Model(BitcoinTransaction{}).AddForeignKey("account", "bitcoin_wallets(account)", "CASCADE", "CASCADE")
	Database.Model(MoneroTransaction{}).AddForeignKey("account", "monero_wallets(account)", "CASCADE", "CASCADE")
}
