package telemetry

import (
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
