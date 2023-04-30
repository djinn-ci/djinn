package github

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"djinn-ci.com/errors"
	"djinn-ci.com/provider"
)

var ErrInvalidAction = errors.New("github: invalid action")

func ParseWebhook(r *http.Request) (*provider.WebhookData, error) {
	var data provider.WebhookData

	actions := map[string]struct{}{
		"opened":      {},
		"reopened":    {},
		"unlocked":    {},
		"synchronize": {},
	}

	switch r.Header.Get("X-Github-Event") {
	case "push":
		var push pushEvent
		json.NewDecoder(r.Body).Decode(&push)

		data.UserID = push.Repo.Owner.ID
		data.RepoID = push.Repo.ID
		data.Event = provider.PushEvent
		data.DirURL = strings.Replace(push.Repo.ContentsURL, "{+path}", ".djinn", 1)
		data.Ref = push.HeadCommit.ID
		data.Comment = push.HeadCommit.Message
		data.Data = map[string]string{
			"repo_id":  strconv.FormatInt(push.Repo.ID, 10),
			"provider": "github",
			"url":      push.HeadCommit.URL,
			"ref":      push.Ref,
			"sha":      push.HeadCommit.ID,
			"email":    push.HeadCommit.Author["email"],
			"username": push.HeadCommit.Author["username"],
		}
	case "pull_event":
		var pull pullRequestEvent
		json.NewDecoder(r.Body).Decode(&pull)

		if _, ok := actions[pull.Action]; !ok {
			return nil, ErrInvalidAction
		}

		data.UserID = pull.PullRequest.Base.User.ID
		data.RepoID = pull.PullRequest.Base.Repo.ID
		data.Event = provider.PullEvent
		data.DirURL = strings.Replace(pull.PullRequest.Head.Repo.ContentsURL, "{+path}", ".djinn", 1)
		data.Ref = pull.PullRequest.Head.Sha
		data.Comment = pull.PullRequest.Title

		action := pull.Action

		if action == "synchronize" {
			action = "synchronized"
		}

		data.Data = map[string]string{
			"repo_id":  strconv.FormatInt(pull.PullRequest.Base.Repo.ID, 10),
			"provider": "github",
			"id":       strconv.FormatInt(pull.Number, 10),
			"url":      pull.PullRequest.HTMLURL,
			"ref":      pull.PullRequest.Base.Ref,
			"sha":      pull.PullRequest.Head.Sha,
			"username": pull.PullRequest.User.Login,
			"action":   action,
		}
	default:
		return nil, provider.ErrUnknownEvent
	}
	return &data, nil
}
