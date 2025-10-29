// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package notification

import (
	"bytes"
	"context"
	"fmt"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/notification/mailer"
	"github.com/harness/gitness/types"
)

const (
	TemplateReviewerAdded        = "reviewer_added.html"
	TemplateCommentPRAuthor      = "comment_pr_author.html"
	TemplateCommentMentions      = "comment_mentions.html"
	TemplateCommentParticipants  = "comment_participants.html"
	TemplatePullReqBranchUpdated = "pullreq_branch_updated.html"
	TemplateNameReviewSubmitted  = "review_submitted.html"
	TemplatePullReqStateChanged  = "pullreq_state_changed.html"
)

type MailClient struct {
	mailer.Mailer
}

func NewMailClient(mailer mailer.Mailer) MailClient {
	return MailClient{
		Mailer: mailer,
	}
}

func (m MailClient) SendCommentPRAuthor(
	ctx context.Context,
	recipients []*types.PrincipalInfo,
	payload *CommentPayload,
) error {
	email, err := GenerateEmailFromPayload(
		TemplateCommentPRAuthor,
		recipients,
		payload.Base,
		payload,
	)
	if err != nil {
		return fmt.Errorf("failed to generate mail requests after processing %s event: %w",
			pullreqevents.CommentCreatedEvent, err)
	}

	return m.Mailer.Send(ctx, *email)
}
func (m MailClient) SendCommentMentions(
	ctx context.Context,
	recipients []*types.PrincipalInfo,
	payload *CommentPayload,
) error {
	email, err := GenerateEmailFromPayload(
		TemplateCommentMentions,
		recipients,
		payload.Base,
		payload,
	)
	if err != nil {
		return fmt.Errorf("failed to generate mail requests after processing %s event: %w",
			pullreqevents.CommentCreatedEvent, err)
	}

	return m.Mailer.Send(ctx, *email)
}
func (m MailClient) SendCommentParticipants(
	ctx context.Context,
	recipients []*types.PrincipalInfo,
	payload *CommentPayload,
) error {
	email, err := GenerateEmailFromPayload(
		TemplateCommentParticipants,
		recipients,
		payload.Base,
		payload,
	)
	if err != nil {
		return fmt.Errorf("failed to generate mail requests after processing %s event: %w",
			pullreqevents.CommentCreatedEvent, err)
	}

	return m.Mailer.Send(ctx, *email)
}

func (m MailClient) SendReviewerAdded(
	ctx context.Context,
	recipients []*types.PrincipalInfo,
	payload *ReviewerAddedPayload,
) error {
	email, err := GenerateEmailFromPayload(
		TemplateReviewerAdded,
		recipients,
		payload.Base,
		payload,
	)
	if err != nil {
		return fmt.Errorf("failed to generate mail requests after processing %s event: %w",
			pullreqevents.ReviewerAddedEvent, err)
	}

	return m.Mailer.Send(ctx, *email)
}

func (m MailClient) SendPullReqBranchUpdated(
	ctx context.Context,
	recipients []*types.PrincipalInfo,
	payload *PullReqBranchUpdatedPayload,
) error {
	email, err := GenerateEmailFromPayload(
		TemplatePullReqBranchUpdated,
		recipients,
		payload.Base,
		payload,
	)
	if err != nil {
		return fmt.Errorf("failed to generate mail requests after processing %s event: %w",
			pullreqevents.BranchUpdatedEvent, err)
	}

	return m.Mailer.Send(ctx, *email)
}

func (m MailClient) SendReviewSubmitted(
	ctx context.Context,
	recipients []*types.PrincipalInfo,
	payload *ReviewSubmittedPayload,
) error {
	email, err := GenerateEmailFromPayload(TemplateNameReviewSubmitted, recipients, payload.Base, payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate mail requests after processing %s event: %w",
			pullreqevents.ReviewSubmittedEvent,
			err,
		)
	}
	return m.Mailer.Send(ctx, *email)
}

func (m MailClient) SendPullReqStateChanged(
	ctx context.Context,
	recipients []*types.PrincipalInfo,
	payload *PullReqStateChangedPayload,
) error {
	email, err := GenerateEmailFromPayload(TemplatePullReqStateChanged, recipients, payload.Base, payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate mail requests after processing pullReqState change event: %w",
			err,
		)
	}

	return m.Mailer.Send(ctx, *email)
}

func GetSubjectPullRequest(
	repoIdentifier string,
	prNum int64,
	prTitle string,
) string {
	return fmt.Sprintf(subjectPullReqEvent, repoIdentifier, prTitle, prNum)
}

func GetHTMLBody(templateName string, data any) ([]byte, error) {
	tmpl := htmlTemplates[templateName]
	tmplOutput := bytes.Buffer{}
	err := tmpl.Execute(&tmplOutput, data)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template %s", templateName)
	}

	return tmplOutput.Bytes(), nil
}

func GenerateEmailFromPayload(
	templateName string,
	recipients []*types.PrincipalInfo,
	base *BasePullReqPayload,
	payload any,
) (*mailer.Payload, error) {
	subject := GetSubjectPullRequest(base.Repo.Identifier, base.PullReq.Number,
		base.PullReq.Title)

	body, err := GetHTMLBody(templateName, payload)
	if err != nil {
		return nil, err
	}

	var email mailer.Payload
	email.Body = string(body)
	email.Subject = subject
	email.RepoRef = base.Repo.Path

	recipientEmails := RetrieveEmailsFromPrincipals(recipients)
	email.ToRecipients = recipientEmails
	return &email, nil
}

func RetrieveEmailsFromPrincipals(principals []*types.PrincipalInfo) []string {
	emails := make([]string, len(principals))
	for i, principal := range principals {
		emails[i] = principal.Email
	}
	return emails
}
