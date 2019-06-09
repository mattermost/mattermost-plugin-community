module github.com/mattermost/mattermost-plugin-community

go 1.12

require (
	github.com/google/go-github/v25 v25.1.1
	github.com/mattermost/mattermost-server v0.0.0-20190516103213-2d3e41783398
	github.com/pkg/errors v0.8.1
	github.com/stretchr/testify v1.3.0
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45
)

// Workaround for https://github.com/golang/go/issues/30831 and fallout.
replace github.com/golang/lint => github.com/golang/lint v0.0.0-20190227174305-8f45f776aaf1
