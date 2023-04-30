package gitlab

import (
	"encoding/json"
	"net/http"
	"strconv"

	"djinn-ci.com/errors"
	"djinn-ci.com/provider"
)

var ErrNoCommits = errors.New("gitlab: no commits")

func ParseWebhook(r *http.Request) (*provider.WebhookData, error) {
	var data provider.WebhookData

	actions := map[string]string{
		"merge":  "merged",
		"open":   "opened",
		"update": "updated",
	}

	switch r.Header.Get("X-Gitlab-Event") {
	case "Push Hook":
		var push pushEvent
		json.NewDecoder(r.Body).Decode(&push)

		if len(push.Commits) == 0 {
			return nil, ErrNoCommits
		}

		head := push.Commits[0]

		data.UserID = push.UserID
		data.RepoID = push.Project.ID
		data.Event = provider.PushEvent
		data.DirURL = "/projects/" + strconv.FormatInt(push.Project.ID, 10) + "/repository/files/.djinn"
		data.Ref = head.ID
		data.Comment = head.Message
		data.Data = map[string]string{
			"repo_id":  strconv.FormatInt(push.Project.ID, 10),
			"provider": "gitlab",
			"url":      head.URL,
			"ref":      push.Ref,
			"sha":      head.ID,
			"email":    head.Author["email"],
			"username": head.Author["name"],
		}
	case "Merge Request Hook":
		var merge mergeRequestEvent
		json.NewDecoder(r.Body).Decode(&merge)

		data.UserID = merge.User.ID
		data.RepoID = merge.Project.ID
		data.Event = provider.PullEvent
		data.DirURL = "/projects/" + strconv.FormatInt(merge.Attrs.SourceProjectID, 10) + "/repository/files/.djinn"
		data.Ref = merge.Attrs.LastCommit.ID
		data.Comment = merge.Attrs.Title
		data.Data = map[string]string{
			"repo_id":  strconv.FormatInt(merge.Attrs.TargetProjectID, 10),
			"provider": "gitlab",
			"id":       strconv.FormatInt(merge.Attrs.ID, 10),
			"url":      merge.Attrs.URL,
			"ref":      merge.Attrs.TargetBranch,
			"sha":      merge.Attrs.LastCommit.ID,
			"username": merge.User.Username,
			"action":   actions[merge.Attrs.Action],
		}
	default:
		return nil, provider.ErrUnknownEvent
	}
	return &data, nil
}
