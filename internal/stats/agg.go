package stats

import (
	"sort"
	"time"
)

// Summary is the headline aggregation: totals, top-K, hit rates, daily buckets.
type Summary struct {
	Calls            int
	TokensSaved      int64
	ArgsTokens       int64
	ResponseTokens   int64
	NativeBaseline   int64
	CacheHitRate     float64 // (schema_hits + result_hits) / (2 * calls), 0..1
	SchemaHitRate    float64
	ResultHitRate    float64
	ErrorRate        float64
	AvgLatencyMS     float64
	P50LatencyMS     int64
	P95LatencyMS     int64

	TopTools     []ToolStat
	TopSavers    []ToolStat
	TopServers   []ServerStat
	Daily        []DailyBucket
	Recent       []Record // last N entries

	Projects []string // distinct project roots seen in window
	Servers  []string // distinct server names seen in window
	Agents   []string // distinct agent names seen in window
}

// ToolStat aggregates per-tool metrics.
type ToolStat struct {
	Server       string
	Tool         string
	Calls        int
	TokensSaved  int64
	AvgLatencyMS float64
	Errors       int
}

// ServerStat aggregates per-server metrics.
type ServerStat struct {
	Server       string
	Calls        int
	TokensSaved  int64
	AvgLatencyMS float64
	P95LatencyMS int64
	Errors       int
}

// DailyBucket aggregates by calendar day (UTC).
type DailyBucket struct {
	Day         time.Time // truncated to day
	Calls       int
	TokensSaved int64
	Errors      int
}

// Aggregate computes a Summary over records matching f. recentN controls how
// many tail records to retain in Summary.Recent.
func Aggregate(path string, f Filter, recentN int) (*Summary, error) {
	if recentN < 0 {
		recentN = 0
	}

	s := &Summary{}
	type tk struct{ srv, tool string }
	toolBy := map[tk]*ToolStat{}
	srvBy := map[string]*ServerStat{}
	srvLatencies := map[string][]int64{}
	dayBy := map[time.Time]*DailyBucket{}
	projects := map[string]struct{}{}
	servers := map[string]struct{}{}
	agents := map[string]struct{}{}

	var schemaHits, resultHits, errors int
	var latencies []int64
	var latencySum int64
	recent := newRing(recentN)

	err := Iter(path, f, func(r Record) error {
		s.Calls++
		s.TokensSaved += int64(r.TokensSaved)
		s.ArgsTokens += int64(r.ArgsTokensEst)
		s.ResponseTokens += int64(r.ResponseTokensEst)
		s.NativeBaseline += int64(r.NativeBaselineToks)
		latencySum += r.LatencyMS
		latencies = append(latencies, r.LatencyMS)
		if r.SchemaCacheHit {
			schemaHits++
		}
		if r.ResultCacheHit {
			resultHits++
		}
		if r.ExitCode != 0 {
			errors++
		}

		key := tk{r.Server, r.Tool}
		ts, ok := toolBy[key]
		if !ok {
			ts = &ToolStat{Server: r.Server, Tool: r.Tool}
			toolBy[key] = ts
		}
		ts.Calls++
		ts.TokensSaved += int64(r.TokensSaved)
		ts.AvgLatencyMS += float64(r.LatencyMS)
		if r.ExitCode != 0 {
			ts.Errors++
		}

		ss, ok := srvBy[r.Server]
		if !ok {
			ss = &ServerStat{Server: r.Server}
			srvBy[r.Server] = ss
		}
		ss.Calls++
		ss.TokensSaved += int64(r.TokensSaved)
		ss.AvgLatencyMS += float64(r.LatencyMS)
		srvLatencies[r.Server] = append(srvLatencies[r.Server], r.LatencyMS)
		if r.ExitCode != 0 {
			ss.Errors++
		}

		day := r.TS.UTC().Truncate(24 * time.Hour)
		db, ok := dayBy[day]
		if !ok {
			db = &DailyBucket{Day: day}
			dayBy[day] = db
		}
		db.Calls++
		db.TokensSaved += int64(r.TokensSaved)
		if r.ExitCode != 0 {
			db.Errors++
		}

		if r.Project != "" {
			projects[r.Project] = struct{}{}
		}
		if r.Server != "" {
			servers[r.Server] = struct{}{}
		}
		if r.Agent != "" {
			agents[r.Agent] = struct{}{}
		}

		recent.push(r)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.Calls > 0 {
		s.AvgLatencyMS = float64(latencySum) / float64(s.Calls)
		s.SchemaHitRate = float64(schemaHits) / float64(s.Calls)
		s.ResultHitRate = float64(resultHits) / float64(s.Calls)
		s.CacheHitRate = (s.SchemaHitRate + s.ResultHitRate) / 2
		s.ErrorRate = float64(errors) / float64(s.Calls)
		s.P50LatencyMS = percentile(latencies, 0.5)
		s.P95LatencyMS = percentile(latencies, 0.95)
	}

	for _, ts := range toolBy {
		if ts.Calls > 0 {
			ts.AvgLatencyMS /= float64(ts.Calls)
		}
		s.TopTools = append(s.TopTools, *ts)
	}
	sort.Slice(s.TopTools, func(i, j int) bool {
		return s.TopTools[i].Calls > s.TopTools[j].Calls
	})
	s.TopSavers = append([]ToolStat(nil), s.TopTools...)
	sort.Slice(s.TopSavers, func(i, j int) bool {
		return s.TopSavers[i].TokensSaved > s.TopSavers[j].TokensSaved
	})

	for _, ss := range srvBy {
		if ss.Calls > 0 {
			ss.AvgLatencyMS /= float64(ss.Calls)
			ss.P95LatencyMS = percentile(srvLatencies[ss.Server], 0.95)
		}
		s.TopServers = append(s.TopServers, *ss)
	}
	sort.Slice(s.TopServers, func(i, j int) bool {
		return s.TopServers[i].Calls > s.TopServers[j].Calls
	})

	for _, db := range dayBy {
		s.Daily = append(s.Daily, *db)
	}
	sort.Slice(s.Daily, func(i, j int) bool {
		return s.Daily[i].Day.Before(s.Daily[j].Day)
	})

	s.Recent = recent.snapshot()
	s.Projects = sortedKeys(projects)
	s.Servers = sortedKeys(servers)
	s.Agents = sortedKeys(agents)

	return s, nil
}

// percentile returns the requested percentile (0..1) of the slice.
// Mutates the slice (sorts it).
func percentile(v []int64, p float64) int64 {
	if len(v) == 0 {
		return 0
	}
	sort.Slice(v, func(i, j int) bool { return v[i] < v[j] })
	idx := int(float64(len(v)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(v) {
		idx = len(v) - 1
	}
	return v[idx]
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ring is a tiny fixed-size ring buffer for "last N" record retention.
type ring struct {
	buf  []Record
	idx  int
	size int
}

func newRing(n int) *ring {
	if n <= 0 {
		return &ring{}
	}
	return &ring{buf: make([]Record, n)}
}

func (r *ring) push(rec Record) {
	if r.buf == nil {
		return
	}
	r.buf[r.idx] = rec
	r.idx = (r.idx + 1) % len(r.buf)
	if r.size < len(r.buf) {
		r.size++
	}
}

func (r *ring) snapshot() []Record {
	if r.size == 0 {
		return nil
	}
	out := make([]Record, r.size)
	start := (r.idx - r.size + len(r.buf)) % len(r.buf)
	for i := 0; i < r.size; i++ {
		out[i] = r.buf[(start+i)%len(r.buf)]
	}
	return out
}
