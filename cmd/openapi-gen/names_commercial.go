//go:build commercial

package main

import (
	"sort"

	// The commercial modules are linked in so their routes are registered and
	// therefore documented. Importing them here is what makes the commercial
	// document describe the commercial binary.
	_ "github.com/madeindigio/remembrances-mcp/modules/commercial/db-sync-server"
	_ "github.com/madeindigio/remembrances-mcp/modules/commercial/webui"

	"github.com/madeindigio/remembrances-mcp/pkg/modules"
)

// The file names of the commercial document. It is a separate file rather than
// an edit of the core one because the two builds serve different surfaces and
// a single document would have to lie about one of them.
const (
	// OpenAPIFileName is the OpenAPI 3.1 document of this build.
	OpenAPIFileName = "remembrances-commercial.openapi.json"
	// PostmanFileName is the Postman collection of this build.
	PostmanFileName = "remembrances-commercial.postman_collection.json"
)

// sortedModules returns the registered modules in id order, so the generated
// documents do not depend on map iteration order.
func sortedModules() []modules.ModuleInfo {
	list := modules.ListModules()
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return list
}
