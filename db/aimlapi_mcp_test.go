package db

import (
	"encoding/json"
	"net/url"
	"testing"
)

func readAimlapiMCP(t *testing.T, d *DB) (transport, rawURL string, env map[string]string, enabled bool, found bool) {
	t.Helper()
	var envRaw []byte
	err := d.QueryRow(`SELECT transport, COALESCE(url,''), env, enabled FROM mcp_servers WHERE name=$1`, aimlapiMCPName).
		Scan(&transport, &rawURL, &envRaw, &enabled)
	if err != nil {
		return "", "", nil, false, false
	}
	if err := json.Unmarshal(envRaw, &env); err != nil {
		t.Fatal(err)
	}
	return transport, rawURL, env, enabled, true
}

func TestSeedAimlapiMCP(t *testing.T) {
	d, err := Open(testDSN(t))
	if err != nil {
		SkipPGUnavailable(t, err)
	}
	origSetting, origHad, err := d.GetSetting(aimlapiMCPSeededSetting)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupExec(t, d, `DELETE FROM mcp_servers WHERE name=$1`, aimlapiMCPName)
		cleanupExec(t, d, `INSERT INTO mcp_servers(name, transport, url, env, enabled)
VALUES ($1, 'http', $2, '{"Authorization":""}', false) ON CONFLICT (name) DO NOTHING`, aimlapiMCPName, aimlapiMCPURL)
		if origHad {
			cleanupErr(t, "restore setting", d.SetSetting(aimlapiMCPSeededSetting, origSetting))
		} else {
			cleanupExec(t, d, `DELETE FROM settings WHERE key=$1`, aimlapiMCPSeededSetting)
		}
		d.Close()
	})

	reset := func(t *testing.T) {
		t.Helper()
		cleanupExec(t, d, `DELETE FROM mcp_servers WHERE name=$1`, aimlapiMCPName)
		cleanupExec(t, d, `DELETE FROM settings WHERE key=$1`, aimlapiMCPSeededSetting)
	}

	t.Run("seeded disabled with exact host and no key", func(t *testing.T) {
		reset(t)
		if err := d.seedBuiltins(); err != nil {
			t.Fatal(err)
		}
		transport, rawURL, env, enabled, found := readAimlapiMCP(t, d)
		if !found {
			t.Fatal("aimlapi mcp row not seeded")
		}
		if enabled {
			t.Fatal("aimlapi mcp must be disabled by default")
		}
		if transport != "http" {
			t.Fatalf("transport=%q, want http", transport)
		}
		u, err := url.Parse(rawURL)
		if err != nil {
			t.Fatal(err)
		}
		if u.Scheme != "https" || u.Host != "mcp.aimlapi.com" || u.Path != "/mcp" {
			t.Fatalf("url=%q, want https://mcp.aimlapi.com/mcp", rawURL)
		}
		if len(env) != 1 || env["Authorization"] != "" {
			t.Fatalf("env=%v, want only an empty Authorization header", env)
		}
	})

	t.Run("user edits kept", func(t *testing.T) {
		reset(t)
		if _, err := d.Exec(`INSERT INTO mcp_servers(name, transport, url, env, enabled)
VALUES ($1, 'sse', 'https://example.test/sse', '{"X":"y"}', true)`, aimlapiMCPName); err != nil {
			t.Fatal(err)
		}
		if err := d.seedBuiltins(); err != nil {
			t.Fatal(err)
		}
		transport, rawURL, env, enabled, _ := readAimlapiMCP(t, d)
		if transport != "sse" || rawURL != "https://example.test/sse" || env["X"] != "y" || !enabled {
			t.Fatalf("user row clobbered: %q %q %v %v", transport, rawURL, env, enabled)
		}
	})

	t.Run("deleted preset not re-added", func(t *testing.T) {
		reset(t)
		if err := d.seedBuiltins(); err != nil {
			t.Fatal(err)
		}
		cleanupExec(t, d, `DELETE FROM mcp_servers WHERE name=$1`, aimlapiMCPName)
		if err := d.seedBuiltins(); err != nil {
			t.Fatal(err)
		}
		if _, _, _, _, found := readAimlapiMCP(t, d); found {
			t.Fatal("deleted aimlapi mcp row came back")
		}
	})
}
