package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
)

// ExecuteCommand fetches contribution stats for a given repository or organistation and posts them in a message
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	channelID := args.ChannelId
	userID := args.UserId
	command := args.Command

	command = strings.TrimPrefix(command, "/"+trigger)
	command = strings.TrimSpace(command)
	commandArgs := strings.Split(command, " ")

	if len(commandArgs) != 4 {
		return nil, &model.AppError{
			Id:         "Need four arguments",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}
	orgName := commandArgs[0]

	var repo string
	if commandArgs[1] != "all" {
		repo = commandArgs[1]
	}

	since, err := time.Parse(shortForm, commandArgs[2])
	if err != nil {
		return nil, &model.AppError{
			Id:         "Failed to parse since time",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	until, err := time.Parse(shortForm, commandArgs[3])
	if err != nil {
		return nil, &model.AppError{
			Id:         "Failed to parse until time",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	org, _, err := p.client.Organizations.Get(context.Background(), orgName)
	if err != nil {
		return nil, &model.AppError{
			Id:         "Failed to fetch data",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	topic := orgName
	if repo != "" {
		topic += "/" + repo
	}

	headline := " between " + since.Format(shortForm) + " and " + until.Format(shortForm)

	attachments := []*model.SlackAttachment{{
		Title:      "Fetching contributor stats" + headline,
		Text:       "Please wait a few minutes.",
		AuthorName: topic,
		AuthorIcon: org.GetAvatarURL(),
		AuthorLink: fmt.Sprintf("https://github.com/%v", topic),
	}}

	post := &model.Post{
		ChannelId: channelID,
		UserId:    p.botUserID,
	}
	model.ParseSlackAttachment(post, attachments)

	post, appErr := p.API.CreatePost(post)
	if appErr != nil {
		return nil, appErr
	}

	go func() {
		var contributors sort.StringSlice
		var numberOfCommits uint64
		if repo != "" {
			contributors, numberOfCommits, err = p.fetchRepo(orgName, repo, since, until)
		} else {
			contributors, numberOfCommits, err = p.fetchAllRepos(orgName, since, until)
		}
		if err != nil {
			var message string
			if _, ok := err.(*github.RateLimitError); ok {
				message = "Hit rate limit. Please try again later."
			} else {
				message = "Failed to fetch data:" + err.Error()
			}

			p.API.LogError("failed to fetch data", "err", err.Error())
			post.Props["attachments"].([]*model.SlackAttachment)[0].Text = message
		} else {
			contributors.Sort()
			var contributorsText string
			for i, c := range contributors {
				contributorsText += fmt.Sprintf("[%[1]s](https://github.com/%[1]v)", c)
				if i+1 != len(contributors) {
					contributorsText += ", "
				}
			}

			attachment := post.Props["attachments"].([]*model.SlackAttachment)[0]
			attachment.Title = "Contributor stats" + headline
			attachment.Text = ""
			attachment.Fields = []*model.SlackAttachmentField{{
				Title: "Number of commits",
				Value: strconv.FormatUint(numberOfCommits, 10),
			}, {
				Title: "Number of Contributors",
				Value: strconv.Itoa(len(contributors)),
			}, {
				Title: "Contributors",
				Value: "```\n" + contributorsText + "\n```",
			}}
		}
		if _, appErr := p.API.UpdatePost(post); appErr != nil {
			p.SendEphemeralPost(channelID, userID, "Something went bad. Please try again.")
			p.API.LogError("failed to update post", "err", err.Error())
			return
		}
	}()

	return &model.CommandResponse{}, nil
}

// getCommand return the /contributors slash command
func getCommand() *model.Command {
	return &model.Command{
		Trigger:          trigger,
		DisplayName:      "Contributors",
		Description:      "List contributors",
		AutoComplete:     true,
		AutoCompleteDesc: "Get a list of contributors",
		AutoCompleteHint: `[organisation] [repo] [since] [until]`,
	}
}
