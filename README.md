# Community Plugin ![CircleCI branch](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-community/master.svg)

This plugin allows users to fetch contributor data from GitHub via a slash command.

## Installation
1. Download the ``master`` version of the Community plugin for your Mattermost server.
2. Upload this file in the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. Install the GitHub plugin and connect your GitHub account. The v2.0 of the Github plugin (or newer) works.
4. Create a personal access token for your GitHub account [here](https://github.com/settings/tokens). This is required because GitHub has a low rate limit for unauthenticated API requests. You do not need to specify a scope for your token.

## Usage
 - Use `/community committer [organization]/[repo] [since] [until]` to fetch data and summarize it in a post, e.g. `/community committer mattermost/mattermost-server 2019-01-01 2019-01-31`. To fetch the data from all repositories in an organization omit the repo name, e.g. `/community committer mattermost 2019-01-01 2019-01-31`.
 - Use `/community changelog mattermost [year-month]` to fetch data for monthly changelogs and summarize it in a post, e.g. `/community changelog mattermost 2024-01`.

## Screenshots
![Fetching data](images/fetching.png)
![Mattermost contributors](images/mattermost_all.png)
![Hugo contributors](images/gohugo_hugo.png)

## How to Release

To trigger a release, follow these steps:

1. **For Patch Release:** Run the following command:
    ```
    make patch
    ```
   This will release a patch change.

2. **For Minor Release:** Run the following command:
    ```
    make minor
    ```
   This will release a minor change.

3. **For Major Release:** Run the following command:
    ```
    make major
    ```
   This will release a major change.

4. **For Patch Release Candidate (RC):** Run the following command:
    ```
    make patch-rc
    ```
   This will release a patch release candidate.

5. **For Minor Release Candidate (RC):** Run the following command:
    ```
    make minor-rc
    ```
   This will release a minor release candidate.

6. **For Major Release Candidate (RC):** Run the following command:
    ```
    make major-rc
    ```
   This will release a major release candidate.

