package main

import (
	"context"
	"net/http"

	"github.com/google/go-github/v31/github"
	"github.com/mattermost/mattermost-plugin-github/server/client"
	"github.com/mattermost/mattermost-server/v5/model"
	"golang.org/x/oauth2"
)

const (
	gitHubPluginID = "github"
)

func (p *Plugin) getGitHubClient(userID string) (*github.Client, error) {
	status, appErr := p.API.GetPluginStatus(gitHubPluginID)
	if appErr != nil {
		p.API.LogDebug("Failed to fetch status of GitHub plugin", "error", appErr.Error())
	}

	if appErr != nil || status.State != model.PluginStateRunning {
		p.API.LogDebug("GitHub plugin is not running. Falling back to config token.")

		configuration := p.getConfiguration()
		var tc *http.Client
		if configuration.Token != "" {
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: configuration.Token},
			)
			tc = oauth2.NewClient(context.Background(), ts)
		}
		return github.NewClient(tc), nil
	}

	p.API.LogDebug("Using token from GitHub plugin.")

	client := client.NewPluginClient(p.API)

	ghClient, err := client.GetGitHubClient(userID)
	if err != nil {
		return nil, err
	}

	return ghClient, nil
}
