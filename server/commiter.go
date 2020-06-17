package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/google/go-github/v25/github"
	"github.com/mattermost/mattermost-plugin-community/server/util"
	"github.com/mattermost/mattermost-server/v5/model"
)

const shortFormWithDay = "2006-01-02"

func (p *Plugin) executeCommiterCommand(commandArgs []string, args *model.CommandArgs) *model.AppError {
	if len(commandArgs) != 3 {
		return &model.AppError{
			Id:         "Need three arguments",
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

	since, err := time.Parse(shortFormWithDay, commandArgs[1])
	if err != nil {
		return &model.AppError{
			Id:         fmt.Sprintf("Failed to parse since time: %v", err.Error()),
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	until, err := time.Parse(shortFormWithDay, commandArgs[2])
	if err != nil {
		return &model.AppError{
			Id:         fmt.Sprintf("Failed to parse until time: %v", err.Error()),
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	isOrg, err := p.verifyOrg(owner)
	if err != nil {
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
		Title:      "Fetching commiter stats between " + since.Format(shortFormWithDay) + " and " + until.Format(shortFormWithDay),
		Text:       waitText,
		AuthorName: topic,
		AuthorIcon: p.GetAvatarLogo(owner, isOrg),
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

	go p.updateCommitersPost(loadingPost, args.UserId, owner, repo, isOrg, since, until)

	return nil
}

func (p *Plugin) updateCommitersPost(post *model.Post, userID, org, repo string, isOrg bool, since, until time.Time) {
	// Fetch commits until one day after at midnight
	fetchUntil := until.AddDate(0, 0, 1).Add(-time.Microsecond)

	var commits []*github.RepositoryCommit
	var err error
	if repo != "" {
		commits, err = p.fetchCommitsFromRepo(org, repo, since, fetchUntil)
	}
	if isOrg {
		commits, err = p.fetchCommitsFromOrg(org, since, fetchUntil)
	}
	commits, err = p.fetchCommitsFromUser(org, since, fetchUntil)
	if err != nil {
		p.API.LogError("failed to fetch data", "err", err.Error())

		var message string
		if _, ok := err.(*github.RateLimitError); ok {
			message = "Hit rate limit. Please try again later."
		} else {
			message = "Failed to fetch data:" + err.Error()
		}
		post.Props["attachments"].([]*model.SlackAttachment)[0].Text = message
	} else {
		commiter := map[string]int{}
		for _, c := range commits {
			author := c.GetAuthor()
			if author == nil {
				continue
			}
			u := author.GetLogin()
			commiter[u]++
		}

		type kv struct {
			Key   string
			Value int
		}

		var ss []kv
		for k, v := range commiter {
			ss = append(ss, kv{k, v})
		}

		sort.Slice(ss, func(i, j int) bool {
			return ss[i].Value > ss[j].Value
		})

		var commiterText string
		for _, e := range ss {
			var c string
			if e.Value > 1 {
				c = "commits"
			} else {
				c = "commit"
			}
			commiterText += fmt.Sprintf("- [%[1]s](https://github.com/%[1]v): %v %v\n", e.Key, e.Value, c)
		}

		attachment := post.Props["attachments"].([]*model.SlackAttachment)[0]
		attachment.Title = "Commiter stats between " + since.Format(shortFormWithDay) + " and " + until.Format(shortFormWithDay)
		attachment.Text = ""
		attachment.Fields = []*model.SlackAttachmentField{{
			Title: "Number of commits",
			Value: strconv.Itoa(len(commits)),
		}, {
			Title: "Number of Commiter",
			Value: strconv.Itoa(len(commiter)),
		}, {
			Title: "Commiter",
			Value: commiterText,
		}}
	}

	if _, appErr := p.API.UpdatePost(post); appErr != nil {
		p.SendEphemeralPost(post.ChannelId, userID, "Something went bad. Please try again.")
		p.API.LogError("failed to update post", "err", appErr.Error())
		return
	}
}

func (p *Plugin) verifyOrg(owner string) (bool, error) {
	_, _, err := p.client.Organizations.Get(context.Background(), owner)
	if err == nil {
		return true, nil
	}
	_, _, err = p.client.Users.Get(context.Background(), owner)
	if err == nil {
		return false, nil
	} else {
		return true, nil
	}

}

func (p *Plugin) GetAvatarLogo(owner string, isOrg bool) string {
	if isOrg {
		org, _, _ := p.client.Organizations.Get(context.Background(), owner)
		return org.GetAvatarURL()
	} else {
		user, _, _ := p.client.Users.Get(context.Background(), owner)
		return user.GetAvatarURL()
	}
}
