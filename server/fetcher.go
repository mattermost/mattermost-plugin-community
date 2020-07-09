package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-github/v25/github"
)

type commitsResult struct {
	commits []*github.RepositoryCommit
	err     error
}

const resultsPerPage = 100

func (p *Plugin) fetchCommitsFromOrg(org string, since, until time.Time) ([]*github.RepositoryCommit, error) {
	var result []*github.RepositoryCommit
	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{
			PerPage: resultsPerPage,
		},
		Type: "sources",
	}

	for {
		repos, resp, err := p.client.Repositories.ListByOrg(context.Background(), org, opts)
		if err != nil {
			return nil, err
		}

		var wg sync.WaitGroup
		var jobResults = make(chan commitsResult, len(repos))

		for _, repo := range repos {
			wg.Add(1)
			go p.fetchCommitsFromRepoJob(&wg, jobResults, org, *repo.Name, since, until)
		}
		go func() {
			wg.Wait()
			close(jobResults)
		}()

		for jr := range jobResults {
			if jr.err != nil {
				p.API.LogWarn("Failed to fetch commits ", "error", jr.err.Error())
			} else {
				result = append(result, jr.commits...)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return result, nil
}

func (p *Plugin) fetchCommitsFromUser(user string, since, until time.Time) ([]*github.RepositoryCommit, error) {
	var result []*github.RepositoryCommit
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{
			PerPage: resultsPerPage,
		},
		Type: "sources",
	}

	for {
		repos, resp, err := p.client.Repositories.List(context.Background(), user, opts)
		if err != nil {
			return nil, err
		}

		var wg sync.WaitGroup
		var jobResults = make(chan commitsResult, len(repos))

		for _, repo := range repos {
			wg.Add(1)
			go p.fetchCommitsFromRepoJob(&wg, jobResults, user, *repo.Name, since, until)
		}
		go func() {
			wg.Wait()
			close(jobResults)
		}()

		for jr := range jobResults {
			if jr.err != nil {
				p.API.LogWarn("Failed to fetch commits ", "error", jr.err.Error())
			} else {
				result = append(result, jr.commits...)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return result, nil
}

func (p *Plugin) fetchCommitsFromRepoJob(wg *sync.WaitGroup, result chan<- commitsResult, org, repo string, since, until time.Time) {
	commits, err := p.fetchCommitsFromRepo(org, repo, since, until)
	output := commitsResult{commits, err}
	result <- output
	wg.Done()
}

func (p *Plugin) fetchCommitsFromRepo(org, repo string, since, until time.Time) ([]*github.RepositoryCommit, error) {
	var result []*github.RepositoryCommit
	opts := &github.CommitsListOptions{
		ListOptions: github.ListOptions{
			PerPage: resultsPerPage,
		},
		Since: since,
		Until: until,
	}

	for {
		commits, resp, err := p.client.Repositories.ListCommits(context.Background(), org, repo, opts)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("repository %v/%v not found", org, repo)
		}
		if err != nil {
			return nil, err
		}
		result = append(result, commits...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return result, nil
}

func (p *Plugin) fetchTeamMemberFromTeam(id int64) ([]*github.User, error) {
	var result []*github.User
	opts := &github.TeamListTeamMembersOptions{
		ListOptions: github.ListOptions{
			PerPage: resultsPerPage,
		},
	}
	for {
		member, resp, err := p.client.Teams.ListTeamMembers(context.Background(), id, opts)
		if err != nil {
			return nil, err
		}
		result = append(result, member...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return result, nil
}

func (p *Plugin) fetchTeams(org string) ([]*github.Team, error) {
	var result []*github.Team
	opts := &github.ListOptions{
		PerPage: resultsPerPage,
	}
	for {
		teams, resp, err := p.client.Teams.ListTeams(context.Background(), org, opts)
		if err != nil {
			return nil, err
		}
		result = append(result, teams...)

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return result, nil
}
