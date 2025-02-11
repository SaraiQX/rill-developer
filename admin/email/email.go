package email

import (
	"fmt"
)

type Client struct {
	sender      Sender
	frontendURL string
}

func New(sender Sender, frontendURL string) *Client {
	return &Client{
		sender:      sender,
		frontendURL: frontendURL,
	}
}

func (c *Client) SendOrganizationInvite(toEmail, toName, orgName, roleName string) error {
	err := c.sender.Send(
		toEmail,
		toName,
		"Invitation to join Rill",
		fmt.Sprintf("You have been invited to organization <b>%s</b> as <b>%s</b>. Please sign into Rill <a href=\"%s\">here</a> to accept invitation.", orgName, roleName, c.frontendURL),
	)
	return err
}

func (c *Client) SendProjectInvite(toEmail, toName, projectName, roleName string) error {
	err := c.sender.Send(
		toEmail,
		toName,
		"Invitation to join Rill",
		fmt.Sprintf("You have been invited to project <b>%s</b> as <b>%s</b>. Please sign into Rill <a href=\"%s\">here</a> to accept invitation.", projectName, roleName, c.frontendURL),
	)
	return err
}
