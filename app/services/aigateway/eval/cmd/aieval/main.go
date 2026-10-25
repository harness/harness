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

// Command aieval evaluates the Mammoth decision questions against a labelled dataset.
//
// Offline (no network): replay recorded answers.
//
//	go run ./app/services/aigateway/eval/cmd/aieval -replay answers.jsonl -report-md report.md
//
// Online: set GITNESS_AI_GATEWAY_API_KEY (and optionally GITNESS_AI_GATEWAY_BASE_URL,
// GITNESS_AI_GATEWAY_DECISION_MODEL, GITNESS_AI_GATEWAY_DECISION_FALLBACK_MODEL,
// GITNESS_AI_GATEWAY_DAILY_REQUEST_LIMIT) and record the answers for later comparison.
//
//	go run ./app/services/aigateway/eval/cmd/aieval -record answers.jsonl -report-md report.md
//
// The dataset needs one request per case, so keep the daily request limit in mind.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/aigateway/eval"
)

const defaultDataset = "app/services/aigateway/eval/testdata/cases.jsonl"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "aieval:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		dataset    = flag.String("dataset", defaultDataset, "JSONL dataset of labelled cases")
		replay     = flag.String("replay", "", "JSONL file of recorded answers; when set no network is used")
		record     = flag.String("record", "", "write the live answers to this JSONL file (live mode only)")
		reportMD   = flag.String("report-md", "", "write the markdown report to this file (default stdout)")
		reportJSON = flag.String("report-json", "", "write the JSON summary to this file")
		baseline   = flag.String("baseline", "", "JSON summary of an earlier run to compare against")
		groups     = flag.String("groups", "", "comma separated groups to run: shield,insights,generic (default all)")
		target     = flag.Float64("target", eval.DefaultTargetAccuracy, "accuracy a suggested threshold must reach")
		minSupport = flag.Int("min-support", eval.DefaultMinSupport, "fewest answers a suggested threshold may rest on")
	)
	flag.Parse()

	cases, err := eval.LoadCasesFile(*dataset)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var decider eval.Decider
	if *replay != "" {
		if *record != "" {
			return fmt.Errorf("-record needs live mode; drop -replay")
		}
		if decider, err = eval.NewReplayDeciderFile(*replay); err != nil {
			return err
		}
	} else {
		if decider, err = liveDecider(); err != nil {
			return err
		}
		if *record != "" {
			f, ferr := os.OpenFile(*record, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
			if ferr != nil {
				return fmt.Errorf("failed to open record file: %w", ferr)
			}
			defer f.Close()
			decider = eval.NewRecorder(decider, f)
		}
	}

	opt := eval.Options{TargetAccuracy: *target, MinSupport: *minSupport}
	for _, g := range strings.Split(*groups, ",") {
		if g = strings.TrimSpace(g); g != "" {
			opt.Groups = append(opt.Groups, eval.Group(g))
		}
	}
	rep, err := eval.Run(ctx, cases, decider, opt)
	if err != nil {
		return err
	}

	var base *eval.Report
	if *baseline != "" {
		if base, err = eval.LoadReport(*baseline); err != nil {
			return err
		}
	}

	md := os.Stdout
	if *reportMD != "" {
		f, ferr := os.Create(*reportMD)
		if ferr != nil {
			return fmt.Errorf("failed to create markdown report: %w", ferr)
		}
		defer f.Close()
		md = f
	}
	if err = rep.WriteMarkdown(md, base); err != nil {
		return err
	}
	if *reportJSON != "" {
		f, ferr := os.Create(*reportJSON)
		if ferr != nil {
			return fmt.Errorf("failed to create json summary: %w", ferr)
		}
		defer f.Close()
		if err = rep.WriteJSON(f); err != nil {
			return err
		}
	}
	return nil
}

// liveDecider builds the real gateway from the GITNESS_AI_GATEWAY_* variables.
func liveDecider() (eval.Decider, error) {
	key := os.Getenv("GITNESS_AI_GATEWAY_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("no -replay file and GITNESS_AI_GATEWAY_API_KEY is not set")
	}
	cfg := aigateway.Config{
		Enabled:               true,
		BaseURL:               os.Getenv("GITNESS_AI_GATEWAY_BASE_URL"),
		APIKey:                key,
		DecisionModel:         os.Getenv("GITNESS_AI_GATEWAY_DECISION_MODEL"),
		DecisionFallbackModel: os.Getenv("GITNESS_AI_GATEWAY_DECISION_FALLBACK_MODEL"),
	}
	cfg.DailyRequestLimit, _ = strconv.Atoi(os.Getenv("GITNESS_AI_GATEWAY_DAILY_REQUEST_LIMIT"))
	cfg.RequestsPerMinute, _ = strconv.Atoi(os.Getenv("GITNESS_AI_GATEWAY_REQUESTS_PER_MINUTE"))
	redactor, err := aigateway.NewRedactor()
	if err != nil {
		return nil, err
	}
	return eval.GatewayDecider{Gateway: aigateway.New(cfg, redactor)}, nil
}
