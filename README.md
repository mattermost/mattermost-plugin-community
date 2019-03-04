# Community Plugin [![Build Status](https://travis-ci.org/mattermost/mattermost-plugin-community.svg?branch=master)](https://travis-ci.org/mattermost/mattermost-plugin-community)

This plugin allows users to fetch contributor data from GitHub via a slash command.

## Installation
1. Go to the [releases page of this GitHub repository](https://github.com/mattermost/mattermost-plugin-community/releases/latest) and download the latest release for your Mattermost server.
2. Upload this file in the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. Create a personal access token for your GitHub account [here](https://github.com/settings/tokens). This is required because GitHub has a low rate limit for unauthenticated API requests. You do not need to specify a scope for your token.

## Usage
Use `/community [feature] [organisation] [repo] [YYYY-MM]` to fetch data and summarize it in a post, e.g. `/community changelog mattermost mattermost-server 2019-02`. To fetch the data from all repositories in an organisation, replace the repository name with `all`, e.g. `/community changelog mattermost all 2019-02`.

## Screenshots
![Fetching data](images/fetching.png)
![Mattermost contributors](images/mattermost_all.png)
![Hugo contributors](images/gohugo_hugo.png)
