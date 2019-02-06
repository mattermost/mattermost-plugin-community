package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/google/go-github/github"
)

type contributorsDataResult struct {
	contributors    sort.StringSlice
	numberOfCommits uint64
	err             error
}

const resultsPerPage = 100

func (p *Plugin) fetchContributorsDataFromOrg(org string, since, until time.Time) (sort.StringSlice, uint64, error) {
	var contributors sort.StringSlice
	var numberOfCommits uint64
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{
			PerPage: resultsPerPage,
		},
		Type: "sources",
	}

	for {
		repos, resp, err := p.client.Repositories.ListByOrg(context.Background(), org, opt)
		if err != nil {
			return nil, 0, err
		}

		var wg sync.WaitGroup
		var results = make(chan contributorsDataResult, len(repos))

		for _, repo := range repos {
			wg.Add(1)
			go p.runfetchContributorsDataFromRepoJob(&wg, results, org, *repo.Name, since, until)
		}
		go func() {
			wg.Wait()
			close(results)
		}()

		for result := range results {
			if result.err != nil {
				return nil, 0, err
			}
			numberOfCommits += result.numberOfCommits
			contributors = union(contributors, result.contributors)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return contributors, numberOfCommits, nil
}

func (p *Plugin) runfetchContributorsDataFromRepoJob(wg *sync.WaitGroup, result chan<- contributorsDataResult, org, repo string, since, until time.Time) {
	c, n, err := p.fetchContributorsDataFromRepo(org, repo, since, until)
	output := contributorsDataResult{c, n, err}
	result <- output
	wg.Done()
}

func (p *Plugin) fetchContributorsDataFromRepo(org, repo string, since, until time.Time) (sort.StringSlice, uint64, error) {
	var contributors sort.StringSlice
	var numberOfCommits uint64

	// Fetch commits until one day after at midnight
	until = until.AddDate(0, 0, 1)
	opt := &github.CommitsListOptions{
		ListOptions: github.ListOptions{
			PerPage: resultsPerPage,
		},
		Since: since,
		Until: until,
	}

	for {
		commits, resp, err := p.client.Repositories.ListCommits(context.Background(), org, repo, opt)
		if resp.StatusCode == http.StatusNotFound {
			return nil, 0, fmt.Errorf("repository %v/%v not found", org, repo)
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
