// Copyright 2015 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ipe

import (
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	log "github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"

	"github.com/isaac-mok/ipe/api"
	"github.com/isaac-mok/ipe/app"
	"github.com/isaac-mok/ipe/config"
	"github.com/isaac-mok/ipe/storage"
	"github.com/isaac-mok/ipe/websockets"
)

// Start Parse the configuration file and starts the ipe server
// It Panic if could not start the HTTP or HTTPS server
func Start(filename string) {
	var conf config.File

	rand.Seed(time.Now().Unix())

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Error(err)
		return
	}

	// Expand env vars
	data = []byte(os.ExpandEnv(string(data)))

	// Decoding config
	if err := yaml.UnmarshalStrict(data, &conf); err != nil {
		log.Error(err)
		return
	}

	// Using a in memory database
	inMemoryStorage := storage.NewInMemory()

	// Adding applications
	for _, a := range conf.Apps {
		application := app.NewApplication(
			a.Name,
			a.AppID,
			a.Key,
			a.Secret,
			a.OnlySSL,
			a.Enabled,
			a.UserEvents,
			a.WebHooks.Enabled,
			a.WebHooks.URL,
		)

		if err := inMemoryStorage.AddApp(application); err != nil {
			log.Error(err)
			return
		}
	}

	router := mux.NewRouter()
	router.Use(handlers.RecoveryHandler())

	router.Path("/app/{key}").Methods("GET").Handler(
		websockets.NewWebsocket(inMemoryStorage),
	)

	appsRouter := router.PathPrefix("/apps/{app_id}").Subrouter()
	appsRouter.Use(
		api.CheckAppDisabled(inMemoryStorage),
		api.Authentication(inMemoryStorage),
	)

	appsRouter.Path("/events").Methods("POST").Handler(
		api.NewPostEvents(inMemoryStorage),
	)
	appsRouter.Path("/channels").Methods("GET").Handler(
		api.NewGetChannels(inMemoryStorage),
	)
	appsRouter.Path("/channels/{channel_name}").Methods("GET").Handler(
		api.NewGetChannel(inMemoryStorage),
	)
	appsRouter.Path("/channels/{channel_name}/users").Methods("GET").Handler(
		api.NewGetChannelUsers(inMemoryStorage),
	)

	if conf.SSL.Enabled {
		go func() {
			log.Infof("Starting HTTPS service on %s ...", conf.SSL.Host)
			log.Fatal(http.ListenAndServeTLS(conf.SSL.Host, conf.SSL.CertFile, conf.SSL.KeyFile, router))
		}()
	}

	log.Infof("Starting HTTP service on %s ...", conf.Host)
	log.Fatal(http.ListenAndServe(conf.Host, router))
}
