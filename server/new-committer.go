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
	org    string
	repo   string
}

func (p *Plugin) executeNewCommitterCommand(commandArgs []string, args *model.CommandArgs) *model.AppError {
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

	go p.updateNewCommittersPost(loadingPost, args.UserId, organisation, since)

	return nil
}

func (p *Plugin) updateNewCommittersPost(post *model.Post, userID, org string, since time.Time) {
	contributors, err := p.fetchContributors(org)
	if err != nil {
		p.API.LogError("failed to fetch data", "err", err.Error())

		var message = "Failed to fetch data:" + err.Error()
		if _, ok := err.(*github.RateLimitError); ok {
			message = "Hit rate limit. Please try again later."
		}
		post.Props["attachments"].([]*model.SlackAttachment)[0].Text = message
		p.updatePost(post, userID)
		return
	}

	firstContributions := p.findFirstContributions(contributors)
	p.filterContributionsByWeek(firstContributions, since)

	var result []*firstContributionInfo
	for committer, contribution := range firstContributions {
		firstCommit := p.getFirstCommit(org, contribution.repo, committer, contribution.contributionWeek)
		if firstCommit != nil && firstCommit.GetCommit().Committer.GetDate().After(since) {
			result = append(result, &firstContributionInfo{firstCommit.GetAuthor().GetLogin(), firstCommit.GetCommit().Committer.GetDate(), firstCommit.GetHTMLURL(), org, contribution.repo})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].date.Before(result[j].date)
	})

	p.createPostContent(post, result, since)
	p.updatePost(post, userID)
}

func (p *Plugin) createPostContent(post *model.Post, result []*firstContributionInfo, since time.Time) {
	var resultText string
	for _, e := range result {
		resultText += fmt.Sprintf("- [%[1]s](https://github.com/%[1]v): [first commit](%s) at %s on [%[4]s](https://github.com/%s/%[4]v)\n", e.author, e.commit, e.date.Format(shortFormWithDay), e.repo, e.org)
	}

	attachment := post.Props["attachments"].([]*model.SlackAttachment)[0]
	attachment.Title = "New Committers since " + since.Format(shortFormWithDay)
	attachment.Text = ""
	attachment.Fields = []*model.SlackAttachmentField{{
		Title: "Number of new committers",
		Value: strconv.Itoa(len(result)),
	}, {
		Title: "Committer",
		Value: resultText,
	}}
}

func (p *Plugin) updatePost(post *model.Post, userID string) {
	if _, appErr := p.API.UpdatePost(post); appErr != nil {
		p.SendEphemeralPost(post.ChannelId, userID, "Something went bad. Please try again. "+appErr.Where+" "+appErr.Id)
		p.API.LogError("failed to update post", "err", appErr.Error())
	}
}

func (p *Plugin) getFirstCommit(org, repo, committer string, week time.Time) *github.RepositoryCommit {
	commits, err := p.fetchCommitsFromRepoByAuthorAndWeek(org, repo, committer, week)
	if err != nil || len(commits) == 0 {
		return nil
	}
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].GetCommit().Committer.GetDate().Before(commits[j].GetCommit().Committer.GetDate())
	})
	return commits[0]
}

func (p *Plugin) findFirstContributions(contributors map[string][]*github.ContributorStats) map[string]contributionInfo {
	firstContributions := map[string]contributionInfo{}
	for repo, repoContributors := range contributors {
		for _, contributor := range repoContributors {
			author := contributor.GetAuthor()
			if author == nil {
				continue
			}
			committer := firstContributions[author.GetLogin()]
			for _, week := range contributor.Weeks {
				if week.GetCommits() != 0 {
					weekStartDate := week.GetWeek().Time
					if committer.contributionWeek.IsZero() || committer.contributionWeek.After(weekStartDate) {
						firstContributions[author.GetLogin()] = contributionInfo{weekStartDate, repo}
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
	for committer, contribution := range contributions {
		if contribution.contributionWeek.Before(sinceWeek) {
			delete(contributions, committer)
		}
	}
}
