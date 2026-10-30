package telemetry

import (
	"strings"

	"github.com/getsentry/sentry-go"
)

func secrets() []string {
	out := make([]string, 0, 4)
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && s != "..." {
			out = append(out, s)
		}
	}
	add(configToken())
	add(configDSN())
	return out
}

func redactString(s string) string {
	if s == "" {
		return s
	}
	for _, secret := range secrets() {
		if secret == "" || !strings.Contains(s, secret) {
			continue
		}
		s = strings.ReplaceAll(s, secret, MaskSecret(secret))
	}
	return s
}

func redactValue(v interface{}) interface{} {
	switch t := v.(type) {
	case string:
		return redactString(t)
	case []interface{}:
		for i := range t {
			t[i] = redactValue(t[i])
		}
		return t
	case map[string]interface{}:
		for k, val := range t {
			t[k] = redactValue(val)
		}
		return t
	default:
		return v
	}
}

func redactEvent(event *sentry.Event) *sentry.Event {
	if event == nil {
		return nil
	}
	event.Message = redactString(event.Message)
	event.Transaction = redactString(event.Transaction)
	if event.Request != nil {
		event.Request.URL = redactString(event.Request.URL)
		event.Request.QueryString = redactString(event.Request.QueryString)
		event.Request.Data = redactString(event.Request.Data)
		event.Request.Cookies = redactString(event.Request.Cookies)
		for k, v := range event.Request.Headers {
			event.Request.Headers[k] = redactString(v)
		}
		for k, v := range event.Request.Env {
			event.Request.Env[k] = redactString(v)
		}
	}
	for k, v := range event.Extra {
		event.Extra[k] = redactValue(v)
	}
	for k, v := range event.Tags {
		event.Tags[k] = redactString(v)
	}
	for k, ctx := range event.Contexts {
		event.Contexts[k] = redactContext(ctx)
	}
	event.User.ID = redactString(event.User.ID)
	event.User.Username = redactString(event.User.Username)
	event.User.Email = redactString(event.User.Email)
	event.User.Name = redactString(event.User.Name)
	for i := range event.Exception {
		event.Exception[i].Type = redactString(event.Exception[i].Type)
		event.Exception[i].Value = redactString(event.Exception[i].Value)
	}
	for i := range event.Breadcrumbs {
		if event.Breadcrumbs[i] == nil {
			continue
		}
		event.Breadcrumbs[i].Message = redactString(event.Breadcrumbs[i].Message)
		event.Breadcrumbs[i].Category = redactString(event.Breadcrumbs[i].Category)
		event.Breadcrumbs[i].Data = redactStringMap(event.Breadcrumbs[i].Data)
	}
	return event
}

func redactContext(ctx sentry.Context) sentry.Context {
	if ctx == nil {
		return ctx
	}
	out := make(sentry.Context, len(ctx))
	for k, v := range ctx {
		out[k] = redactValue(v)
	}
	return out
}

func redactStringMap(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	for k, v := range data {
		data[k] = redactValue(v)
	}
	return data
}

func redactBreadcrumb(b *sentry.Breadcrumb) *sentry.Breadcrumb {
	if b == nil {
		return nil
	}
	b.Message = redactString(b.Message)
	b.Category = redactString(b.Category)
	b.Data = redactStringMap(b.Data)
	return b
}
