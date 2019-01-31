# Community Plugin [![Build Status](https://travis-ci.org/mattermost/mattermost-plugin-community.svg?branch=master)](https://travis-ci.org/mattermost/mattermost-plugin-community)

This plugin allows users to fetch contributors data from GitHub via a slash command.

## Installation
1. Go to the [releases page of this GitHub repository](https://github.com/mattermost/mattermost-plugin-community/releases/latest) and download the latest release for your Mattermost server.
2. Upload this file in the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. Create a personal access token for your GitHub Account [here](https://github.com/settings/tokens). It doesn't need any scope. This is needed, because GitHub has a low rate limit for unauthenticated API requests.

## Usage
Use `/contributors [organisation] [repo] [since] [until]` to fetch the data and report it in a post, e.g. `/contributors mattermost mattermost-server 2019-01-01 2019-01-31`. To fetch the data from all repositories in an organisation replace the repository name with `all`, e.g. `/contributors mattermost all 2019-01-01 2019-01-31`.

## Screenshots
![Fetching data](images/fetching.png)
![Mattermost contributors](images/mattermost_all.png)
![Hugo contributors](images/gohugo_hugo.png)