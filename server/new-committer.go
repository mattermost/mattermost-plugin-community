package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/google/go-github/v25/github"
	"github.com/mattermost/mattermost-server/v5/model"
)

type firstContributionInfo struct {
	author string
	date   time.Time
	commit string
	org    string
	repo   string
}

const rateLimitMessage = "Hit rate limit. Please try again later."

func (p *Plugin) executeNewCommitterCommand(commandArgs []string, args *model.CommandArgs) *model.AppError {
	if len(commandArgs) != 2 {
		return &model.AppError{
			Id:         "Need two arguments",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	organization := commandArgs[0]

	since, err := time.Parse(shortFormWithDay, commandArgs[1])
	if err != nil {
		return &model.AppError{
			Id:         fmt.Sprintf("Failed to parse since time: %v", err.Error()),
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	org, _, err := p.client.Organizations.Get(context.Background(), organization)
	if err != nil {
		return &model.AppError{
			Id:         "Failed to fetch data",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	attachments := []*model.SlackAttachment{{
		Title:      "Fetching new commiters since " + since.Format(shortFormWithDay),
		Text:       waitText,
		AuthorName: organization,
		AuthorIcon: org.GetAvatarURL(),
		AuthorLink: fmt.Sprintf("https://github.com/%v", organization),
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

	go p.updateNewCommittersPost(loadingPost, args.UserId, organization, since)

	return nil
}

func (p *Plugin) updateNewCommittersPost(post *model.Post, userID, org string, since time.Time) {
	contributors, err := p.fetchContributors(org)
	if err != nil {
		p.logAndPropUserAboutError(post, userID, err)
		return
	}

	firstContributions, err := p.findFirstContributions(contributors, org, since)
	if err != nil {
		p.logAndPropUserAboutError(post, userID, err)
		return
	}

	var result []firstContributionInfo
	for _, contribution := range firstContributions {
		result = append(result, contribution)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].date.Before(result[j].date)
	})

	p.updatePostContent(post, result, since)
	p.updatePost(post, userID)
	p.createContributorsPost(post.ChannelId, userID, result)
}

func (p *Plugin) findFirstContributions(contributors map[string][]*github.Contributor, org string, since time.Time) (map[string]firstContributionInfo, error) {
	firstContributions := map[string]firstContributionInfo{}
	earlierContributors := map[string]bool{}

	for repo, repoContributors := range contributors {
		for _, contributor := range repoContributors {
			author := contributor.GetLogin()

			_, contributedEarlier := earlierContributors[author]
			if contributedEarlier {
				continue
			}

			commits, err := p.fetchCommitsFromRepoByAuthor(org, repo, author)
			if err != nil {
				return nil, err
			}

			if len(commits) == 0 {
				continue
			}
			sort.Slice(commits, func(i, j int) bool {
				return commits[i].GetCommit().Committer.GetDate().Before(commits[j].GetCommit().Committer.GetDate())
			})
			firstCommitInRepo := commits[0]

			if firstCommitInRepo.GetCommit().Committer.GetDate().Before(since) {
				earlierContributors[author] = true
				delete(firstContributions, author)
				continue
			}

			firstContribution, contains := firstContributions[author]
			if !contains || firstContribution.date.After(firstCommitInRepo.GetCommit().Committer.GetDate()) {
				firstContributions[author] = firstContributionInfo{author,
					firstCommitInRepo.GetCommit().Committer.GetDate(),
					firstCommitInRepo.GetHTMLURL(),
					org,
					repo}
			}
		}
	}
	return firstContributions, nil
}

func (p *Plugin) logAndPropUserAboutError(post *model.Post, userID string, err error) {
	p.API.LogError("failed to fetch data", "err", err.Error())

	var message = "Failed to fetch data:" + err.Error()
	if _, ok := err.(*github.RateLimitError); ok {
		message = rateLimitMessage
	}
	post.Props["attachments"].([]*model.SlackAttachment)[0].Text = message
	p.updatePost(post, userID)
}

func (p *Plugin) updatePostContent(post *model.Post, result []firstContributionInfo, since time.Time) {
	attachment := post.Props["attachments"].([]*model.SlackAttachment)[0]
	attachment.Title = "New Committers since " + since.Format(shortFormWithDay)
	attachment.Text = ""
	attachment.Fields = []*model.SlackAttachmentField{{
		Title: "Number of new committers:",
		Value: strconv.Itoa(len(result)),
	}}
}

func (p *Plugin) updatePost(post *model.Post, userID string) {
	if _, appErr := p.API.UpdatePost(post); appErr != nil {
		p.SendEphemeralPost(post.ChannelId, userID, "Something went bad. Please try again. "+appErr.Where+" "+appErr.Id)
		p.API.LogError("failed to update post", "err", appErr.Error())
	}
}

func (p *Plugin) createContributorsPost(channelID, userID string, result []firstContributionInfo) {
	var resultText string
	for _, e := range result {
		resultText += fmt.Sprintf("- [%[1]s](https://github.com/%[1]v): [first commit](%s) at %s on [%[4]s](https://github.com/%s/%[4]v)\n", e.author, e.commit, e.date.Format(shortFormWithDay), e.repo, e.org)
	}

	committersPost := &model.Post{
		ChannelId: channelID,
		UserId:    p.botUserID,
		Message:   resultText,
	}

	if _, appErr := p.API.CreatePost(committersPost); appErr != nil {
		p.SendEphemeralPost(committersPost.ChannelId, userID, "Something went bad. Please try again. "+appErr.Where+" "+appErr.Id)
		p.API.LogError("failed to create post", "err", appErr.Error())
	}
}
