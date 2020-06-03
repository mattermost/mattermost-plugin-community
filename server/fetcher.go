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

type repoContributors struct {
	contributorStats []*github.ContributorStats
	repo             string
}

type contributorsResult struct {
	contributors *repoContributors
	err          error
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
				return nil, err
			}
			result = append(result, jr.commits...)
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

func (p *Plugin) fetchContributorsFromOrg(org string) ([]*repoContributors, error) {
	var result []*repoContributors
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
		var jobResults = make(chan contributorsResult, len(repos))

		for _, repo := range repos {
			wg.Add(1)
			go p.fetchContributorsFromRepoJob(&wg, jobResults, org, *repo.Name)
		}
		go func() {
			wg.Wait()
			close(jobResults)
		}()

		for jr := range jobResults {
			if jr.err == nil {
				result = append(result, jr.contributors)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return result, nil
}

func (p *Plugin) fetchContributorsFromRepoJob(wg *sync.WaitGroup, result chan<- contributorsResult, org, repo string) {
	contributors, err := p.fetchContributorsFromRepo(org, repo)
	output := contributorsResult{&repoContributors{contributors, repo}, err}
	result <- output
	wg.Done()
}

func (p *Plugin) fetchContributorsFromRepo(org, repo string) ([]*github.ContributorStats, error) {
	var result []*github.ContributorStats

	for {
		contributors, resp, err := p.client.Repositories.ListContributorsStats(context.Background(), org, repo)
		if resp.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("repository %v/%v not found", org, repo)
		}
		if err != nil {
			return nil, err
		}
		result = append(result, contributors...)

		if resp.NextPage == 0 {
			break
		}
	}

	return result, nil
}

func (p *Plugin) fetchCommitsFromRepoByAuthorAndWeek(org, repo, author string, weekStart time.Time) ([]*github.RepositoryCommit, error) {
	var result []*github.RepositoryCommit
	opts := &github.CommitsListOptions{
		ListOptions: github.ListOptions{
			PerPage: resultsPerPage,
		},
		Since:  weekStart,
		Until:  weekStart.AddDate(0, 0, 7),
		Author: author,
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

func (p *Plugin) fetchOrgMember(org string) ([]*github.User, error) {
	teams, err := p.fetchTeams(org)
	if err != nil {
		return nil, err
	}

	var orgMember []*github.User
	for _, team := range teams {
		teamMember, err := p.fetchTeamMemberFromTeam(team.GetID())
		if err != nil {
			return nil, err
		}
		orgMember = append(orgMember, teamMember...)
	}
	return orgMember, nil
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
