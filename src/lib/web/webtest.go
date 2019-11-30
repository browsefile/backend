package web

import (
	"bytes"
	"encoding/json"
	"github.com/browsefile/backend/src/cnst"
	"github.com/browsefile/backend/src/config"
	"github.com/browsefile/backend/src/lib"
	"github.com/dgrijalva/jwt-go"
	"golang.org/x/net/http2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

type TServContext struct {
	*config.TContext
}

//This method will create token, and set as X-Auth header for specific user, if the user nil, it will reuse existing token
//r- router
//files paths for files param
//u url for current working dir
//usr user make request from

func (tc *TServContext) MakeRequest(r int, params map[string]interface{}, usr *config.UserConfig, t *testing.T, isShare bool) (*http.Request, *http.Response, *http.Transport) {
	if usr != nil {
		// Builds the claims.
		claims := Claims{
			*lib.ToUserModel(usr, tc.GlobalConfig),
			jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
				Issuer:    "Browse File",
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		k, err := tc.GlobalConfig.GetKeyBytes()
		if err != nil {
			t.Fatal(err)
		}
		tc.Token, err = token.SignedString(k)
		if err != nil {
			t.Fatal(err)
		}
	} else if len(tc.Token) == 0 {
		t.Error("user or token must be present, even for guest user")
	}
	var method string
	if _, isMethod := params["method"]; isMethod {
		method = params["method"].(string)
	} else {
		method = http.MethodGet
	}
	var req *http.Request
	if _, isBody := params["body"]; isBody {
		req, _ = http.NewRequest(method, "", params["body"].(*bytes.Buffer))
	} else {
		req, _ = http.NewRequest(method, "", nil)
	}
	if ct, ok := params["content-type"]; ok {
		req.Header.Set("Content-Type", ct.(string))
	}

	req.Header.Set(cnst.H_XAUTH, tc.Token)
	req.URL = tc.BuildUrl(r, params, isShare)
	tr := &http.Transport{}
	t.Log(req.Method, req.URL.String())
	res, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	return req, res, tr
}
func (tc *TServContext) BuildUrl(r int, params map[string]interface{}, isShare bool) *url.URL {
	parsedURL := tc.Srv.URL + "/api"

	urlSuf := params["u"].(string)
	q := url.Values{}

	if _, rh := params[cnst.P_EXSHARE]; isShare && rh {
		q.Set(cnst.P_EXSHARE, params[cnst.P_EXSHARE].(string))
	}

	switch r {
	case cnst.R_PLAYLIST:
		if isShare {
			parsedURL += "/shares"
		}
		parsedURL += "/download" + urlSuf

		if _, isFiles := params["files"]; isFiles {
			q.Set("files", strings.Join(params["files"].([]string)[:], ","))
		}
		q.Set("algo", "m3u_vai")
	case cnst.R_SEARCH:
		parsedURL += "/search"
		if isShare {
			parsedURL += "/shares"
		}
		parsedURL += urlSuf
		q.Set("query", params["query"].(string))
	case cnst.R_DOWNLOAD:
		if isShare {
			parsedURL += "/shares"
		}
		parsedURL += "/download" + urlSuf
		_, isFiles := params["files"]
		if isFiles {
			q.Set("files", strings.Join(params["files"].([]string)[:], ","))
		}
		q.Set("algo", "zip")
	case cnst.R_RESOURCE:
		parsedURL += "/resource" + urlSuf
		if ov, ok := params["override"]; ok {
			q.Set("override", ov.(string))
		}
		if dst, ok := params["destination"]; ok {
			q.Set("destination", dst.(string))
		}
	case cnst.R_SHARES:
		parsedURL += "/shares" + urlSuf
		if share, ok := params["share"]; ok {
			q.Set("share", share.(string))
		}
	case cnst.R_USERS:
		parsedURL += "/users" + urlSuf
	}
	_, ok := params[cnst.P_PREVIEW_TYPE]
	if ok {
		q.Set(cnst.P_PREVIEW_TYPE, params[cnst.P_PREVIEW_TYPE].(string))
	}

	resURL, _ := url.Parse(parsedURL)
	resURL.RawQuery = q.Encode()
	return resURL
}

//run internal server for testing and set specific variables at TContext
func (tc *TServContext) InitServ(t *testing.T) {
	cfg := &config.TContext{}
	cfg.InitWithUsers(t)

	shrUp := &config.ShareItem{Path: cfg.SharePathUp, AllowLocal: true, AllowExternal: true}
	cfg.Usr1.AddShare(shrUp)

	//add shares to user1
	shrDeep := &config.ShareItem{Path: cfg.SharePathDeep, AllowUsers: []string{"admin"}}
	cfg.Usr1.AddShare(shrDeep)
	_ = cfg.Update(cfg.Usr1)
	cfg.WriteConfig()

	cfg.Srv = httptest.NewServer(SetupHandler(cfg.GlobalConfig))
	cfg.Tr = &http.Transport{}
	_ = http2.ConfigureTransport(cfg.Tr)
	tc.TContext = cfg
}
func (tc *TServContext) ValidateDownloadLink(l string, t *testing.T) bool {
	req, err := http.NewRequest("GET", l, nil)
	if err != nil {
		t.Error(l, err)
	}
	req.Header.Set(cnst.H_XAUTH, tc.Token)
	tr := &http.Transport{}
	res, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode == http.StatusOK
}
func CheckLink(f lib.File, dat map[string]interface{}, cfg TServContext, t *testing.T, isShare, reqPreview bool) {

	for _, itm := range f.Items {
		if itm.IsDir {
			dat["u"] = itm.URL + "/"
			delete(dat, "files")
		} else {
			dat["u"] = "/"
			dat["files"] = []string{itm.URL + "/"}
		}

		url := cfg.BuildUrl(cnst.R_DOWNLOAD, dat, isShare)
		if !cfg.ValidateDownloadLink(url.String(), t) {
			t.Fatal("not valid url :", url.String())
		}
		if reqPreview && (itm.Type == "image" || itm.Kind == "Type") {
			dat[cnst.P_PREVIEW_TYPE] = "thumb"
			url = cfg.BuildUrl(cnst.R_DOWNLOAD, dat, isShare)
			if !cfg.ValidateDownloadLink(url.String(), t) {
				t.Fatal("not valid preview url :", url.String())
			}
			dat[cnst.P_PREVIEW_TYPE] = ""
		}

	}
}
func ValidateListingResp(rs *http.Response, t *testing.T, count int) *lib.File {

	if rs.StatusCode != http.StatusOK {
		t.Error("wrong listing status at link :", rs.Request.URL.String())
	}
	f := &lib.File{}

	err := json.NewDecoder(rs.Body).Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	l := len(f.Items)
	if l != count {
		t.Fatal("items count must be", count, "but actual count", l)
	}

	return f

}
