Added YAML configuration support to Remembrances-MCP server. Key changes:

1. **CLI Flag**: Added `--config` flag to specify YAML configuration file path.

2. **Config Loading**: Modified `internal/config/config.go` to read YAML file using Viper before binding CLI flags and environment variables. YAML values can be overridden by CLI flags or env vars.

3. **Sample Config**: Created `config.sample.yaml` with all configuration options, their default values commented, and usage instructions.

4. **Documentation**: Updated README.md to include the new --config flag in CLI flags list and added a YAML Configuration section with examples.

The implementation uses Viper's standard YAML reading capabilities. Configuration precedence: YAML < Environment Variables < CLI flags.