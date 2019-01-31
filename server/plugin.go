package main

import (
	"context"
	"sync"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	trigger        = "contributors"
	shortForm      = "2006-01-02"
	botUsername    = "community"
	botDisplayName = "Community Bot"
	botDescription = "TODO"
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
// creates a bot account, if it doesn't exist and registers the /contributors slash command
func (p *Plugin) OnActivate() error {
	config := p.getConfiguration()
	if config.Token == "" {
		return errors.New("Need to specify a personal access token")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	p.client = github.NewClient(tc)

	bot, appErr := p.API.GetUserByUsername(botUsername)
	if appErr != nil {
		if appErr.StatusCode != 404 {
			return appErr
		}
		newBot := &model.Bot{
			Username:    botUsername,
			DisplayName: botDisplayName,
			Description: botDescription,
		}
		rBot, appErr := p.API.CreateBot(newBot)
		if appErr != nil {
			return appErr
		}
		p.botUserID = rBot.UserId
	} else {
		p.botUserID = bot.Id
	}

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
	ephemeralPost.AddProp("from_webhook", "true")
	_ = p.API.SendEphemeralPost(userID, ephemeralPost)
}
