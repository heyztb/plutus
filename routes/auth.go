package routes

import (
	"strings"

	"github.com/awnumar/memguard"
	"github.com/gin-gonic/gin"
	"go.uber.org/ratelimit"
	"github.com/heyztb/plutus/plutus"
)

// CheckAPIKey will check for the presence of an authorization header and then run some checks against this key to ensure that it is valid. If the header is not present, or if any of the checks fail, the request will immediately 404 with no other indication of what has happened.
func CheckAPIKey(c *gin.Context) {

	limiter := ratelimit.New(20)
	limiter.Take()

	// if the authorization header isn't present at all then we will 404
	if c.Request.Header.Get(("Authorization")) == "" {
		c.AbortWithStatus(404)
		return
	}

	// This is a bit verbose, but I've done this to avoid storing the api key in any kind of string type variable. Hopefully this makes a difference.
	keyEnclave := memguard.NewEnclave([]byte(strings.Split(c.Request.Header.Get("Authorization"), " ")[1]))

	// build a new APIKey from this key
	apiKey, err := plutus.APIKeyFromString(keyEnclave)
	// If we get an error here, it's because the client has supplied an invalid key (incorrect length in one or more components)
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	// If this is true, the key was either not found in our database (invalid), or it was and has been blacklisted. Either way, 404.
	if apiKey.Blacklisted == true {
		c.AbortWithStatus(404)
		return
	}

	// the key is valid, continue on with the request
	c.Next()
}

// RevokeAPIKey will take an API key in as JSON, and will of course have the CheckAPIKey route ran ahead of it's execution. The key included in the post body will be invalidated henceforth. Keys will not be given out by anyone except the admin of this service. Keys will not be automatically generated via any kind of API accessible to a client of this service. Only the admin will have the ability to provision API keys.
func RevokeAPIKey(c *gin.Context) {

	body := make(map[string]string)

	if err := c.ShouldBindJSON(&body); err != nil {
		c.AbortWithStatus(404)
		return
	}

	prefixEnclave := memguard.NewEnclave([]byte(strings.Split(body["key"], ".")[0]))
	delete(body, "key")

	apiKey, err := plutus.FindAPIKeyByPrefix(prefixEnclave)
	if err != nil {
		c.AbortWithStatus(404)
		return
	}

	err = apiKey.Revoke()
	if err != nil {
		c.AbortWithStatusJSON(500, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": "Key successfully revoked",
	})

}
