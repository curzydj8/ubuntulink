// Package web embeds static web assets served by the agent.
package web

import _ "embed"

// DashboardHTML is the embedded machine-info dashboard page.
//
//go:embed dashboard.html
var DashboardHTML []byte
