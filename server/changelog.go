package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/go-github/v25/github"
	"github.com/mattermost/mattermost-plugin-community/server/util"
	"github.com/mattermost/mattermost-server/model"
)

const shortForm = "2006-01"

func (p *Plugin) executeChangelogCommand(commandArgs []string, args *model.CommandArgs) *model.AppError {
	if len(commandArgs) != 2 {
		return &model.AppError{
			Id:         "Need two arguments",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	owner, repo, err := util.ParseOwnerAndRepository(commandArgs[0])
	if err != nil {
		return &model.AppError{
			Id:         err.Error(),
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	month, err := time.Parse(shortForm, commandArgs[1])
	if err != nil {
		return &model.AppError{
			Id:         "Failed to parse month",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	org, _, err := p.client.Organizations.Get(context.Background(), owner)
	if err != nil {
		p.API.LogWarn("failed to fetch organisation", "error", err.Error())
		return &model.AppError{
			Id:         "Failed to fetch data",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	topic := owner
	if repo != "" {
		topic += "/" + repo
	}

	attachments := []*model.SlackAttachment{{
		Title:      fmt.Sprintf("Fetching changelog for %v", month.Month().String()),
		Text:       waitText,
		AuthorName: topic,
		AuthorIcon: org.GetAvatarURL(),
		AuthorLink: fmt.Sprintf("https://github.com/%v", topic),
	}}

	loadingPost := &model.Post{
		ChannelId: args.ChannelId,
		UserId:    p.botUserID,
	}
	model.ParseSlackAttachment(loadingPost, attachments)

	loadingPost, appErr := p.API.CreatePost(loadingPost)
	if appErr != nil {
		return appErr
	}

	go p.updateChangelogPost(loadingPost, args.UserId, owner, repo, month)

	return nil
}

func (p *Plugin) updateChangelogPost(post *model.Post, userID, org, repo string, month time.Time) {
	// Fetch commits until the end of this month
	nextMonth := month.AddDate(0, 1, 0).Add(-time.Microsecond)

	var commits []*github.RepositoryCommit
	var err error
	if repo != "" {
		commits, err = p.fetchCommitsFromRepo(org, repo, month, nextMonth)
	} else {
		commits, err = p.fetchCommitsFromOrg(org, month, nextMonth)
	}
	if err != nil {
		p.API.LogError("failed to fetch data", "err", err.Error())

		var message string
		if _, ok := err.(*github.RateLimitError); ok {
			message = "Hit rate limit. Please try again later."
		} else {
			message = "Failed to fetch data. Please try again later. Error: " + err.Error()
		}
		post.Props["attachments"].([]*model.SlackAttachment)[0].Text = message
	} else {
		var commiter []string
		for _, c := range commits {
			author := c.GetAuthor()
			if author == nil {
				continue
			}
			u := author.GetLogin()
			if !util.Contains(commiter, u) {
				commiter = append(commiter, u)
			}
		}
		util.SortSlice(commiter)

		var commiterText string
		for i, c := range commiter {
			commiterText += fmt.Sprintf("[%[1]s](https://github.com/%[1]v)", c)
			if i+1 != len(commiter) {
				commiterText += ", "
			}
		}

		attachment := post.Props["attachments"].([]*model.SlackAttachment)[0]
		attachment.Title = fmt.Sprintf("Commiter list for %v changelog", month.Month().String())
		attachment.Text = ""
		attachment.Fields = []*model.SlackAttachmentField{{
			Title: "Number of Commiter",
			Value: strconv.Itoa(len(commiter)),
		}, {
			Title: "Commiter",
			Value: "```\n" + commiterText + "\n```",
		}}
	}

	if _, appErr := p.API.UpdatePost(post); appErr != nil {
		p.SendEphemeralPost(post.ChannelId, userID, "Something went bad. Please try again.")
		p.API.LogError("failed to update post", "err", appErr.Error())
		return
	}
}
