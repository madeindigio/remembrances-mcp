//go:build !commercial

package main

import (
	"sort"

	"github.com/madeindigio/remembrances-mcp/pkg/modules"
)

// The file names of the core document, the one a build with no module tags
// produces. It describes the routes every deployment has.
const (
	// OpenAPIFileName is the OpenAPI 3.1 document of this build.
	OpenAPIFileName = "remembrances.openapi.json"
	// PostmanFileName is the Postman collection of this build.
	PostmanFileName = "remembrances.postman_collection.json"
)

// sortedModules returns the registered modules in id order, so the generated
// documents do not depend on map iteration order.
func sortedModules() []modules.ModuleInfo {
	list := modules.ListModules()
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	return list
}
