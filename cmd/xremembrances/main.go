package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		if err := buildCmd(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "xremembrances build failed: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: xremembrances build [--with module[@version]]... [--output path]")
}

func buildCmd(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var with stringSlice
	var output string
	var base string

	fs.Var(&with, "with", "Module to include (repeatable). Example: github.com/remembrances/tools-reasoning@v1.2.0")
	fs.StringVar(&output, "output", "./remembrances-mcp", "Output binary path")
	fs.StringVar(&base, "base", "github.com/madeindigio/remembrances-mcp", "Base module path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	outputPath := output
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(cwd, outputPath)
	}

	tmpDir, err := os.MkdirTemp("", "xremembrances-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := writeMain(tmpDir, base, with); err != nil {
		return err
	}

	if err := runCmd(tmpDir, "go", "mod", "init", "xremembrances-build"); err != nil {
		return err
	}

	baseImport, baseGet := splitModuleVersion(base)
	_ = baseImport
	if err := runCmd(tmpDir, "go", "get", baseGet); err != nil {
		return err
	}

	for _, mod := range with {
		_, modGet := splitModuleVersion(mod)
		if err := runCmd(tmpDir, "go", "get", modGet); err != nil {
			return err
		}
	}

	if err := runCmd(tmpDir, "go", "mod", "tidy"); err != nil {
		return err
	}

	if err := runCmd(tmpDir, "go", "build", "-o", outputPath); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Built %s\n", outputPath)
	return nil
}

func writeMain(dir, base string, modules []string) error {
	baseImport, _ := splitModuleVersion(base)

	var imports []string
	imports = append(imports, fmt.Sprintf("remembrances \"%s/cmd/remembrances-mcp\"", baseImport))
	imports = append(imports, fmt.Sprintf("_ \"%s/modules/standard\"", baseImport))
	for _, mod := range modules {
		importPath, _ := splitModuleVersion(mod)
		imports = append(imports, fmt.Sprintf("_ \"%s\"", importPath))
	}

	content := "package main\n\nimport (\n\t" + strings.Join(imports, "\n\t") + "\n)\n\nfunc main() {\n\tremembrances.Main()\n}\n"

	return os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644)
}

func splitModuleVersion(module string) (string, string) {
	parts := strings.SplitN(module, "@", 2)
	if len(parts) == 2 {
		return parts[0], module
	}
	return module, module
}

func runCmd(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
