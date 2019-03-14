// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import (
	"context"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/drone/drone/core"
	"github.com/drone/drone/logger"
	"github.com/drone/go-scm/scm"
)

// this is intended for local testing and instructs the handler
// to print the contents of the hook to stdout.
var debugPrintHook = false

func init() {
	debugPrintHook, _ = strconv.ParseBool(
		os.Getenv("DRONE_DEBUG_DUMP_HOOK"),
	)
}

// HandleHook returns an http.HandlerFunc that handles webhooks
// triggered by source code management.
func HandleHook(
	repos core.RepositoryStore,
	builds core.BuildStore,
	triggerer core.Triggerer,
	parser core.HookParser,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		if debugPrintHook {
			// if DRONE_DEBUG_DUMP_HOOK=true print the http.Request
			// headers and body to stdout.
			out, _ := httputil.DumpRequest(r, true)
			os.Stderr.Write(out)
		}

		hook, remote, err := parser.Parse(r, func(slug string) string {
			namespace, name := scm.Split(slug)
			repo, err := repos.FindName(r.Context(), namespace, name)
			if err != nil {
				logrus.WithFields(
					logrus.Fields{
						"namespace": namespace,
						"name":      name,
					}).Debugln("cannot find repository")
				return ""
			}
			return repo.Signer
		})

		if err != nil {
			logrus.Debugf("cannot parse webhook: %s", err)
			writeBadRequest(w, err)
			return
		}

		if hook == nil {
			logrus.Debugf("webhook ignored")
			return
		}

		// TODO handle ping requests
		// TODO consider using scm.Repository in the function callback.

		// // TODO break this to a separate function that
		// // we can unit test.
		// fn := func(webhook interface{}) (string, error) {
		// 	var remote scm.Repository
		// 	switch v := core.(type) {
		// 	case *scm.PushHook:
		// 		remote = v.Repo
		// 	case *scm.BranchHook:
		// 		remote = v.Repo
		// 	case *scm.TagHook:
		// 		remote = v.Repo
		// 	case *scm.PullRequestHook:
		// 		remote = v.Repo
		// 	case *scm.IssueHook:
		// 		remote = v.Repo
		// 	case *scm.IssueCommentHook:
		// 		remote = v.Repo
		// 	case *scm.PullRequestCommentHook:
		// 		remote = v.Repo
		// 	case *scm.ReviewCommentHook:
		// 		remote = v.Repo
		// 	}
		// 	repo, err := repos.FindName(r.Context(), remote.Namespace, remote.Name)
		// 	if err != nil {
		// 		hlog.FromRequest(r).Error().
		// 			Err(err).
		// 			Str("namespace", remote.Namespace).
		// 			Str("name", remote.Name).
		// 			Msg("cannot find repository")
		// 		return "", err
		// 	}
		// 	return repo.Token, nil
		// }

		// hook, err := client.Webhooks.Parse(r, fn)
		// if err != nil {
		// 	hlog.FromRequest(r).Error().
		// 		Err(err).
		// 		Msg("cannot parse webhook")
		// 	writeError(w, err)
		// 	return
		// }

		// var name, namespace string
		// var base = new(core.Build)
		// switch v := hook.(type) {
		// case *scm.PushHook:
		// 	namespace, name = v.Repo.Namespace, v.Repo.Name
		// 	base.Event = core.EventPush

		// 	base.Link = v.Commit.Link
		// 	base.Timestamp = v.Commit.Author.Date.Unix()
		// 	base.Title = ""
		// 	base.Message = v.Commit.Message
		// 	base.Before = ""
		// 	base.After = v.Commit.Sha
		// 	base.Ref = v.Ref
		// 	base.Source = strings.TrimPrefix(v.Ref, "refs/heads/")
		// 	base.Target = strings.TrimPrefix(v.Ref, "refs/heads/")
		// 	base.Author = v.Commit.Author.Login
		// 	base.AuthorName = v.Commit.Author.Name
		// 	base.AuthorEmail = v.Commit.Author.Email
		// 	base.AuthorAvatar = v.Commit.Author.Avatar
		// 	base.Sender = v.Sender.Login

		// 	// TODO: this is a deficiency with gogs
		// 	if base.AuthorAvatar == "" {
		// 		base.AuthorAvatar = v.Sender.Avatar
		// 	}
		// case *scm.TagHook:

		// 	namespace, name = v.Repo.Namespace, v.Repo.Name
		// 	base.Event = core.EventTag
		// 	base.Action = v.Action.String()

		// 	base.Link = ""     // TODO
		// 	base.Timestamp = 0 // TODO
		// 	base.Title = ""
		// 	base.Message = "" // TODO
		// 	base.Before = ""  // TODO
		// 	base.After = v.Ref.Sha
		// 	base.Ref = v.Ref.Name // TODO prepend refs/tags?
		// 	base.Source = ""
		// 	base.Target = ""
		// 	base.Author = v.Sender.Login
		// 	base.AuthorName = v.Sender.Name
		// 	base.AuthorEmail = v.Sender.Email
		// 	base.AuthorAvatar = v.Sender.Avatar
		// 	base.Sender = v.Sender.Login

		// 	switch v.Action {
		// 	case scm.ActionCreate:
		// 	default:
		// 		hlog.FromRequest(r).Debug().
		// 			Str("namespace", namespace).
		// 			Str("name", name).
		// 			Str("event", base.Event).
		// 			Str("action", base.Action).
		// 			Msgf("ignore webhook with action %s", base.Action)
		// 		w.WriteHeader(200)
		// 		return
		// 	}

		// case *scm.PullRequestHook:
		// 	namespace, name = v.Repo.Namespace, v.Repo.Name
		// 	base.Event = core.EventPullRequest
		// 	base.Action = v.Action.String()

		// 	base.Link = v.PullRequest.Link
		// 	base.Timestamp = v.PullRequest.Created.Unix()
		// 	base.Title = v.PullRequest.Title
		// 	base.Message = "" // TODO
		// 	base.Before = ""  // TODO
		// 	base.After = v.PullRequest.Sha
		// 	base.Ref = v.PullRequest.Ref
		// 	base.Source = v.PullRequest.Source
		// 	base.Target = v.PullRequest.Target
		// 	base.Author = v.PullRequest.Author.Login
		// 	base.AuthorName = v.PullRequest.Author.Name
		// 	base.AuthorEmail = v.PullRequest.Author.Email
		// 	base.AuthorAvatar = v.PullRequest.Author.Avatar
		// 	base.Sender = v.Sender.Login

		// 	switch v.Action {
		// 	case scm.ActionCreate, scm.ActionOpen, scm.ActionSync:
		// 	default:
		// 		hlog.FromRequest(r).Debug().
		// 			Str("namespace", namespace).
		// 			Str("name", name).
		// 			Str("event", base.Event).
		// 			Str("action", base.Action).
		// 			Msgf("ignore pull request hook with action %s", base.Action)
		// 		w.WriteHeader(200)
		// 		return
		// 	}
		// default:
		// 	w.WriteHeader(200)
		// 	return
		// }

		log := logrus.WithFields(logrus.Fields{
			"namespace": remote.Namespace,
			"name":      remote.Name,
			"event":     hook.Event,
			"commit":    hook.After,
		})

		log.Debugln("webhook parsed")

		repo, err := repos.FindName(r.Context(), remote.Namespace, remote.Name)
		if err != nil {
			log = log.WithError(err)
			log.Debugln("cannot find repository")
			writeNotFound(w, err)
			return
		}

		if !repo.Active {
			log.Debugln("ignore webhook, repository inactive")
			w.WriteHeader(200)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
		ctx = logger.WithContext(ctx, log)
		defer cancel()

		builds, err := triggerer.Trigger(ctx, repo, hook)
		if err != nil {
			writeError(w, err)
			return
		}

		writeJSON(w, builds, 200)
	}
}
