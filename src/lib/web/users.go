package web

import (
	"encoding/json"
	"errors"
	"github.com/browsefile/backend/src/cnst"
	"net/http"
	"os"
	"strings"

	fb "github.com/browsefile/backend/src/lib"
)

type ModifyRequest struct {
	What  string `json:"what"`  // Answer to: what data type?
	Which string `json:"which"` // Answer to: which field?
}

type ModifyUserRequest struct {
	ModifyRequest
	Data *fb.UserModel `json:"data"`
}

// usersHandler is the entry point of the users API. It's just a router
// to send the request to its
func usersHandler(c *fb.Context) (int, error) {
	// If the user isn't admin and isn't making a PUT
	// request, then return forbidden.
	if !c.User.Admin && c.Method != http.MethodGet {
		return http.StatusForbidden, nil
	}

	switch c.Method {
	case http.MethodGet:
		return usersGetHandler(c)
	case http.MethodPost:
		return usersPostHandler(c)
	case http.MethodDelete:
		return usersDeleteHandler(c)
	case http.MethodPut:
		return usersPutHandler(c)
	}

	return http.StatusNotImplemented, nil
}

// getUserName returns the id from the user which is present
// in the request url. If the url is invalid and doesn't
// contain a valid ID, it returns an fb.Error.
func getUserName(r string) (string) {
	// Obtains the ID in string from the URL and converts
	// it into an integer.
	sid := strings.TrimPrefix(r, "/")
	sid = strings.TrimSuffix(sid, "/")

	return sid
}

// parseUserFromRequest returns the user which is present in the request
// body. If the body is empty or the JSON is invalid, it
// returns an fb.Error.
func parseUserFromRequest(c *fb.Context) (*fb.UserModel, string, error) {
	// Checks if the request body is empty.
	if c.REQ.Body == nil {
		return nil, "", cnst.ErrEmptyRequest
	}

	// Parses the request body and checks if it's well formed.
	mod := &ModifyUserRequest{}
	err := json.NewDecoder(c.REQ.Body).Decode(mod)
	if err != nil {
		return nil, "", err
	}

	// Checks if the request type is right.
	if mod.What != "user" {
		return nil, "", cnst.ErrWrongDataType
	}

	mod.Data.FileSystem = c.NewFS(c.GetUserHomePath())
	mod.Data.FileSystemPreview = c.NewFS(c.GetUserPreviewPath())
	return mod.Data, mod.Which, nil
}

func usersGetHandler(c *fb.Context) (int, error) {
	// Request for the default user data.
	if c.URL == "/base" {
		return renderJSON(c, c.Config.GetAdmin())
	}

	// Request for the listing of users.
	if c.URL == "/" {
		users := c.Config.GetUsers()
		if users == nil || len(users) == 0 {
			return http.StatusInternalServerError, errors.New("cant find any users")
		}

		for _, u := range users {
			// Removes the user password so it won't
			// be sent to the front-end.
			u.Password = ""
			//allow view users, in order to share
			if !c.User.Admin {
				u.UID = -1
				u.GID = -1
				u.IpAuth = nil
				u.Shares = nil
				u.ViewMode = ""
			}
		}

		return renderJSON(c, users)
	}

	name := getUserName(c.URL)
	u, ok := c.Config.GetUserByUsername(name)
	if !ok {
		return http.StatusNotFound, cnst.ErrNotExist
	}

	u.Password = ""
	return renderJSON(c, u)
}

func usersPostHandler(c *fb.Context) (int, error) {
	if c.URL != "/" {
		return http.StatusMethodNotAllowed, nil
	}

	u, _, err := parseUserFromRequest(c)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// Checks if username isn't empty.
	if u.Username == "" {
		return http.StatusBadRequest, cnst.ErrEmptyUsername
	}

	// Checks if password isn't empty.
	if u.Password == "" {
		return http.StatusBadRequest, cnst.ErrEmptyPassword
	}

	// Checks if the scope exists.
	if code, err := makeFS(c.Config.GetUserHomePath(u.Username)); err != nil {
		return code, err
	}

	// Hashes the password.
	pw, err := fb.HashPassword(u.Password)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	u.Password = pw
	u.ViewMode = cnst.MosaicViewMode

	// Saves the user to the database.
	err = c.Config.AddUser(u.UserConfig)
	if err == cnst.ErrExist {
		return http.StatusConflict, err
	}

	if err != nil {
		return http.StatusInternalServerError, err
	}

	// Set the Location header and return.
	c.RESP.Header().Set("Location", "/settings/users/"+u.Username)
	c.RESP.WriteHeader(http.StatusCreated)
	return http.StatusOK, nil
}

func makeFS(path string) (int, error) {
	info, err := os.Stat(path)

	if err != nil {
		if !os.IsNotExist(err) {
			return http.StatusInternalServerError, err
		}

		err = os.MkdirAll(path, cnst.PERM_DEFAULT)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		return 0, nil
	}

	if !info.IsDir() {
		return http.StatusBadRequest, errors.New("scope is not a dir, but should be")
	}

	return 0, nil
}

func usersDeleteHandler(c *fb.Context) (int, error) {
	if c.URL == "/" {
		return http.StatusMethodNotAllowed, nil
	}

	name := getUserName(c.URL)
	_, ok := c.Config.GetUserByUsername(name)
	if !ok {
		return http.StatusInternalServerError, cnst.ErrNotExist
	}

	// Deletes the user from the database.
	err := c.Config.DeleteUser(name)
	if err == cnst.ErrNotExist {
		return http.StatusNotFound, cnst.ErrNotExist
	} else if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}

func usersPutHandler(c *fb.Context) (int, error) {
	// New users should be created on /api/users.
	if c.URL == "/" {
		return http.StatusMethodNotAllowed, nil
	}

	// GetUsers the user ID from the URL and checks if it's valid.
	name := getUserName(c.URL)
	_, ok := c.Config.GetUserByUsername(name)
	if !ok {
		return http.StatusInternalServerError, cnst.ErrNotExist
	}

	// Checks if the user has permission to access this page.
	if !c.User.Admin && strings.Compare(name, c.User.Username) != 0 {
		return http.StatusForbidden, nil
	}

	// GetUsers the user from the request body.
	u, which, err := parseUserFromRequest(c)
	if err != nil {
		return http.StatusBadRequest, err
	}

	// If we're updating the default user. Only for NoAuth
	// implementations. Used to change the viewMode.
	cfgM := c.GetAuthConfig()
	if cfgM.AuthMethod == "none" {
		admin := c.Config.GetAdmin()
		admin.ViewMode = u.ViewMode
		err = c.Config.Update(admin)
		if err != nil {
			return http.StatusBadRequest, err
		}
		return http.StatusOK, nil
	}

	// Updates the CSS and locale.
	if which == "partial" || which == "locale" {
		if which == "locale" {
			c.User.Locale = u.Locale
		} else {
			c.User.ViewMode = u.ViewMode
		}

		err = c.Config.Update(c.User.UserConfig)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		return http.StatusOK, nil
	}

	// Updates the Password.
	if which == "password" {
		if u.Password == "" {
			return http.StatusBadRequest, cnst.ErrEmptyPassword
		}

		if strings.Compare(name, c.User.Username) != 0 && c.User.LockPassword {
			return http.StatusForbidden, nil
		}

		c.User.Password, err = fb.HashPassword(u.Password)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		err = c.Config.UpdatePassword(c.User.UserConfig)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		return http.StatusOK, nil
	}

	if which == "viewMode" {
		c.User.ViewMode = u.ViewMode
		err = c.Config.Update(c.User.UserConfig)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		return http.StatusOK, nil

	}

	/*	// If can only be all.
		if which != "all" {
			return http.StatusBadRequest, fb.ErrInvalidUpdateField
		}*/

	// Checks if username isn't empty.
	if u.Username == "" {
		return http.StatusBadRequest, cnst.ErrEmptyUsername
	}

	// Checks if the scope exists.
	if code, err := makeFS(c.Config.GetUserHomePath(u.Username)); err != nil {
		return code, err
	}
	// GetUsers the current saved user from the in-memory map.
	original, ok := c.Config.GetUserByUsername(name)
	if !ok {
		return http.StatusNotFound, nil
	}

	u.Username = name

	// Changes the password if the request wants it.
	if u.Password != "" {
		pw, err := fb.HashPassword(u.Password)
		if err != nil {
			return http.StatusInternalServerError, err
		}

		u.Password = pw
	} else {
		u.Password = original.Password
	}
	u.Shares = original.Shares

	// Updates the whole User struct because we always are supposed
	// to send a new entire object.
	err = c.Config.Update(u.UserConfig)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	return http.StatusOK, nil
}
