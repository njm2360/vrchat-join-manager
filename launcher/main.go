package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/singleflight"
)

const cacheTTL = time.Minute

var jst = time.FixedZone("JST", 9*60*60)

var groupIDPattern = regexp.MustCompile(`^grp_[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type config struct {
	ListenAddr    string
	APIBaseURL    string
	AllowedGroups map[string]bool
}

func loadConfig() (config, error) {
	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8090"
	}
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8080"
	}
	allowed := make(map[string]bool)
	for id := range strings.SplitSeq(os.Getenv("ALLOWED_GROUPS"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			allowed[id] = true
		}
	}
	if len(allowed) == 0 {
		return config{}, errors.New("ALLOWED_GROUPS must be set")
	}
	return config{
		ListenAddr:    listenAddr,
		APIBaseURL:    strings.TrimRight(apiBaseURL, "/"),
		AllowedGroups: allowed,
	}, nil
}

type instanceOut struct {
	LocationID string  `json:"location_id"`
	WorldID    string  `json:"world_id"`
	InstanceID *string `json:"instance_id"`
	WorldName  *string `json:"world_name"`
	Region     *string `json:"region"`
	UserCount  int     `json:"user_count"`
	OpenedAt   string  `json:"opened_at"`
}

type instanceView struct {
	WorldName  string
	InstanceID string
	Region     string
	UserCount  int
	OpenedAt   string
	JoinURL    string
}

type pageData struct {
	GroupID   string
	Instances []instanceView
	Error     string
}

var pageTmpl = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="ja">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.GroupID}} - Join</title>
<style>
  :root {
    --bg: #f4f5f7; --card: #fff; --border: #e2e4e8; --text: #1c1e21;
    --muted: #6b7280; --pill: #eef0f3; --accent: #2563eb; --accent-hover: #1d4ed8;
  }
  @media (prefers-color-scheme: dark) {
    :root {
      --bg: #14161a; --card: #1d2025; --border: #2c3038; --text: #e7e9ec;
      --muted: #9aa1ab; --pill: #2c3038; --accent: #3b82f6; --accent-hover: #2f6fe0;
    }
  }
  * { box-sizing: border-box; }
  body { font-family: system-ui, sans-serif; background: var(--bg); color: var(--text); max-width: 640px; margin: 0 auto; padding: 2.5rem 1rem 4rem; }
  h1 { font-size: 1.3rem; margin: 0; }
  .group-id { font-family: ui-monospace, monospace; font-size: .78rem; color: var(--muted); word-break: break-all; margin: .4rem 0 1.5rem; }
  .card { background: var(--card); border: 1px solid var(--border); border-radius: 12px; padding: 1rem 1.25rem; margin-bottom: .75rem; display: flex; align-items: center; gap: 1rem; }
  .info { flex: 1; min-width: 0; }
  .world { font-weight: 600; line-height: 1.4; word-break: break-all; }
  .instance-id { font-weight: 400; color: var(--muted); }
  .meta { display: flex; align-items: center; flex-wrap: wrap; gap: .35rem .7rem; margin-top: .45rem; font-size: .85rem; color: var(--muted); }
  .region { background: var(--pill); border-radius: 999px; padding: .1rem .55rem; font-size: .72rem; font-weight: 600; text-transform: uppercase; letter-spacing: .03em; }
  .join { flex-shrink: 0; background: var(--accent); color: #fff; text-decoration: none; font-weight: 600; font-size: .9rem; padding: .55rem 1.2rem; border-radius: 8px; }
  .join:hover { background: var(--accent-hover); }
  .empty { background: var(--card); border: 1px solid var(--border); border-radius: 12px; padding: 2rem 1reSm; text-align: center; color: var(--muted); }
  .error { color: #e5484d; }
</style>
</head>
<body>
<h1>オープン中のインスタンス</h1>
{{if .GroupID}}<p class="group-id">{{.GroupID}}</p>{{end}}
{{if .Error}}
  <p class="error">{{.Error}}</p>
{{else if not .Instances}}
  <p class="empty">現在オープン中のインスタンスはありません。</p>
{{else}}
  {{range .Instances}}
  <div class="card">
    <div class="info">
      <div class="world">{{.WorldName}}{{if .InstanceID}} <span class="instance-id">#{{.InstanceID}}</span>{{end}}</div>
      <div class="meta">
        {{if .Region}}<span class="region">{{.Region}}</span>{{end}}
        <span>{{.UserCount}}人</span>
        <span>オープン: {{.OpenedAt}}</span>
      </div>
    </div>
    <a class="join" href="{{.JoinURL}}" target="_blank" rel="noopener noreferrer">Join</a>
  </div>
  {{end}}
{{end}}
</body>
</html>`))

type cacheEntry struct {
	views     []instanceView
	expiresAt time.Time
}

type server struct {
	cfg    config
	client *http.Client

	g singleflight.Group

	mu    sync.Mutex
	cache map[string]cacheEntry
}

func (s *server) apiGet(ctx context.Context, path string, query url.Values, out any) error {
	u := s.cfg.APIBaseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api %s: unexpected status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func joinURL(locationID string) (string, error) {
	worldID, instanceID, ok := strings.Cut(locationID, ":")
	if !ok || worldID == "" || instanceID == "" {
		return "", fmt.Errorf("invalid location_id: %q", locationID)
	}
	v := url.Values{}
	v.Set("worldId", worldID)
	v.Set("instanceId", instanceID)
	return "https://vrchat.com/home/launch?" + v.Encode(), nil
}

func (s *server) openInstances(groupID string) ([]instanceView, error) {
	s.mu.Lock()
	entry, ok := s.cache[groupID]
	s.mu.Unlock()
	if ok && time.Now().Before(entry.expiresAt) {
		return entry.views, nil
	}

	v, err, _ := s.g.Do(groupID, func() (any, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		views, err := s.fetchOpenInstances(ctx, groupID)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		s.cache[groupID] = cacheEntry{views: views, expiresAt: time.Now().Add(cacheTTL)}
		s.mu.Unlock()
		return views, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]instanceView), nil
}

func (s *server) fetchOpenInstances(ctx context.Context, groupID string) ([]instanceView, error) {
	var instances []instanceOut
	q := url.Values{}
	q.Set("group_id", groupID)
	q.Set("is_open", "true")
	if err := s.apiGet(ctx, "/api/instances", q, &instances); err != nil {
		return nil, err
	}

	views := make([]instanceView, 0, len(instances))
	for _, inst := range instances {
		link, err := joinURL(inst.LocationID)
		if err != nil {
			slog.Warn("skip instance with invalid location_id", "location_id", inst.LocationID, "err", err)
			continue
		}
		worldName := inst.WorldID
		if inst.WorldName != nil && *inst.WorldName != "" {
			worldName = *inst.WorldName
		}
		region := ""
		if inst.Region != nil {
			region = *inst.Region
		}
		instanceID := ""
		if inst.InstanceID != nil {
			instanceID = *inst.InstanceID
		}
		openedAt := inst.OpenedAt
		if t, err := time.Parse(time.RFC3339, inst.OpenedAt); err == nil {
			openedAt = t.In(jst).Format("2006-01-02 15:04")
		}
		views = append(views, instanceView{
			WorldName:  worldName,
			InstanceID: instanceID,
			Region:     region,
			UserCount:  inst.UserCount,
			OpenedAt:   openedAt,
			JoinURL:    link,
		})
	}
	return views, nil
}

func (s *server) handleLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h := w.Header()
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Frame-Options", "DENY")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; form-action 'none'")

	groupID := strings.TrimSpace(r.URL.Query().Get("group"))
	if groupID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = pageTmpl.Execute(w, pageData{Error: "?group=<group_id> を指定してください。"})
		return
	}
	if !groupIDPattern.MatchString(groupID) {
		w.WriteHeader(http.StatusBadRequest)
		_ = pageTmpl.Execute(w, pageData{Error: "グループIDの形式が不正です。"})
		return
	}
	if !s.cfg.AllowedGroups[groupID] {
		w.WriteHeader(http.StatusNotFound)
		_ = pageTmpl.Execute(w, pageData{Error: "このグループは公開されていません。"})
		return
	}

	views, err := s.openInstances(groupID)
	if err != nil {
		slog.Error("fetch instances failed", "err", err)
		w.WriteHeader(http.StatusBadGateway)
		_ = pageTmpl.Execute(w, pageData{GroupID: groupID, Error: "インスタンス情報の取得に失敗しました。"})
		return
	}

	_ = pageTmpl.Execute(w, pageData{GroupID: groupID, Instances: views})
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		slog.Error("load config failed", "err", err)
		os.Exit(1)
	}
	s := &server{
		cfg:    cfg,
		client: &http.Client{Timeout: 5 * time.Second},
		cache:  make(map[string]cacheEntry),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/launch", s.handleLaunch)

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 << 10,
	}

	go func() {
		slog.Info("listening", "addr", cfg.ListenAddr, "api_base_url", cfg.APIBaseURL)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	stop()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
}
