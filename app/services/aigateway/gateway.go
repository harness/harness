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

package aigateway

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/harness/gitness/app/services/aigateway/decisions"

	"github.com/rs/zerolog/log"
)

var ErrDisabled = errors.New("ai gateway is disabled")

const (
	decisionCacheTTL  = time.Hour
	decisionCacheSize = 1024
)

// DecideResult is the outcome of a batch of questions sent to the decision model.
type DecideResult struct {
	// Model is the exact build that answered, as reported by OpenRouter.
	Model string
	// Fallback is true when the primary decision model failed and the fallback answered.
	Fallback  bool
	Cached    bool
	Decisions map[string]decisions.Decision
}

// Gateway is the single path from Harness to OpenRouter. Every request goes through
// the free-only guard, the quota planner and the redactor.
type Gateway struct {
	config   Config
	client   *httpClient
	guard    *Guard
	catalog  *Catalog
	planner  *Planner
	redactor *Redactor

	cacheMu sync.Mutex
	cache   map[string]cachedDecision

	// sleep waits between retries; tests replace it to avoid real delays.
	sleep func(ctx context.Context, d time.Duration) error
}

type cachedDecision struct {
	result  DecideResult
	expires time.Time
}

func New(config Config, redactor *Redactor) *Gateway {
	config.Sanitize()

	client := &httpClient{
		client:  &http.Client{Timeout: config.HTTPTimeout},
		baseURL: config.BaseURL,
		apiKey:  config.APIKey,
	}
	guard := NewGuard()

	return &Gateway{
		config:   config,
		client:   client,
		guard:    guard,
		catalog:  NewCatalog(client, guard),
		planner:  NewPlanner(config.DailyRequestLimit, config.RequestsPerMinute, nil),
		redactor: redactor,
		cache:    map[string]cachedDecision{},
		sleep:    sleepContext,
	}
}

func (g *Gateway) Enabled() bool { return g != nil && g.config.Enabled }

func (g *Gateway) Guard() *Guard { return g.guard }

func (g *Gateway) Catalog() *Catalog { return g.catalog }

func (g *Gateway) Planner() *Planner { return g.planner }

func (g *Gateway) Quota() QuotaSnapshot { return g.planner.Snapshot() }

func (g *Gateway) Redact(s string) string { return g.redactor.Redact(s) }

// StartCatalogSync keeps the free model catalog fresh. Each replica keeps its own copy in memory.
func (g *Gateway) StartCatalogSync(ctx context.Context) {
	if !g.Enabled() {
		return
	}
	go func() {
		ticker := time.NewTicker(g.config.CatalogSyncInterval)
		defer ticker.Stop()
		for {
			if n, err := g.catalog.Sync(ctx); err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("ai gateway: failed to sync free model catalog")
			} else {
				log.Ctx(ctx).Info().Int("free_models", n).Msg("ai gateway: synced free model catalog")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

// Decide asks the decision model (Mammoth) a batch of questions about a redacted state.
// If the primary model fails, the fallback model (Elephant) is asked the same questions.
// Quota errors are returned as-is: the quota is per account, so the fallback can't help.
func (g *Gateway) Decide(
	ctx context.Context,
	sessionID string,
	state any,
	questions map[string]decisions.Question,
) (*DecideResult, error) {
	if !g.Enabled() {
		return nil, ErrDisabled
	}

	state, err := g.redactState(state)
	if err != nil {
		return nil, err
	}

	key, err := cacheKey(state, questions)
	if err != nil {
		return nil, err
	}
	if res, ok := g.cached(key); ok {
		return res, nil
	}

	models := []string{g.config.DecisionModel}
	if g.config.DecisionFallbackModel != "" && g.config.DecisionFallbackModel != g.config.DecisionModel {
		models = append(models, g.config.DecisionFallbackModel)
	}

	var errs []error
	for i, model := range models {
		res, err := g.decideWith(ctx, model, sessionID, state, questions)
		if err == nil {
			res.Fallback = i > 0
			g.store(key, *res)
			return res, nil
		}
		if errors.Is(err, ErrQuotaExhausted) || IsRateLimited(err) || ctx.Err() != nil {
			return nil, err
		}
		log.Ctx(ctx).Warn().Err(err).Str("model", model).Msg("ai gateway: decision model failed")
		errs = append(errs, fmt.Errorf("%s: %w", model, err))
	}
	return nil, errors.Join(errs...)
}

func (g *Gateway) decideWith(
	ctx context.Context,
	model, sessionID string,
	state any,
	questions map[string]decisions.Question,
) (*DecideResult, error) {
	if err := g.guard.Check(model); err != nil {
		return nil, err
	}

	req := &decisions.Request{
		Model:     model,
		State:     state,
		Questions: questions,
		SessionID: sessionID,
		Provider:  FreeOnlyProvider(),
	}
	if err := req.Validate(); err != nil {
		return nil, err
	}

	var resp decisions.Response
	err := g.withRetry(ctx, ClassDecision, func() error {
		return g.client.doJSON(ctx, http.MethodPost, decisions.Path, req, &resp)
	})
	if err != nil {
		return nil, err
	}
	if err := g.guard.VerifyCost(model, resp.Usage.Cost); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("ai gateway: free-only guarantee violated")
		return nil, err
	}

	parsed, err := decisions.Parse(req, &resp)
	if err != nil {
		return nil, err
	}

	return &DecideResult{Model: resp.Model, Decisions: parsed}, nil
}

// redactState redacts the state through its JSON form. Placeholders never contain quotes,
// so the redacted JSON stays valid; if it doesn't, the redacted text is sent as a string state.
func (g *Gateway) redactState(state any) (any, error) {
	if s, ok := state.(string); ok {
		return g.redactor.Redact(s), nil
	}

	raw, err := json.Marshal(state)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal decision state: %w", err)
	}
	redacted := g.redactor.Redact(string(raw))

	var out any
	if err := json.Unmarshal([]byte(redacted), &out); err != nil {
		return redacted, nil //nolint:nilerr // fall back to a string state
	}
	return out, nil
}

func cacheKey(state any, questions map[string]decisions.Question) (string, error) {
	// json.Marshal sorts map keys, so equal inputs produce equal keys
	raw, err := json.Marshal(struct {
		State     any                           `json:"s"`
		Questions map[string]decisions.Question `json:"q"`
	}{state, questions})
	if err != nil {
		return "", fmt.Errorf("failed to build decision cache key: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (g *Gateway) cached(key string) (*DecideResult, bool) {
	g.cacheMu.Lock()
	defer g.cacheMu.Unlock()

	c, ok := g.cache[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(c.expires) {
		delete(g.cache, key)
		return nil, false
	}
	res := c.result
	res.Cached = true
	return &res, true
}

func (g *Gateway) store(key string, res DecideResult) {
	g.cacheMu.Lock()
	defer g.cacheMu.Unlock()

	now := time.Now()
	if len(g.cache) >= decisionCacheSize {
		for k, c := range g.cache {
			if now.After(c.expires) {
				delete(g.cache, k)
			}
		}
		if len(g.cache) >= decisionCacheSize {
			g.cache = map[string]cachedDecision{}
		}
	}
	g.cache[key] = cachedDecision{result: res, expires: now.Add(decisionCacheTTL)}
}
