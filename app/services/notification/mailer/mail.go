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

package mailer

import (
	"context"
	"crypto/tls"

	gomail "gopkg.in/mail.v2"
)

type GoMailClient struct {
	dialer   *gomail.Dialer
	fromMail string
}

func NewMailClient(
	host string,
	port int,
	username string,
	fromMail string,
	password string,
	insecure bool,
) GoMailClient {
	d := gomail.NewDialer(host, port, username, password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: insecure} // #nosec G402 (insecure TLS configuration)

	return GoMailClient{
		dialer:   d,
		fromMail: fromMail,
	}
}

func (c GoMailClient) Send(_ context.Context, mailPayload Payload) error {
	mail := ToGoMail(mailPayload)
	mail.SetHeader("From", c.fromMail)
	return c.dialer.DialAndSend(mail)
}
