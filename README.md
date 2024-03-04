# Community Plugin ![CircleCI branch](https://img.shields.io/circleci/project/github/mattermost/mattermost-plugin-community/master.svg)

This plugin allows users to fetch contributor data from GitHub via a slash command.

## Installation
1. Go to the [releases page of this GitHub repository](https://github.com/mattermost/mattermost-plugin-community/releases/latest) and download the latest release for your Mattermost server.
2. Upload this file in the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. Install the GitHub plugin and connect your GitHub account. **Note:** The Community Plugin only works with the ``master`` branch version of the GitHub plugin.
4. Create a personal access token for your GitHub account [here](https://github.com/settings/tokens). This is required because GitHub has a low rate limit for unauthenticated API requests. You do not need to specify a scope for your token.

## Usage
Use `/community changelog mattermost [year-month]` to fetch data and summarize it in a post, e.g. `/community changelog mattermost 2024-01`.

## Screenshots
![Fetching data](images/fetching.png)
![Mattermost contributors](images/mattermost_all.png)
![Hugo contributors](images/gohugo_hugo.png)
