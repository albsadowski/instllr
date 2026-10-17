package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestProxyTemplate(t *testing.T) {
	cases := []struct {
		name        string
		serverNames []string
		ws          bool
		wantNames   string
		wantTarget  string
	}{
		{
			name:       "default keeps canonical host",
			wantNames:  "api.example.com",
			wantTarget: "api.example.com",
		},
		{
			name:        "apex and wildcard preserve requested host",
			serverNames: []string{"example.com", "*.example.com"},
			ws:          true,
			wantNames:   "example.com *.example.com",
			wantTarget:  "$host",
		},
		{
			name:        "explicit single name preserves requested host",
			serverNames: []string{"example.com"},
			wantNames:   "example.com",
			wantTarget:  "$host",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := renderProxyTemplate("api.example.com", tc.serverNames, "127.0.0.2", 1323, tc.ws)
			if err != nil {
				t.Fatal(err)
			}
			got := string(config)
			if count := strings.Count(got, "server_name "+tc.wantNames+";"); count != 2 {
				t.Fatalf("expected configured names in both blocks, got %d:\n%s", count, got)
			}
			for _, want := range []string{
				"return 301 https://" + tc.wantTarget + "$request_uri;",
				"ssl_certificate /etc/letsencrypt/live/api.example.com/fullchain.pem;",
				"ssl_certificate_key /etc/letsencrypt/live/api.example.com/privkey.pem;",
				"access_log /var/log/api.example.com/access_log;",
				"proxy_pass http://127.0.0.2:1323/;",
				"proxy_set_header Host $host;",
			} {
				if !strings.Contains(got, want) {
					t.Errorf("missing %q:\n%s", want, got)
				}
			}
			if strings.Contains(got, "proxy_set_header Upgrade $http_upgrade;") != tc.ws {
				t.Error("WebSocket support does not match --ws")
			}
		})
	}
}

func TestServerNameFlag(t *testing.T) {
	app := newApp()
	want := []string{"example.com", "*.example.com"}
	called := false
	app.Action = func(ctx *cli.Context) error {
		called = true
		if got := ctx.StringSlice("server-name"); !reflect.DeepEqual(got, want) {
			t.Errorf("server names = %v, want %v", got, want)
		}
		if got := ctx.String("host"); got != "api.example.com" {
			t.Errorf("host = %q, want unchanged installation host", got)
		}
		return nil
	}
	if err := app.Run([]string{"instllr", "--host", "api.example.com", "--server-name", want[0], "--server-name", want[1], "--port", "1323", "install", "owner/repo"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("CLI action was not called")
	}
}

func TestInvalidServerNamesFailBeforeInstall(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"missing host", []string{"--name", "existing-service", "--server-name", "example.com"}, "--server-name requires --host"},
		{"empty name", []string{"--host", "api.example.com", "--server-name", ""}, "invalid server name"},
		{"multiple names in one value", []string{"--host", "api.example.com", "--server-name", "example.com *.example.com"}, "invalid server name"},
		{"nginx directive", []string{"--host", "api.example.com", "--server-name", "example.com; return 200"}, "invalid server name"},
		{"newline", []string{"--host", "api.example.com", "--server-name", "example.com\n"}, "invalid server name"},
		{"bare wildcard", []string{"--host", "api.example.com", "--server-name", "*"}, "invalid server name"},
		{"empty wildcard", []string{"--host", "api.example.com", "--server-name", "*."}, "invalid server name"},
		{"embedded wildcard", []string{"--host", "api.example.com", "--server-name", "clinic.*.example.com"}, "invalid server name"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"instllr"}, tc.args...)
			args = append(args, "--port", "1323", "install", "owner/repo")
			app := newApp()
			app.Action = func(*cli.Context) error {
				t.Fatal("invalid server names reached the install action")
				return nil
			}
			err := app.Run(args)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q before fetching a release or changing services", err, tc.want)
			}
		})
	}
}
