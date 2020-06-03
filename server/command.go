package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/mattermost/mattermost-server/v5/plugin"
)

const (
	trigger = "community"

	waitText = "Please wait a bit"
)

// ExecuteCommand fetches contribution stats for a given repository or organistation and posts them in a message
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	commandArgs := strings.Split(strings.TrimSpace(strings.TrimPrefix(args.Command, "/"+trigger)), " ")
	command := commandArgs[0]
	commandArgs = commandArgs[1:]

	var appErr *model.AppError
	switch command {
	case "commiter":
		appErr = p.executeCommiterCommand(commandArgs, args)
	case "changelog":
		appErr = p.executeChangelogCommand(commandArgs, args)
	case "hackfest":
		appErr = p.executeHackfestCommand(commandArgs, args)
	case "new-commiter":
		appErr = p.executeNewCommiterCommand(commandArgs, args)
	default:
		return nil, &model.AppError{
			Id:         fmt.Sprintf("Unkown command %v", command),
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	if appErr != nil {
		return nil, appErr
	}

	return &model.CommandResponse{}, nil
}

// getCommand return the /community slash command
func getCommand() *model.Command {
	return &model.Command{
		Trigger:          trigger,
		DisplayName:      "Community",
		Description:      "Do community stuff",
		AutoComplete:     true,
		AutoCompleteDesc: "Available commands: commiter, changelog, hackfest",
		AutoCompleteHint: "[command]",
	}
}
