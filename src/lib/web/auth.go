package web

import (
	"encoding/json"
	"github.com/browsefile/backend/src/config"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	fb "github.com/browsefile/backend/src/lib"
	"github.com/dgrijalva/jwt-go"
	"github.com/dgrijalva/jwt-go/request"
)

const reCaptchaAPI = "/recaptcha/api/siteverify"

type cred struct {
	Password  string `json:"password"`
	Username  string `json:"username"`
	ReCaptcha string `json:"recaptcha"`
}

// reCaptcha checks the reCaptcha code.
func reCaptcha(host, secret, response string) (bool, error) {
	body := url.Values{}
	body.Set("secret", secret)
	body.Add("response", response)

	client := &http.Client{}

	resp, err := client.Post(host+reCaptchaAPI, "application/x-www-form-urlencoded", strings.NewReader(body.Encode()))
	if err != nil {
		return false, err
	}

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var data struct {
		Success bool `json:"success"`
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		return false, err
	}

	return data.Success, nil
}

// authHandler processes the authentication for the user.
func authHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	if c.FileBrowser.Config.Method == "none" {
		// NoAuth instances shouldn't call this method.
		return 0, nil
	} else if c.FileBrowser.Config.Method == "proxy" || c.FileBrowser.Config.Method == "ip" {
		isIp := c.FileBrowser.Config.Method == "ip"
		var uc *config.UserConfig
		var ok bool
		if isIp {
			uc, ok = c.Config.GetByIp(r.RemoteAddr)
		} else {
			uc, ok = c.Config.GetByUsername(r.Header.Get(c.FileBrowser.Config.Header))
		}

		// Receive the Username from the Header and check if it exists.
		if !ok {
			return http.StatusForbidden, nil
		}
		c.User = fb.ToUserModel(uc, c.Config)

		return printToken(c, w)
	}

	// Receive the credentials from the request and unmarshal them.
	var cred cred

	if r.Body == nil {
		return http.StatusForbidden, nil
	}

	err := json.NewDecoder(r.Body).Decode(&cred)
	if err != nil {
		return http.StatusForbidden, err
	}

	// If ReCaptcha is enabled, check the code.
	if len(c.ReCaptcha.Secret) > 0 {
		ok, err := reCaptcha(c.ReCaptcha.Host, c.ReCaptcha.Secret, cred.ReCaptcha)
		if err != nil {
			return http.StatusForbidden, err
		}
		if !ok {
			return http.StatusForbidden, nil
		}
	}

	uc, ok := c.Config.GetByUsername(cred.Username)
	if !ok{
		return http.StatusForbidden, nil
	}
	if !uc.IsGuest() {
		// Checks if the password is correct.
		if !ok || !fb.CheckPasswordHash(cred.Password, uc.Password) {
			return http.StatusForbidden, nil
		}
	}

	c.User = fb.ToUserModel(uc, c.Config)
	return printToken(c, w)
}

// renewAuthHandler is used when the front-end already has a JWT token
// and is checking if it is up to date. If so, updates its info.
func renewAuthHandler(c *fb.Context, w http.ResponseWriter, r *http.Request) (int, error) {
	ok, u := validateAuth(c, r)
	if !ok {
		return http.StatusForbidden, nil
	}
	c.User = u
	return printToken(c, w)
}

// claims is the JWT claims.
type claims struct {
	fb.UserModel
	jwt.StandardClaims
}

// printToken prints the final JWT token to the user.
func printToken(c *fb.Context, w http.ResponseWriter) (int, error) {
	// Creates a copy of the user and removes it password
	// hash so it never arrives to the user.
	u := fb.UserModel{}
	u = *c.User
	u.Password = ""

	// Builds the claims.
	claims := claims{
		u,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
			Issuer:    "Browse File",
		},
	}

	// Creates the token and signs it.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	k, err := c.Config.GetKeyBytes()
	if err != nil {
		return http.StatusInternalServerError, err
	}

	//expired
	if !claims.VerifyExpiresAt(time.Now().Add(time.Hour).Unix(), true) {
		w.Header().Add("X-Renew-Token", "true")
	}

	signed, err := token.SignedString(k)

	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Writes the token.
	w.Header().Set("Content-Type", "cty")
	w.Write([]byte(signed))
	return 0, nil
}

type extractor []string

func (e extractor) ExtractToken(r *http.Request) (string, error) {
	token, _ := request.HeaderExtractor{"X-Auth"}.ExtractToken(r)

	// Checks if the token isn't empty and if it contains two dots.
	// The former prevents incompatibility with URLs that previously
	// used basic auth.
	if token != "" && strings.Count(token, ".") == 2 {
		return token, nil
	}

	auth := r.URL.Query().Get("auth")
	if auth == "" {
		return "", request.ErrNoTokenInRequest
	}

	return auth, nil
}
func validateGuestAuth(c *fb.Context, r *http.Request) (bool, *fb.UserModel) {
	uc, _ := c.Config.GetByUsername(config.GUEST)
	return true, fb.ToUserModel(uc, c.Config)
}

// validateAuth is used to validate the authentication and returns the
// User if it is valid.
func validateAuth(c *fb.Context, r *http.Request) (bool, *fb.UserModel) {
	if c.Config.Method == "none" {
		admin := c.Config.GetAdmin()
		if admin == nil {
			return false, nil
		}

		c.User = fb.ToUserModel(admin, c.Config)
		return true, c.User
	}
	// If proxy auth is used do not verify the JWT token if the header is provided.
	if c.Config.Method == "proxy" {
		u, ok := c.Config.GetByUsername(r.Header.Get(c.Config.Header))
		if !ok {
			return false, nil
		}
		c.User = fb.ToUserModel(u, c.Config)
		return true, c.User
	}

	keyFunc := func(token *jwt.Token) (interface{}, error) {
		return c.Config.GetKeyBytes()
	}

	var claims claims
	var u *config.UserConfig
	var ok bool
	if c.Config.Method == "ip" {
		u, ok = c.Config.GetByIp(r.RemoteAddr)
		if !ok {
			return false, nil
		}

	} else {
		token, err := request.ParseFromRequest(r, extractor{}, keyFunc, request.WithClaims(&claims))

		if err != nil || !token.Valid {
			log.Println(err)
			return false, nil
		}

		u, ok = c.Config.GetByUsername(claims.Username)
		if !ok {
			return false, nil
		}
	}
	c.User = fb.ToUserModel(u, c.Config)
	return true, c.User

}
