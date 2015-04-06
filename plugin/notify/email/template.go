package email

import (
	"html/template"
)

// raw email message template
var rawMessage = `From: %s
To: %s
Subject: %s
MIME-version: 1.0
Content-Type: text/html; charset="UTF-8"

%s`

// default success email template
var successTemplate = template.Must(template.New("_").Parse(`
<p>
	<b>Build was Successful</b>
	(<a href="{{.Host}}/{{.Repo.Host}}/{{.Repo.Owner}}/{{.Repo.Name}}/{{.Commit.Branch}}/{{.Commit.Sha}}">see results</a>)
</p>
<p>Repository : {{.Repo.Owner}}/{{.Repo.Name}}</p>
<p>Commit     : {{.Commit.ShaShort}}</p>
<p>Author     : {{.Commit.Author}}</p>
<p>Branch     : {{.Commit.Branch}}</p>
<p>Message:</p>
<p>{{ .Commit.Message }}</p>
`))

// default failure email template
var failureTemplate = template.Must(template.New("_").Parse(`
<p>
	<b>Build Failed</b>
	(<a href="{{.Host}}/{{.Repo.Host}}/{{.Repo.Owner}}/{{.Repo.Name}}/{{.Commit.Branch}}/{{.Commit.Sha}}">see results</a>)
</p>
<p>Repository : {{.Repo.Owner}}/{{.Repo.Name}}</p>
<p>Commit     : {{.Commit.ShaShort}}</p>
<p>Author     : {{.Commit.Author}}</p>
<p>Branch     : {{.Commit.Branch}}</p>
<p>Message:</p>
<p>{{ .Commit.Message }}</p>
`))
