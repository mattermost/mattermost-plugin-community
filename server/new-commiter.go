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

type contributionInfo struct {
	contributionWeek time.Time
	repo             string
}

type firstContributionInfo struct {
	author string
	date   time.Time
	commit string
}

func (p *Plugin) executeNewCommiterCommand(commandArgs []string, args *model.CommandArgs) *model.AppError {
	if len(commandArgs) != 2 {
		return &model.AppError{
			Id:         "Need two arguments",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	organisation := commandArgs[0]

	since, err := time.Parse(shortFormWithDay, commandArgs[1])
	if err != nil {
		return &model.AppError{
			Id:         fmt.Sprintf("Failed to parse since time: %v", err.Error()),
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	org, _, err := p.client.Organizations.Get(context.Background(), organisation)
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
		AuthorName: organisation,
		AuthorIcon: org.GetAvatarURL(),
		AuthorLink: fmt.Sprintf("https://github.com/%v", organisation),
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

	go p.updateNewCommitersPost(loadingPost, args.UserId, organisation, since)

	return nil
}

func (p *Plugin) updateNewCommitersPost(post *model.Post, userID, org string, since time.Time) {
	contributors, err := p.fetchContributorsFromOrg(org)
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
		firstContributions := p.findFirstContributions(contributors)
		p.filterContributionsByWeek(firstContributions, since)

		var result []*firstContributionInfo
		for commiter, contribution := range firstContributions {
			commits, _ := p.fetchCommitsFromRepoByAuthorAndWeek(org, contribution.repo, commiter, contribution.contributionWeek)
			firstCommit := commits[0]
			if firstCommit.GetCommit().Committer.GetDate().After(since) {
				result = append(result, &firstContributionInfo{firstCommit.GetAuthor().GetLogin(), firstCommit.GetCommit().Committer.GetDate(), firstCommit.GetHTMLURL()})
			}
		}

		sort.Slice(result, func(i, j int) bool {
			return result[i].date.Before(result[j].date)
		})

		var resultText string
		for _, e := range result {
			resultText += fmt.Sprintf("- [%[1]s](https://github.com/%[1]v)  [first commit](%s)  %s\n", e.author, e.commit, e.date.Format(shortFormWithDay))
		}

		attachment := post.Props["attachments"].([]*model.SlackAttachment)[0]
		attachment.Title = "New Commiters since " + since.Format(shortFormWithDay)
		attachment.Text = ""
		attachment.Fields = []*model.SlackAttachmentField{{
			Title: "Number of new commiters",
			Value: strconv.Itoa(len(result)),
		}, {
			Title: "Commiters and their first commits",
			Value: resultText,
		}}
	}

	if _, appErr := p.API.UpdatePost(post); appErr != nil {
		p.SendEphemeralPost(post.ChannelId, userID, "Something went bad. Please try again. "+appErr.Where+" "+appErr.Id)
		p.API.LogError("failed to update post", "err", appErr.Error())
		return
	}
}

func (p *Plugin) findFirstContributions(contributors []*repoContributors) map[string]contributionInfo {
	firstContributions := map[string]contributionInfo{}
	for _, repoContributors := range contributors {
		for _, contributor := range repoContributors.contributorStats {
			author := contributor.GetAuthor()
			if author == nil {
				continue
			}
			commiter := author.GetLogin()
			for _, week := range contributor.Weeks {
				commitsThatWeek := week.GetCommits()
				if commitsThatWeek != 0 {
					weekStartDate := week.GetWeek().Time
					if firstContributions[commiter].contributionWeek.IsZero() || firstContributions[commiter].contributionWeek.After(weekStartDate) {
						firstContributions[commiter] = contributionInfo{weekStartDate, repoContributors.repo}
					}
					break
				}
			}
		}
	}
	return firstContributions
}

func (p *Plugin) filterContributionsByWeek(contributions map[string]contributionInfo, since time.Time) {
	sinceWeek := since.AddDate(0, 0, -int(since.Weekday()))
	for commiter, contribution := range contributions {
		if contribution.contributionWeek.Before(sinceWeek) {
			delete(contributions, commiter)
		}
	}
}
