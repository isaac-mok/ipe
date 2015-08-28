// Copyright 2014 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ipe

import (
	"net/http"
	"net/url"
	"sort"
	"strings"

	log "github.com/golang/glog"
	"github.com/gorilla/mux"

	"github.com/dimiro1/ipe/utils"
)

// Prepare Querystring
func prepareQueryString(params url.Values) string {
	var keys []string

	for key := range params {
		keys = append(keys, strings.ToLower(key))
	}

	sort.Strings(keys)

	var pieces []string

	for _, key := range keys {
		pieces = append(pieces, key+"="+params.Get(key))
	}

	return strings.Join(pieces, "&")
}

// Authenticate pusher
// see: https://gist.github.com/mloughran/376898
//
// The signature is a HMAC SHA256 hex digest.
// This is generated by signing a string made up of the following components concatenated with newline characters \n.
//
//  * The uppercase request method (e.g. POST)
//  * The request path (e.g. /some/resource)
//  * The query parameters sorted by key, with keys converted to lowercase, then joined as in the query string.
//    Note that the string must not be url escaped (e.g. given the keys auth_key: foo, Name: Something else, you get auth_key=foo&name=Something else)
func restAuthenticationHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		appID := vars["app_id"]

		app, err := conf.GetAppByAppID(appID)

		if err != nil {
			log.Error(err)
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}

		params := r.URL.Query()

		signature := params.Get("auth_signature")
		params.Del("auth_signature")

		queryString := prepareQueryString(params)

		toSign := strings.ToUpper(r.Method) + "\n" + r.URL.Path + "\n" + queryString

		if utils.HashMAC([]byte(toSign), []byte(app.Secret)) == signature {
			h.ServeHTTP(w, r)
		} else {
			log.Error("Not authorized")
			http.Error(w, "Not authorized", http.StatusUnauthorized)
		}
	})
}
