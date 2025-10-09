// Source: https://gitlab.com/gitlab-org/container-registry

// Copyright 2019 Gitlab Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"

	repostore "github.com/harness/gitness/registry/app/store/database"
	"github.com/harness/gitness/registry/types"
)

const (
	linkPrevious             = "previous"
	linkNext                 = "next"
	encodingSeparator        = "|"
	NQueryParamKey           = "n"
	PublishedAtQueryParamKey = "published_at"
	BeforeQueryParamKey      = "before"
	TagNameQueryParamKey     = "name"
	SortQueryParamKey        = "sort"
	LastQueryParamKey        = "last"
)

// Use the original URL from the request to create a new URL for
// the link header.
func CreateLinkEntry(
	origURL string,
	filters types.FilterParams,
	publishedBefore, publishedLast string,
) (string, error) {
	var combinedURL string

	if filters.BeforeEntry != "" {
		beforeURL, err := generateLink(origURL, linkPrevious, filters, publishedBefore, publishedLast)
		if err != nil {
			return "", err
		}
		combinedURL = beforeURL
	}

	if filters.LastEntry != "" {
		lastURL, err := generateLink(origURL, linkNext, filters, publishedBefore, publishedLast)
		if err != nil {
			return "", err
		}

		if filters.BeforeEntry == "" {
			combinedURL = lastURL
		} else {
			// Put the "previous" URL first and then "next" as shown in the
			// RFC5988 examples https://datatracker.ietf.org/doc/html/rfc5988#section-5.5.
			combinedURL = fmt.Sprintf("%s, %s", combinedURL, lastURL)
		}
	}

	return combinedURL, nil
}

func generateLink(
	originalURL, rel string,
	filters types.FilterParams,
	publishedBefore, publishedLast string,
) (string, error) {
	calledURL, err := url.Parse(originalURL)
	if err != nil {
		return "", err
	}

	qValues := url.Values{}
	qValues.Add(NQueryParamKey, strconv.Itoa(filters.MaxEntries))

	switch rel {
	case linkPrevious:
		before := filters.BeforeEntry
		if filters.OrderBy == PublishedAtQueryParamKey && publishedBefore != "" {
			before = EncodeFilter(publishedBefore, filters.BeforeEntry)
		}
		qValues.Add(BeforeQueryParamKey, before)
	case linkNext:
		last := filters.LastEntry
		if filters.OrderBy == PublishedAtQueryParamKey && publishedLast != "" {
			last = EncodeFilter(publishedLast, filters.LastEntry)
		}
		qValues.Add(LastQueryParamKey, last)
	}

	if filters.Name != "" {
		qValues.Add(TagNameQueryParamKey, filters.Name)
	}

	orderBy := filters.OrderBy
	if orderBy != "" {
		if filters.SortOrder == repostore.OrderDesc {
			orderBy = "-" + orderBy
		}
		qValues.Add(SortQueryParamKey, orderBy)
	}

	calledURL.RawQuery = qValues.Encode()

	calledURL.Fragment = ""
	urlStr := fmt.Sprintf("<%s>; rel=\"%s\"", calledURL.String(), rel)

	return urlStr, nil
}

// EncodeFilter base64 encode by concatenating the published_at value with the tagName using an encodingSeparator.
func EncodeFilter(publishedAt, tagName string) (v string) {
	return base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%s%s%s", publishedAt, encodingSeparator, tagName)),
	)
}
