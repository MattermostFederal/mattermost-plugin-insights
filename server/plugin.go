package main

import (
	"net/http"
	"sync"

	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/pkg/errors"

	"github.com/MattermostFederal/mattermost-plugin-insights/server/api"
	"github.com/MattermostFederal/mattermost-plugin-insights/server/store"
)

type Plugin struct {
	plugin.MattermostPlugin

	configurationLock sync.RWMutex
	configuration     *configuration

	client *pluginapi.Client
	store  *store.Store
	api    *api.API
}

func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)

	st, err := store.New(p.client)
	if err != nil {
		return errors.Wrap(err, "failed to initialize insights store")
	}
	p.store = st
	p.api = api.FromPluginAPI(p.client, p.store)

	return p.API.RegisterCommand(getCommand())
}

func (p *Plugin) ServeHTTP(_ *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.api.ServeHTTP(w, r)
}
