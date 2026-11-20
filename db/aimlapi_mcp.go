package db

import "fmt"

// AI/ML API hosted MCP server (https://docs.aimlapi.com/quickstart/mcp): a remote
// Streamable HTTP endpoint. It is offered as a disabled preset row, like the
// browser MCP, so it shows up on the MCP page without being wired into any agent
// until the user pastes a key into the Authorization header and enables it.
const (
	aimlapiMCPName = "aimlapi"
	aimlapiMCPURL  = "https://mcp.aimlapi.com/mcp"
	// aimlapiMCPSeededSetting marks the preset as offered once, so a user who
	// deletes the row does not get it back on the next start.
	aimlapiMCPSeededSetting = "aimlapi_mcp_seeded_v1"
)

// seedAimlapiMCP inserts the AI/ML API MCP preset once: http transport, fixed
// URL, empty Authorization header (no key is stored), disabled. An existing row
// with the same name is never touched.
func (d *DB) seedAimlapiMCP() error {
	if v, _, _ := d.GetSetting(aimlapiMCPSeededSetting); v == "true" {
		return nil
	}
	if _, err := d.Exec(`
INSERT INTO mcp_servers(name, transport, url, env, enabled)
VALUES ($1, 'http', $2, '{"Authorization":""}', false)
ON CONFLICT (name) DO NOTHING`,
		aimlapiMCPName, aimlapiMCPURL); err != nil {
		return fmt.Errorf("seed aimlapi mcp: %w", err)
	}
	return d.SetSetting(aimlapiMCPSeededSetting, "true")
}
