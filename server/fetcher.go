package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/google/go-github/github"
)

func (p *Plugin) fetchAllRepos(org string, since, until time.Time) (sort.StringSlice, uint64, error) {
	var contributors sort.StringSlice
	var numberOfCommits uint64
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{
			PerPage: 10,
		},
		Type: "sources",
	}

	for {
		repos, resp, err := p.client.Repositories.ListByOrg(context.Background(), org, opt)
		if err != nil {
			return nil, 0, err
		}

		for _, repo := range repos {
			rName := *repo.Name
			repoContributors, n, err := p.fetchRepo(org, rName, since, until)
			if err != nil {
				return nil, 0, err
			}
			numberOfCommits += n
			contributors = union(contributors, repoContributors)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return contributors, numberOfCommits, nil
}

func (p *Plugin) fetchRepo(org, repo string, since, until time.Time) (sort.StringSlice, uint64, error) {
	var contributors sort.StringSlice
	var numberOfCommits uint64
	opt := &github.CommitsListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
		Since: since,
		Until: until,
	}

	for {
		commits, resp, err := p.client.Repositories.ListCommits(context.Background(), org, repo, opt)
		if resp.StatusCode == http.StatusNotFound {
			return nil, 0, fmt.Errorf("Repository %v/%v not found", org, repo)
		}
		if err != nil {
			return nil, 0, err
		}

		numberOfCommits += uint64(len(commits))

		for _, c := range commits {
			author := c.GetAuthor()
			if author == nil {
				continue
			}
			u := author.GetLogin()
			if !contains(contributors, u) {
				contributors = append(contributors, u)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return contributors, numberOfCommits, nil
}
