package goTap

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
)

// Accounts defines a key/value for user/pass list of authorized logins
type Accounts map[string]string

// BasicAuthPair represents a user/password pair for BasicAuth
type BasicAuthPair struct {
	Username string
	Password string
}

// BasicAuth returns a Basic HTTP Authorization middleware
// It takes a map[string]string where the key is the username and the value is the password,
// as well as the name of the Realm.
// If the realm is empty string, "Authorization Required" will be used by default.
func BasicAuth(accounts Accounts) HandlerFunc {
	return BasicAuthForRealm(accounts, "Authorization Required")
}

// BasicAuthForRealm returns a Basic HTTP Authorization middleware
// It takes a map[string]string where the key is the username and the value is the password,
// as well as the name of the Realm.
func BasicAuthForRealm(accounts Accounts, realm string) HandlerFunc {
	if realm == "" {
		realm = "Authorization Required"
	}

	// Preprocess accounts to avoid doing it on every request
	pairs := processAccounts(accounts)

	return func(c *Context) {
		// Search user in the credentials
		user, found := pairs.searchCredential(c.Request.Header.Get("Authorization"))
		if !found {
			// Credentials doesn't match, we return 401 and abort handlers chain
			c.Header("WWW-Authenticate", `Basic realm="`+realm+`"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		// The user credentials were found, set user's id to key "user" in this context
		c.Set("user", user)
		c.Next()
	}
}

type authPairs []BasicAuthPair

func (a authPairs) searchCredential(authValue string) (string, bool) {
	if authValue == "" {
		return "", false
	}

	if !strings.HasPrefix(authValue, "Basic ") {
		return "", false
	}

	// Decode base64 credentials
	payload, err := base64.StdEncoding.DecodeString(authValue[6:])
	if err != nil {
		return "", false
	}

	// Split username:password
	pair := strings.SplitN(string(payload), ":", 2)
	if len(pair) != 2 {
		return "", false
	}

	// Use constant time comparison to prevent timing attacks
	for _, p := range a {
		if subtle.ConstantTimeCompare([]byte(p.Username), []byte(pair[0])) == 1 &&
			subtle.ConstantTimeCompare([]byte(p.Password), []byte(pair[1])) == 1 {
			return p.Username, true
		}
	}

	return "", false
}

func processAccounts(accounts Accounts) authPairs {
	pairs := make(authPairs, 0, len(accounts))
	for user, password := range accounts {
		pairs = append(pairs, BasicAuthPair{
			Username: user,
			Password: password,
		})
	}
	return pairs
}
