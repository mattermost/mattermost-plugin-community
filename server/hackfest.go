package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v25/github"
	"github.com/mattermost/mattermost-server/v5/model"
)

func (p *Plugin) executeHackfestCommand(commandArgs []string, args *model.CommandArgs) *model.AppError {
	if len(commandArgs) != 1 {
		return &model.AppError{
			Id:         "Need one arguments",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}
	command := commandArgs[0]

	var appErr *model.AppError
	switch command {
	case "info":
		appErr = p.postHackfestInfo(args)
	case "list":
		appErr = p.listHackfestContributors(args)
	default:
		return &model.AppError{
			Id:         fmt.Sprintf("Unknown command %v", command),
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}
	return appErr
}

func (p *Plugin) postHackfestInfo(args *model.CommandArgs) *model.AppError {
	config := p.getConfiguration()

	start, err := time.Parse(shortFormWithDay, config.HackfestStart)
	if err != nil {
		return &model.AppError{
			Id:         "Hackfest start date not proper configured. Please contact you system administrator",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	end, err := time.Parse(shortFormWithDay, config.HackfestEnd)
	if err != nil {
		return &model.AppError{
			Id:         "Hackfest end date not proper configured. Please contact you system administrator",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	now := time.Now()

	var text string
	if now.After(start) && now.Before(end.AddDate(0, 0, 1).Add(-time.Microsecond)) {
		text = fmt.Sprintf("There is a hackfest running from %v to %v", start.Format(shortFormWithDay), end.Format(shortFormWithDay))
	} else {
		text = "No hackfest is running"
	}

	attachments := []*model.SlackAttachment{{
		Title: "Hackfest info",
		Text:  text,
	}}

	post := &model.Post{
		ChannelId: args.ChannelId,
		UserId:    p.botUserID,
	}
	model.ParseSlackAttachment(post, attachments)

	if _, appErr := p.API.CreatePost(post); appErr != nil {
		return appErr
	}
	return nil
}

func (p *Plugin) listHackfestContributors(args *model.CommandArgs) *model.AppError {
	config := p.getConfiguration()

	start, err := time.Parse(shortFormWithDay, config.HackfestStart)
	if err != nil {
		return &model.AppError{
			Id:         "Hackfest start date not proper configured. Please contact you system administrator",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}

	end, err := time.Parse(shortFormWithDay, config.HackfestEnd)
	if err != nil {
		return &model.AppError{
			Id:         "Hackfest end date not proper configured. Please contact you system administrator",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}
	org := config.HackfestOrg
	if org == "" {
		return &model.AppError{
			Id:         "Hackfest organization not configured. Please contact you system administrator",
			StatusCode: http.StatusBadRequest,
			Where:      "p.ExecuteCommand",
		}
	}
	repo := config.HackfestRepo

	topic := org
	if repo != "" {
		topic += "/" + repo
	}

	attachments := []*model.SlackAttachment{{
		Title:      "Fetching Hackfest contributors",
		Text:       waitText,
		AuthorName: topic,
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

	go p.updateHackfestContributorsPost(loadingPost, args.UserId, org, repo, start, end)
	return nil
}

func (p *Plugin) updateHackfestContributorsPost(post *model.Post, userID, org, repo string, since, until time.Time) {
	config := p.getConfiguration()

	// Fetch commits until one day after at midnight
	fetchUntil := until.AddDate(0, 0, 1).Add(-time.Microsecond)

	var commits []*github.RepositoryCommit
	var err error
	if repo != "" {
		commits, err = p.fetchCommitsFromRepo(org, repo, since, fetchUntil)
	} else {
		commits, err = p.fetchCommitsFromOrg(org, since, fetchUntil)
	}

	excludedUsers := strings.Split(config.HackfestExcludeUsers, ", ")

	excludedTeams := strings.Split(config.HackfestExcludeTeams, ", ")
	if len(excludedTeams) > 0 {
		var teams []*github.Team
		teams, err = p.fetchTeams(org)
		if err != nil {
			p.API.LogWarn("failed to fetch teams", "error", err.Error())
			return
		}

		for _, excludedTeam := range excludedTeams {
			for _, team := range teams {
				if team.GetSlug() == excludedTeam {
					var member []*github.User
					member, err = p.fetchTeamMemberFromTeam(team.GetID())
					if err != nil {
						p.API.LogWarn("failed to fetch team member", "error", err.Error())
						return
					}

					for _, m := range member {
						excludedUsers = append(excludedUsers, m.GetLogin())
					}
				}
			}
		}
	}

	if err != nil {
		p.API.LogWarn("failed to fetch data", "err", err.Error())

		message := githubErrorHandle(err)
		post.Props["attachments"].([]*model.SlackAttachment)[0].Text = message
	} else {
		contributors := map[string]int{}
		for _, c := range commits {
			author := c.GetAuthor()
			if author == nil {
				continue
			}

			userName := author.GetLogin()
			isExcluded := false
			for _, m := range excludedUsers {
				if userName == m {
					isExcluded = true
				}
			}
			if isExcluded {
				continue
			}

			contributors[userName]++
		}

		type kv struct {
			Key   string
			Value int
		}

		var ss []kv
		for k, v := range contributors {
			ss = append(ss, kv{k, v})
		}

		sort.Slice(ss, func(i, j int) bool {
			return ss[i].Value > ss[j].Value
		})

		var contributorsText string
		for _, e := range ss {
			var c string
			if e.Value > 1 {
				c = "contributions"
			} else {
				c = "contribution"
			}
			contributorsText += fmt.Sprintf("- [%[1]s](https://github.com/%[1]v): %v %v\n", e.Key, e.Value, c)
		}

		attachment := post.Props["attachments"].([]*model.SlackAttachment)[0]
		attachment.Title = "Hackfest stats"
		attachment.Text = ""
		attachment.Fields = []*model.SlackAttachmentField{{
			Title: "Number of Contributors",
			Value: strconv.Itoa(len(contributors)),
		}, {
			Title: "Contributors",
			Value: contributorsText,
		}}
	}

	if _, appErr := p.API.UpdatePost(post); appErr != nil {
		p.SendEphemeralPost(post.ChannelId, userID, "Something went bad. Please try again.")
		p.API.LogWarn("failed to update post", "err", appErr.Error())
		return
	}
}
