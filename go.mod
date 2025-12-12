module github.com/madeindigio/remembrances-mcp

go 1.23.4

require (
	github.com/ThinkInAIXYZ/go-mcp v0.2.21
	github.com/agnivade/levenshtein v1.2.1
	github.com/ebitengine/purego v0.7.1
	github.com/fsnotify/fsnotify v1.9.0
	github.com/google/uuid v1.6.0
	github.com/madeindigio/surrealdb-embedded-golang v0.0.0-00010101000000-000000000000
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
	github.com/spf13/pflag v1.0.7
	github.com/spf13/viper v1.20.1
	github.com/surrealdb/surrealdb.go v1.0.0
	github.com/tmc/langchaingo v0.1.13
	github.com/toon-format/toon-go v0.0.0-20251202084852-7ca0e27c4e8c
)

require (
	github.com/dlclark/regexp2 v1.10.0 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/orcaman/concurrent-map/v2 v2.0.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.3 // indirect
	github.com/pkoukk/tiktoken-go v0.1.6 // indirect
	github.com/sagikazarmark/locafero v0.7.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.12.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Local development replacements
// These use absolute paths expanded at build time. For portable builds, create a go.work file:
//   go work init .
//   go work use $HOME/www/MCP/Remembrances/surrealdb-embedded
//
// Or set environment variables before building:
//   export SURREALDB_DIR=$HOME/www/MCP/Remembrances/surrealdb-embedded
//   go mod edit -replace github.com/madeindigio/surrealdb-embedded-golang=$SURREALDB_DIR

// Default paths for the main development environment
replace github.com/madeindigio/surrealdb-embedded-golang => /Users/digio/www/MCP/Remembrances/surrealdb-embedded
