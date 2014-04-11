package template

import (
	"crypto/md5"
	"errors"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/GeertJohan/go.rice"
)

// ErrTemplateNotFound indicates the requested template
// does not exists in the TemplateStore.
var ErrTemplateNotFound = errors.New("Template Not Found")

// registry stores a map of Templates where the key
// is the template name and the value is the *template.Template.
var registry = map[string]*template.Template{}

// ExecuteTemplate applies the template associated with t that has
// the given name to the specified data object and writes the output to wr.
func ExecuteTemplate(wr io.Writer, name string, data interface{}) error {
	templ, ok := registry[name]
	if !ok {
		return ErrTemplateNotFound
	}

	return templ.ExecuteTemplate(wr, "_", data)
}

// all template are loaded on initialization.
func init() {
	// location of templates
	box := rice.MustFindBox("pages")

	// these are all the files we need to parse. it is
	// kind of annoying that we can't list files in the
	// box, and have to enumerate each file here, but it is
	// a small price to pay to embed everything and simplify
	// the user installation process :)
	var files = []string{
		// these templates use the form.html
		// shared layout
		"login.html",
		"login_error.html",
		"forgot.html",
		"forgot_sent.html",
		"reset.html",
		"signup.html",
		"register.html",
		"install.html",

		// these templates use the default.html
		// shared layout
		"403.html",
		"404.html",
		"500.html",
		"user_dashboard.html",
		"user_password.html",
		"user_profile.html",
		"user_delete.html",
		"user_teams.html",
		"user_teams_add.html",
		"team_dashboard.html",
		"team_profile.html",
		"team_members.html",
		"team_delete.html",
		"members_add.html",
		"members_edit.html",
		"repo_dashboard.html",
		"repo_settings.html",
		"repo_delete.html",
		"repo_params.html",
		"repo_badges.html",
		"repo_keys.html",
		"repo_commit.html",
		"admin_users.html",
		"admin_users_edit.html",
		"admin_users_add.html",
		"admin_settings.html",
		"github_add.html",
		"github_limited_add.html",
		"github_link.html",
		"gitlab_add.html",
		"gitlab_link.html",
		"bitbucket_add.html",
		"bitbucket_link.html",
	}

	// extract the base template as a string
	base, err := box.String("base.html")
	if err != nil {
		panic(err)
	}

	assets := rice.MustFindBox("../../cmd/droned/assets")
	mainjs, err := assets.String("js/main.js")
	if err != nil {
		panic(err)
	}

	h := md5.New()
	io.WriteString(h, mainjs)
	jshash := fmt.Sprintf("%x", h.Sum(nil))
	base = strings.Replace(base, "main.js", "main.js?h="+jshash, 1)

	// extract the base form template as a string
	form, err := box.String("form.html")
	if err != nil {
		panic(err)
	}

	// loop through files and create templates
	for i, file := range files {
		// extract template from box
		page, err := box.String(file)
		if err != nil {
			panic(err)
		}

		// HACK: choose which base template to use FOR THE RECORD I
		// don't really like this, but it works for now.
		var baseTemplate = base
		if i < 8 {
			baseTemplate = form
		}

		// parse the template and then add to the global map
		baseParsed, err := template.New("_").Parse(baseTemplate)
		if err != nil {
			panic(fmt.Errorf("Error parsing base.html template: %s", err))
		}
		pageParsed, err := baseParsed.Parse(page)
		if err != nil {
			panic(fmt.Errorf("Error parsing page template for %s: %s", file, err))
		}

		registry[file] = pageParsed
	}

	// location of templates
	box = rice.MustFindBox("emails")

	files = []string{
		"activation.html",
		"failure.html",
		"success.html",
		"invitation.html",
		"reset_password.html",
	}

	// extract the base template as a string
	base, err = box.String("base_email.html")
	if err != nil {
		panic(err)
	}

	// loop through files and create templates
	for _, file := range files {
		// extract template from box
		email, err := box.String(file)
		if err != nil {
			panic(err)
		}
		baseParsed, err := template.New("_").Parse(base)
		if err != nil {
			panic(fmt.Errorf("Error parsing base_email.html template: %s", err))
		}
		emailParsed, err := baseParsed.Parse(email)
		if err != nil {
			panic(fmt.Errorf("Error parsing email template for %s: %s", file, err))
		}

		// parse the template and then add to the global map
		registry[file] = emailParsed
	}
}
