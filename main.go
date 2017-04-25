/*
vanity: static vanity index generator for S3

See -help for usage information.

Example

The following generates a listing for an entire vanity domain,
"vanity.example.com" where the source code is hosted under a specific
GitHub user:

	go list vanity.example.com/... | vanity -replace vanity.example.com=github.com/actual-user -o .
*/
package main // import "whitehouse.id.au/vanity"

import (
	"bufio"
	"flag"
	"fmt"
	"go/build"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/vcs"
)

var (
	replacerFlag replacerValue
	outputFlag   string
)

func init() {
	flag.Var(&replacerFlag, "replace", "a comma-separated list of canonical=noncanonical pairs of package paths")
	flag.StringVar(&outputFlag, "o", "", "base directory where HTML files should be created")
}

func main() {
	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	// Packages are either read as extra arguments or one line at
	// a time from standard input.
	var reader io.Reader
	if flag.NArg() > 0 {
		reader = strings.NewReader(strings.Join(flag.Args(), "\n"))
	} else {
		reader = os.Stdin
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		pkg, err := load(scanner.Text())
		exitOnErr(err)

		err = writePackageIndex(pkg)
		exitOnErr(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: %s [options] [packages]\n", os.Args[0])
	flag.PrintDefaults()
}

func exitOnErr(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func writePackageIndex(pkg *build.Package) error {
	// Determine the base package that contains the VCS.
	root, err := vcsRoot(pkg)
	if err != nil {
		return err
	}

	// Open an output for writing the HTML template.
	w, err := open(pkg.ImportPath)
	if err != nil {
		return err
	}
	defer w.Close()

	// Generate a HTML file with meta tags for each.
	data := struct {
		ImportPath string
		VCS        GitHub
	}{
		ImportPath: pkg.ImportPath,
		// FIXME: This currently only supports GitHub VCS endpoints.
		VCS: GitHub{
			ImportPath: root,
			Repository: replacerFlag.Replace(root),
		},
	}
	return indexTpl.Execute(w, data)
}

func open(importPath string) (io.WriteCloser, error) {
	// Write to console by default, unless a path is specified.
	if outputFlag == "" {
		return NopCloser(os.Stdout), nil
	}

	// Ensure the directory tree exists.
	dir := filepath.Join(outputFlag, importPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return os.Create(filepath.Join(dir, "index.html"))
}

// packages loads package information for each argument.
func load(name string) (*build.Package, error) {
	return build.Import(name, ".", 0)
}

// vcsRoot returns the import path of the package VCS.
func vcsRoot(pkg *build.Package) (string, error) {
	dir := pkg.Dir
	for dir != pkg.SrcRoot {
		_, err := vcs.DetectVcsFromFS(dir)

		// We found a parent package that has a repository.
		if err == nil {
			break
		}

		if err == vcs.ErrCannotDetectVCS {
			dir = filepath.Dir(dir)
			continue
		}

		return "", err
	}

	// Convert directory back to an import path.
	rel, err := filepath.Rel(pkg.SrcRoot, dir)
	if err != nil {
		return "", err
	}

	// If import path is relative to the current directy, the
	// original package _is_ the VCS root.
	if rel == "." {
		return pkg.ImportPath, nil
	}
	return rel, nil
}

var indexTpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
<head>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8"/>
<meta name="go-import" content="{{ .VCS.GoImport }}">
<meta name="go-source" content="{{ .VCS.GoSource }}">
<meta http-equiv="refresh" content="0; url=https://godoc.org/{{ .ImportPath }}">
</head>
<body>
Nothing to see here; <a href="https://godoc.org/{{ .ImportPath }}">move along</a>.
</body>
</html>
`))

type replacerValue struct {
	*strings.Replacer
}

func (v *replacerValue) Set(str string) error {
	// Flatten comma-separated list of old=new pairs into a list.
	var oldnew []string
	for _, pair := range strings.Split(str, ",") {
		oldnew = append(oldnew, strings.Split(pair, "=")...)
	}

	v.Replacer = strings.NewReplacer(oldnew...)
	return nil
}

func (v *replacerValue) String() string {
	return "<replacer>"
}

// GitHub produces Golang import and source URLs suitable for GitHub.
type GitHub struct {
	ImportPath string
	Repository string
}

// GoImport produces go-import meta tag content for GitHub.
//
// See: https://golang.org/cmd/go/#hdr-Remote_import_paths
func (g GitHub) GoImport() string {
	return fmt.Sprintf("%s git https://%s.git", g.ImportPath, g.Repository)
}

// GoSource produces go-source meta tag content for GitHub.
//
// See: https://github.com/golang/gddo/wiki/Source-Code-Links
func (g GitHub) GoSource() string {
	return fmt.Sprintf("%s _ %s %s",
		g.ImportPath,
		fmt.Sprintf("https://%s/blob/master{/dir}", g.Repository),
		fmt.Sprintf("https://%s/blob/master{/dir}/{file}#L{line}", g.Repository))
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

// NopCloser returns a ReadCloser with a no-op Close method wrapping
// the provided Writer w.
func NopCloser(w io.Writer) io.WriteCloser {
	return nopCloser{w}
}
