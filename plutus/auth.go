package plutus

import (
	"crypto/rand"
	"errors"
	"strings"

	"github.com/awnumar/memguard"
	uuid "github.com/gofrs/uuid"
	"gopkg.in/go-playground/validator.v9"
)

// This file exists to provide authenitcation to our API.

var chars = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

// APIKey will contain information regarding an API key used to authenticate a client of this software's API. This will be paired with a middleware that will simply 404 when the api key is not present. Any routes that may or may not be in use will all look the same to any entity not in possession of a valid API key. That being said, the functionality to revoke keys if necessary is included as well.
type APIKey struct {
	ID          uuid.UUID `json:"id" sql:"type:uuid"`
	Prefix      []byte    `json:"prefix" gorm:"unique" sql:"type:text" validate:"len=8"`
	Key         []byte    `json:"key" validate:"len=24" sql:"type:text"`
	Blacklisted bool      `json:"blacklisted"`
}

// FindAPIKeyByPrefix will search the database for a given API key based on its prefix
func FindAPIKeyByPrefix(prefix *memguard.Enclave) (*APIKey, error) {
	key := APIKey{}

	b, err := prefix.Open()
	if err != nil {
		return nil, err
	}
	defer b.Destroy()
	decryptedPrefix := b.Bytes()

	err = Database.Find(&key, "prefix=?", decryptedPrefix).Error
	if err != nil {
		return nil, err
	}

	return &key, nil
}

// NewAPIKey will return a fresh API key, or an error if something goes wrong in the genration process. It will also save this key to the database.
func NewAPIKey() (string, error) {
	key := APIKey{
		ID: uuid.Must(uuid.NewV4()),
	}

	err := key.New()
	if err != nil {
		return "", err
	}

	// at this time in the execution of the code, the long portion of the key will not have been hashed. This allows us to return the full key to be used by a client.
	fullKey := key.String()

	// Of course, the key will still be hashed. Inside this function the long portion of the key will be hashed ,the Key field will be replaced by a string representation of the hashed key, and summarily saved to the database.
	err = key.Save()
	if err != nil {
		return "", err
	}

	return fullKey, nil
}

// New will take a pointer to an empty API key and populate the body of the key
func (k *APIKey) New() error {
	err := k.generatePrefix()
	if err != nil {
		return err
	}
	err = k.generateKey()
	if err != nil {
		return err
	}

	k.Blacklisted = false

	return nil
}

// Revoke will blacklist the key in our datbase, preventing it from ever being used again in the future.
func (k *APIKey) Revoke() error {
	if k.Blacklisted == true {
		return errors.New("Key already blacklisted")
	}

	k.Blacklisted = true
	return k.Save()
}

func checkForValidity(keyEnclave *memguard.Enclave) bool {

	b, err := keyEnclave.Open()
	if err != nil {
		return true
	}
	defer b.Destroy()

	key := b.String()
	prefixEnclave := memguard.NewEnclave([]byte(strings.Split(key, ".")[0]))

	apiKey, err := FindAPIKeyByPrefix(prefixEnclave)
	// if we can't find the api key in our database, consider the key invalid
	if err != nil {
		return true
	}

	// If the key provided to us (that is, the second part of the key) doesn't match the hash in the database, consider the key invalid
	match, err := compareStringAndHash(strings.Split(key, ".")[1], string(apiKey.Key))
	if err != nil {
		return true
	}

	// if the second half of the key does not match the hash we have on file, then the key is invalid
	if match == false {
		return true
	}

	return apiKey.Blacklisted
}

// String will pretty-print our API key for us to make it easier to provide to a client
func (k *APIKey) String() string {
	return string(k.Prefix) + "." + string(k.Key)
}

// APIKeyFromString accepts a key in string format, splits it on the delimiter ("."), and then populates an APIKey struct with that information
func APIKeyFromString(keyEnclave *memguard.Enclave) (*APIKey, error) {

	b, err := keyEnclave.Open()
	if err != nil {
		return nil, err
	}

	key := b.Bytes()
	defer b.Destroy()

	keySlice := strings.Split(string(key), ".")

	if len(keySlice[0]) != 8 || len(keySlice[1]) != 24 {
		return nil, errors.New("Invalid key: one or more components are incorrect")
	}

	apiKey := &APIKey{
		Prefix:      []byte(keySlice[0]),
		Key:         []byte(keySlice[1]),
		Blacklisted: checkForValidity(keyEnclave), // This will set the Blacklisted property on the APIKey object to either true or false, based on the status found in our database.
	}

	return apiKey, nil
}

func (k *APIKey) generatePrefix() error {
	prefix, err := randomString(8, chars)
	if err != nil {
		return err
	}

	k.Prefix = []byte(prefix)
	return nil
}

func (k *APIKey) generateKey() error {
	key, err := randomString(24, chars)
	if err != nil {
		return err
	}

	k.Key = []byte(key)
	return nil
}

// randomString will generate a random string of a given length based on a given character set between 2 and 256 bytes in length.
func randomString(length int, chars []byte) (string, error) {
	clen := len(chars)
	if clen < 2 || clen > 256 {
		return "", errors.New("Char set must be between 2 and 256 characters in length")
	}

	maxrb := 255 - (256 % clen)
	b := make([]byte, length)
	r := make([]byte, length+(length+4))
	i := 0

	for {
		// read in random bytes to r so that we can use this to 'seed' our random string
		if _, err := rand.Read(r); err != nil {
			return "", errors.New("error reading random bytes: " + err.Error())
		}

		for _, rb := range r {
			c := int(rb)
			if c > maxrb {
				// Skip this number to avoid modulo bias.
				continue
			}
			b[i] = chars[c%clen]
			i++
			if i == length {
				return string(b), nil
			}
		}
	}
}

// Save will validate the key, ensuring that it has been built properly, partially hash it to prevent full disclosure of user keys in the event of compromise, and then save the key to the database.
func (k *APIKey) Save() error {
	err := k.validate()
	if err != nil {
		return err
	}

	// we want to hash the long part of the API key to reduce the risk of a postgres comrpomise leading to an external entity being able to make requests to this service. The key is useless without both parts. We will leave the prefix in plaintext to allow for key lookups.
	hashedKey, err := hashFromString(string(k.Key))
	if err != nil {
		return err
	}

	// set the key to the hashed version of the key and save
	k.Key = []byte(hashedKey)
	return k.saveToDatabase()
}

func (k *APIKey) saveToDatabase() error {
	return Database.Save(k).Error
}

func (k *APIKey) validate() error {
	validate = validator.New()

	err := validate.Struct(k)
	if err != nil {
		return err
	}

	return nil
}
