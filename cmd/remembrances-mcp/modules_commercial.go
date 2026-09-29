//go:build commercial

package main

import (
	_ "github.com/madeindigio/remembrances-mcp/v2/modules/commercial/db-sync-server"
	_ "github.com/madeindigio/remembrances-mcp/v2/modules/commercial/webui"
)
