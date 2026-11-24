package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/dukex/hark"
)

type fakeProvider []hark.Check

func (fakeProvider) Watch(context.Context, func(hark.Event)) error { return nil }
func (f fakeProvider) Doctor(context.Context) []hark.Check         { return f }

func withProviders(t *testing.T, p map[string]constructor) {
	t.Helper()
	old, oldVersion := providers, version
	t.Cleanup(func() { providers, version = old, oldVersion })
	providers, version = p, "0.7.0"
}

func provider(p hark.Provider, err error) constructor {
	return func(func(string) string, options) (hark.Provider, error) { return p, err }
}

func TestDoctorReportsEveryProvider(t *testing.T) {
	withProviders(t, map[string]constructor{
		"alpha": provider(fakeProvider{{Name: "token", Detail: "logged in as duke"}}, nil),
		"beta":  provider(nil, fmt.Errorf("%w: set BETA_TOKEN", errNotConfigured)),
		"gamma": provider(fakeProvider{{Name: "token", Detail: "valid"}, {Name: "activity log", Err: errors.New("403 Forbidden")}}, nil),
		"delta": provider(nil, errors.New("no home directory")),
	})
	var stdout bytes.Buffer
	code := run(context.Background(), []string{"doctor"}, env(nil), &stdout, &bytes.Buffer{})
	want := `hark 0.7.0
alpha
  ✓ token: logged in as duke
beta
  - not configured: set BETA_TOKEN
delta
  ✗ setup: no home directory
gamma
  ✓ token: valid
  ✗ activity log: 403 Forbidden
`
	if code != 1 || stdout.String() != want {
		t.Errorf("exit %d\ngot:\n%s\nwant:\n%s", code, stdout.String(), want)
	}
}

func TestDoctorPassesWhenConfiguredProvidersWork(t *testing.T) {
	withProviders(t, map[string]constructor{
		"alpha": provider(fakeProvider{{Name: "token", Detail: "ok"}}, nil),
		"beta":  provider(nil, fmt.Errorf("%w: set BETA_TOKEN", errNotConfigured)),
	})
	if code := run(context.Background(), []string{"doctor"}, env(nil), &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Errorf("exit %d", code)
	}
}

func TestDoctorFailsWithNothingConfigured(t *testing.T) {
	withProviders(t, map[string]constructor{
		"beta": provider(nil, fmt.Errorf("%w: set BETA_TOKEN", errNotConfigured)),
	})
	var stdout bytes.Buffer
	if code := run(context.Background(), []string{"doctor"}, env(nil), &stdout, &bytes.Buffer{}); code != 1 {
		t.Errorf("exit %d", code)
	}
	if want := "hark 0.7.0\nbeta\n  - not configured: set BETA_TOKEN\nno provider is configured\n"; stdout.String() != want {
		t.Errorf("got %q", stdout.String())
	}
}

func TestMissingConfigurationIsNotConfigured(t *testing.T) {
	defer func(f func() (string, error)) { ghToken = f }(ghToken)
	ghToken = func() (string, error) { return "", errors.New("not logged in") }
	for name, newProvider := range providers {
		if _, err := newProvider(env(nil), options{}); !errors.Is(err, errNotConfigured) {
			t.Errorf("%s: %v is not errNotConfigured", name, err)
		}
	}
}
