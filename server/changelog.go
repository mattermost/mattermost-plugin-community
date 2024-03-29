package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/go-github/v31/github"
	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-plugin-community/server/util"
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

	client, err := p.getGitHubClient(args.UserId)
	if err != nil {
		p.API.LogWarn("Failed to create GitHub client", "error", err.Error())

		return &model.AppError{
			Id:         "Failed to connect to GitHub.",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	org, _, err := client.Organizations.Get(context.Background(), owner)
	if err != nil {
		p.API.LogWarn("Failed to fetch organization", "error", err.Error())
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
		Title:      fmt.Sprintf("Fetching changelog for %v %v", month.Month().String(), month.Year()),
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

	go p.updateChangelogPost(client, loadingPost, args.UserId, owner, repo, month)

	return nil
}

func (p *Plugin) updateChangelogPost(client *github.Client, post *model.Post, userID, org, repo string, month time.Time) {
	// Fetch commits until the end of this month
	nextMonth := month.AddDate(0, 1, 0).Add(-time.Microsecond)

	var commits []*github.RepositoryCommit
	var err error
	if repo != "" {
		commits, err = p.fetchCommitsFromRepo(client, org, repo, month, nextMonth)
	} else {
		commits, err = p.fetchCommitsFromOrg(client, org, month, nextMonth)
	}
	if err != nil {
		p.API.LogError("Failed to fetch data", "err", err.Error())

		message := githubErrorHandle(err)
		post.Props["attachments"].([]*model.SlackAttachment)[0].Text = message
	} else {
		var committer []string
		for _, c := range commits {
			author := c.GetAuthor()
			if author == nil {
				continue
			}
			u := author.GetLogin()
			if !util.Contains(committer, u) {
				committer = append(committer, u)
			}
		}
		util.SortSlice(committer)

		const userPerPost = 150
		committerTexts := make([]string, len(committer)/userPerPost+1)
		for i, c := range committer {
			profile := fmt.Sprintf("[%[1]s](https://github.com/%[1]v)", c)
			if i+1 != len(committer) {
				profile += ", "
			}
			committerTexts[i/150] += profile
		}

		attachment := post.Props["attachments"].([]*model.SlackAttachment)[0]
		attachment.Title = fmt.Sprintf("Committer list for %v %v changelog", month.Month().String(), month.Year())
		attachment.Text = ""
		attachment.Fields = []*model.SlackAttachmentField{{
			Title: "Number of Committer",
			Value: strconv.Itoa(len(committer)),
		}, {
			Title: "Committer",
			Value: "```\n" + committerTexts[0] + "\n```",
		}}

		for i := 1; i < len(committerTexts); i++ {
			attachment := *attachment
			attachment.Title += fmt.Sprintf(" (Part %v)", i+1)
			attachment.Fields = []*model.SlackAttachmentField{{
				Title: "Committer",
				Value: "```\n" + committerTexts[i] + "\n```",
			}}

			additionalPost := &model.Post{
				ChannelId: post.ChannelId,
				UserId:    post.UserId,
			}
			model.ParseSlackAttachment(additionalPost, []*model.SlackAttachment{&attachment})

			_, appErr := p.API.CreatePost(additionalPost)
			if appErr != nil {
				p.API.LogError("Failed to create additional changelog post", "err", appErr.Error())
			}
		}
	}

	if _, appErr := p.API.UpdatePost(post); appErr != nil {
		p.SendEphemeralPost(post.ChannelId, userID, "Something went bad. Please try again.")
		p.API.LogError("Failed to update post", "err", appErr.Error())
		return
	}
}

func githubErrorHandle(err error) string {
	var message string
	if _, ok := err.(*github.RateLimitError); ok {
		message = rateLimitMessage
	} else {
		message = "Failed to fetch data:" + err.Error()
	}
	return message
}
