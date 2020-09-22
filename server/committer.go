package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/google/go-github/v31/github"
	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-plugin-community/server/util"
)

const shortFormWithDay = "2006-01-02"

func (p *Plugin) executeCommitterCommand(commandArgs []string, args *model.CommandArgs) *model.AppError {
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

	client, err := p.getGitHubClient(args.UserId)
	if err != nil {
		p.API.LogWarn("Failed to create GitHub client", "error", err.Error())

		return &model.AppError{
			Id:         "Failed to connect to GitHub.",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	isOrg, err := p.verifyOrg(client, owner)
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

	avatarLogo, err := p.GetAvatarLogo(client, owner, isOrg)
	if err != nil {
		avatarLogo = ""
		p.API.LogError(err.Error())
	}

	attachments := []*model.SlackAttachment{{
		Title:      "Fetching committer stats between " + since.Format(shortFormWithDay) + " and " + until.Format(shortFormWithDay),
		Text:       waitText,
		AuthorName: topic,
		AuthorIcon: avatarLogo,
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

	go p.updateCommittersPost(client, loadingPost, args.UserId, owner, repo, isOrg, since, until)

	return nil
}

func (p *Plugin) updateCommittersPost(client *github.Client, post *model.Post, userID, org, repo string, isOrg bool, since, until time.Time) {
	// Fetch commits until one day after at midnight
	fetchUntil := until.AddDate(0, 0, 1).Add(-time.Microsecond)

	var commits []*github.RepositoryCommit
	var err error

	switch {
	case repo != "":
		commits, err = p.fetchCommitsFromRepo(client, org, repo, since, fetchUntil)
	case isOrg:
		commits, err = p.fetchCommitsFromOrg(client, org, since, fetchUntil)
	case !isOrg:
		commits, err = p.fetchCommitsFromUser(client, org, since, fetchUntil)
	}

	if err != nil {
		p.API.LogError("failed to fetch data", "err", err.Error())

		message := githubErrorHandle(err)
		post.Props["attachments"].([]*model.SlackAttachment)[0].Text = message
	} else {
		committer := map[string]int{}
		for _, c := range commits {
			author := c.GetAuthor()
			if author == nil {
				continue
			}
			u := author.GetLogin()
			committer[u]++
		}

		type kv struct {
			Key   string
			Value int
		}

		var ss []kv
		for k, v := range committer {
			ss = append(ss, kv{k, v})
		}

		sort.Slice(ss, func(i, j int) bool {
			return ss[i].Value > ss[j].Value
		})

		var committerText string
		for _, e := range ss {
			var c string
			if e.Value > 1 {
				c = "commits"
			} else {
				c = "commit"
			}
			committerText += fmt.Sprintf("- [%[1]s](https://github.com/%[1]v): %v %v\n", e.Key, e.Value, c)
		}

		attachment := post.Props["attachments"].([]*model.SlackAttachment)[0]
		attachment.Title = "Committer stats between " + since.Format(shortFormWithDay) + " and " + until.Format(shortFormWithDay)
		attachment.Text = ""
		attachment.Fields = []*model.SlackAttachmentField{{
			Title: "Number of commits",
			Value: strconv.Itoa(len(commits)),
		}, {
			Title: "Number of Committer",
			Value: strconv.Itoa(len(committer)),
		}, {
			Title: "Committer",
			Value: committerText,
		}}
	}

	if _, appErr := p.API.UpdatePost(post); appErr != nil {
		p.SendEphemeralPost(post.ChannelId, userID, "Something went bad. Please try again.")
		p.API.LogError("failed to update post", "err", appErr.Error())
		return
	}
}

func (p *Plugin) verifyOrg(client *github.Client, owner string) (bool, error) {
	_, _, err := client.Organizations.Get(context.Background(), owner)
	if err == nil {
		return true, nil
	}
	_, _, err = client.Users.Get(context.Background(), owner)
	if err == nil {
		return false, nil
	}

	return true, fmt.Errorf("unable to find GitHub Organization, or User with matching owner: %s", owner)
}

// GetAvatarLogo fetches the AvatarLogo from respective Github Organization or User
func (p *Plugin) GetAvatarLogo(client *github.Client, owner string, isOrg bool) (string, error) {
	if isOrg {
		org, _, err := client.Organizations.Get(context.Background(), owner)
		if err != nil {
			return "", fmt.Errorf("unable to find GitHub Organization with matching owner: %s", owner)
		}
		return org.GetAvatarURL(), nil
	}

	user, _, err := client.Users.Get(context.Background(), owner)
	if err != nil {
		return "", fmt.Errorf("unable to find GitHub User with matching owner: %s", owner)
	}
	return user.GetAvatarURL(), nil
}
