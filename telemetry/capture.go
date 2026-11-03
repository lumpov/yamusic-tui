package telemetry

import (
	"fmt"
	"strconv"

	"github.com/getsentry/sentry-go"
)

func CaptureException(err error) {
	if !started || err == nil {
		return
	}
	sentry.CaptureException(err)
}

func CaptureMessage(msg string) {
	if !started || msg == "" {
		return
	}
	sentry.CaptureMessage(msg)
}

func SetTag(key, value string) {
	if !started || key == "" {
		return
	}
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag(key, value)
	})
}

func AddBreadcrumb(category, message string, data map[string]interface{}) {
	if !started || message == "" {
		return
	}
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Type:     "info",
		Category: category,
		Message:  message,
		Level:    sentry.LevelInfo,
		Data:     data,
	})
}

func CaptureBuffering(err error, tags map[string]string, extra map[string]interface{}) {
	if !started || err == nil {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("kind", "track-buffering")
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		for k, v := range extra {
			scope.SetExtra(k, v)
		}
		scope.SetFingerprint([]string{"track-buffering"})
		sentry.CaptureException(fmt.Errorf("track buffering: %w", err))
	})
}

func CaptureWhenActive(msg string, tags map[string]string) {
	if !started || msg == "" {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelWarning)
		scope.SetTag("kind", "when-active")
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		scope.SetFingerprint([]string{"when-active", msg})
		sentry.CaptureMessage("when-active: " + msg)
	})
}

func CaptureAPI(err error, method, path string, status int) {
	if !started || err == nil {
		return
	}
	sentry.WithScope(func(scope *sentry.Scope) {
		if method != "" {
			scope.SetTag("http_method", method)
		}
		if path != "" {
			scope.SetTag("yandex_api_path", path)
		}
		if status != 0 {
			scope.SetTag("http_status", strconv.Itoa(status))
		}
		scope.SetFingerprint([]string{"yandex-api", method, path, strconv.Itoa(status)})
		sentry.CaptureException(err)
	})
}
