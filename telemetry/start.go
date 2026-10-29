package telemetry

import (
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dece2183/yamusic-tui/config"
	"github.com/getsentry/sentry-go"
)

var started bool

func configToken() string {
	return strings.TrimSpace(config.Current.Token)
}

func configDSN() string {
	return strings.TrimSpace(config.Current.SentryDSN)
}

func Start() {
	dsn := configDSN()
	if dsn == "" {
		started = false
		return
	}

	opts := sentry.ClientOptions{
		Dsn:              dsn,
		Release:          config.Version,
		Dist:             runtime.GOARCH,
		Environment:      runtime.GOOS,
		AttachStacktrace: true,
		SendDefaultPII:   false,
		EnableTracing:    false,
		MaxBreadcrumbs:   50,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			return redactEvent(event)
		},
		BeforeBreadcrumb: func(breadcrumb *sentry.Breadcrumb, hint *sentry.BreadcrumbHint) *sentry.Breadcrumb {
			return redactBreadcrumb(breadcrumb)
		},
	}
	if host, err := os.Hostname(); err == nil {
		opts.ServerName = host
	}
	if client := sentryHTTPClient(); client != nil {
		opts.HTTPClient = client
	}

	if err := sentry.Init(opts); err != nil {
		started = false
		return
	}
	started = true

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("goos", runtime.GOOS)
		scope.SetTag("goarch", runtime.GOARCH)
		scope.SetTag("go_version", runtime.Version())
		scope.SetTag("cgo", cgoEnabled())
		scope.SetTag("num_cpu", strconv.Itoa(runtime.NumCPU()))
		scope.SetTag("app_unofficial", config.AppUnofficialNote)
		scope.SetContext("app", map[string]interface{}{
			"name":              config.AppTitle,
			"version":           config.Version,
			"unofficial":        config.AppUnofficialNote,
			"cache_tracks":      cacheTracksName(config.Current.CacheTracks),
			"show_lyrics":       config.Current.ShowLyrics,
			"suppress_errors":   config.Current.SuppressErrors,
			"buffer_size_ms":    config.Current.BufferSize,
			"rewind_duration_s": config.Current.RewindDuration,
			"proxy_configured":  strings.TrimSpace(config.Current.Proxy) != "",
			"token_masked":      MaskSecret(configToken()),
		})
		scope.SetContext("runtime", map[string]interface{}{
			"go":         runtime.Version(),
			"compiler":   runtime.Compiler,
			"num_cpu":    runtime.NumCPU(),
			"gomaxprocs": runtime.GOMAXPROCS(0),
		})
	})
}

func Stop() {
	if !started {
		return
	}
	sentry.Flush(2 * time.Second)
	started = false
}

func RecoverPanic() {
	r := recover()
	if r == nil {
		return
	}
	if started {
		sentry.CurrentHub().Recover(r)
		sentry.Flush(2 * time.Second)
	}
	panic(r)
}

func sentryHTTPClient() *http.Client {
	proxy := strings.TrimSpace(config.Current.Proxy)
	if proxy == "" {
		return nil
	}
	u, err := url.Parse(proxy)
	if err != nil {
		return nil
	}
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyURL(u),
		},
	}
}

func cacheTracksName(t config.CacheType) string {
	switch t {
	case config.CACHE_LIKED_ONLY:
		return "likes"
	case config.CACHE_ALL:
		return "all"
	default:
		return "none"
	}
}

func cgoEnabled() string {
	if Cgo {
		return "1"
	}
	return "0"
}
