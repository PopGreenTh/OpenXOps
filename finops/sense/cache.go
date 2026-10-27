package sense

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PopGreenTh/OpenXOps/aws/ce"
)

const cacheMaxAge = 12 * time.Hour

type cacheEntry struct {
	Results []ce.ResultByTime `json:"results"`
}

type cacheStore struct {
	root              string
	managementAccount string
	now               time.Time
}

func cacheRoot(mock bool) string {
	if mock {
		return filepath.Join("data", "mock", "cache", "aws", "ce")
	}
	return filepath.Join("data", "cache", "aws", "ce")
}

func (c *cacheStore) dir(level int, recordGranularity, metric, tagKey, account string) string {
	name := fmt.Sprintf("%d-%s-%s-%s", level, recordGranularity, metric, tagKey)
	if account != "" {
		name += "-" + account
	}
	return filepath.Join(c.root, c.managementAccount, name)
}

type cacheCandidate struct {
	path string
	ts   time.Time
}

func (c *cacheStore) read(level int, recordGranularity, metric, tagKey, account, period string, refresh bool) (*cacheEntry, bool) {
	if refresh {
		slog.Debug("cache miss", slog.String("period", period), slog.String("reason", "refresh"))
		return nil, false
	}
	dir := c.dir(level, recordGranularity, metric, tagKey, account)
	var cands []cacheCandidate
	plain := filepath.Join(dir, period+".json")
	if data, err := os.ReadFile(plain); err == nil {
		var ent cacheEntry
		if json.Unmarshal(data, &ent) == nil {
			cands = append(cands, cacheCandidate{plain, time.Time{}})
		}
	}
	entries, _ := filepath.Glob(filepath.Join(dir, period+"@*.json"))
	for _, path := range entries {
		cands = append(cands, cacheCandidate{path, timestampFromCacheFile(path)})
	}
	if len(cands) == 0 {
		slog.Debug("cache miss", slog.String("period", period))
		return nil, false
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].ts.Equal(cands[j].ts) {
			return cands[i].path < cands[j].path
		}
		return cands[i].ts.Before(cands[j].ts)
	})
	best := cands[len(cands)-1]
	if !best.ts.IsZero() && c.now.Sub(best.ts) > cacheMaxAge {
		for i := len(cands) - 1; i >= 0; i-- {
			if cands[i].ts.IsZero() {
				best = cands[i]
				break
			}
			if c.now.Sub(cands[i].ts) <= cacheMaxAge {
				best = cands[i]
				break
			}
		}
		if !best.ts.IsZero() && c.now.Sub(best.ts) > cacheMaxAge {
			slog.Debug("cache miss", slog.String("period", period), slog.String("reason", "stale"))
			return nil, false
		}
	}
	data, err := os.ReadFile(best.path)
	if err != nil {
		slog.Debug("cache miss", slog.String("period", period))
		return nil, false
	}
	slog.Debug("cache hit", slog.String("period", period), slog.String("file", best.path))
	var ent cacheEntry
	if err := json.Unmarshal(data, &ent); err != nil {
		return nil, false
	}
	return &ent, true
}

func (c *cacheStore) write(level int, recordGranularity, metric, tagKey, account, period string, closed, refresh bool, results []ce.ResultByTime) error {
	dir := c.dir(level, recordGranularity, metric, tagKey, account)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ent := cacheEntry{Results: results}
	data, err := json.Marshal(ent)
	if err != nil {
		return err
	}
	var path string
	if closed && !refresh {
		path = filepath.Join(dir, period+".json")
		if _, err := os.Stat(path); err == nil {
			return nil
		}
	} else {
		path = filepath.Join(dir, fmt.Sprintf("%s@%s.json", period, c.now.UTC().Format("20060102T150405Z")))
	}
	return os.WriteFile(path, data, 0o644)
}

func timestampFromCacheFile(path string) time.Time {
	base := filepath.Base(path)
	i := strings.Index(base, "@")
	if i < 0 {
		return time.Time{}
	}
	j := strings.Index(base[i+1:], ".")
	if j < 0 {
		return time.Time{}
	}
	ts := base[i+1 : i+1+j]
	t, err := time.Parse("20060102T150405Z", ts)
	if err != nil {
		return time.Time{}
	}
	return t
}

func filterResultsForPeriod(results []ce.ResultByTime, period, ceGranularity string) []ce.ResultByTime {
	wantStart, _ := periodAPIBounds(period, ceGranularity)
	var out []ce.ResultByTime
	for _, r := range results {
		if r.Start == wantStart {
			out = append(out, r)
		}
	}
	return out
}
