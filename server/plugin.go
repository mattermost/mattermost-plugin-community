package main

import (
	"sync"

	"github.com/google/go-github/v25/github"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

const (
	botUsername    = "community"
	botDisplayName = "Community Bot"
	botDescription = "Created by the Community plugin."
)

// Plugin is the object to run the plugin
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration

	// botUserID is the ID of the community bot user
	botUserID string

	// client used to fetch data from the GitHub API
	client *github.Client
}

// OnActivate creates a github client with the access token from the configurations,
// creates a bot account, if it doesn't exist and registers the /community slash command
func (p *Plugin) OnActivate() error {
	bot := &model.Bot{
		Username:    botUsername,
		DisplayName: botDisplayName,
		Description: botDescription,
	}
	botUserID, appErr := p.Helpers.EnsureBot(bot)
	if appErr != nil {
		return errors.Wrap(appErr, "failed to ensure bot user")
	}
	p.botUserID = botUserID

	err := p.API.RegisterCommand(getCommand())
	if err != nil {
		return errors.Wrap(err, "failed to register new command")
	}

	return nil
}

// SendEphemeralPost sends a ephemeral message in a given channel to a given user
func (p *Plugin) SendEphemeralPost(channelID, userID, message string) {
	// This is mostly taken from https://github.com/mattermost/mattermost-server/blob/master/app/command.go#L304
	ephemeralPost := &model.Post{}
	ephemeralPost.ChannelId = channelID
	ephemeralPost.UserId = p.botUserID
	ephemeralPost.Message = message
	_ = p.API.SendEphemeralPost(userID, ephemeralPost)
}
