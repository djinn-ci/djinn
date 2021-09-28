package djinn

import (
	"net/http"
)

type Invite struct {
	ID          int64      `json:"id"`
	NamespaceID int64      `json:"namespace_id"`
	InviteeID   int64      `json:"invitee_id"`
	InviterID   int64      `json:"inviter_id"`
	URL         URL        `json:"url"`
	Invitee     *User      `json:"invitee"`
	Inviter     *User      `json:"inviter"`
	Namespace   *Namespace `json:"namespace"`
}

func (i *Invite) Accept(cli *Client) error {
	resp, err := cli.Patch(i.URL.Path, "application/json; charset=utf-8", nil)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return cli.err(resp)
	}
	return nil
}

func (i *Invite) Reject(cli *Client) error {
	resp, err := cli.Delete(i.URL.Path)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		return cli.err(resp)
	}
	return nil
}
