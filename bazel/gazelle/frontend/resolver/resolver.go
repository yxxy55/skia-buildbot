package resolver

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bazelbuild/bazel-gazelle/config"
	"github.com/bazelbuild/bazel-gazelle/label"
	"github.com/bazelbuild/bazel-gazelle/repo"
	"github.com/bazelbuild/bazel-gazelle/resolve"
	"github.com/bazelbuild/bazel-gazelle/rule"
	"go.skia.org/infra/bazel/gazelle/frontend/common"
	"go.skia.org/infra/go/util"
)

const (
	// gazelleExtensionName is the extension name passed to Gazelle.
	//
	// This name can be used to enable or disable this Gazelle extension via the --lang flag, e.g.
	//
	//     $ bazel run gazelle -- update --lang go,frontend
	gazelleExtensionName = "frontend"

	// Bazel package with aliases to NPM modules.
	//
	// Under rules_nodejs, one can establish a dependency to a NPM module named "foo" via the
	// "@npm//foo" Bazel label. However, we are currently migrating to rules_js (b/314813928), which
	// exposes NPM modules to Bazel using a different naming convention. To ease the transition and
	// minimize diffs during reviews, we have created aliases for all NPM dependencies in
	// //npm_deps/BUILD.bazel so that we can express dependencies to the "foo" NPM module via
	// "//npm_deps:foo" rather than "@npm//foo".
	//
	// After the bulk of the migration is done, these aliase will be deleted and we will update this
	// Gazelle extension to generate dependencies using the naming convention native to rules_js.
	// Again, the goal of these aliases is to minimize diffs during reviews.
	npmDepsBazelPackage = "npm_deps"

	// packageJsonPath is the path to the package.json file used by the npm_install rule in the
	// workspace file. This path is relative to the workspace root directory.
	packageJsonPath = "package.json"
)

// Resolver implements the resolve.Resolver interface.
//
// Interface documentation:
//
// Resolver is an interface that language extensions can implement to resolve
// dependencies in rules they generate.
type Resolver struct {
	// sassImportsToDeps maps Sass imports to the rules that provide those imports.
	sassImportsToDeps map[string]map[ruleKindAndLabel]bool

	// tsImportsToDeps maps TypeScript imports to rules that provide those imports.
	tsImportsToDeps map[string]map[ruleKindAndLabel]bool

	// npmPackages is the set of NPM dependencies and devDependencies read from the package.json file.
	npmPackages map[string]bool
}

// ruleAKindAndLabel is a (rule kind, rule label) pair (e.g. "ts_library", "//path/to:my_ts_lib").
type ruleKindAndLabel struct {
	kind  string
	label label.Label
}

// noRuleKindAndLabel is the zero value of ruleKindAndLabel. Used as a sentinel value when no rule
// is found.
var noRuleKindAndLabel = ruleKindAndLabel{}

// indexImportsProvidedByRule indexes the imports provided by the given rule. The rule can be later
// obtained from an import via the findRuleThatProvidesImport method.
func (rslv *Resolver) indexImportsProvidedByRule(lang string, importPaths []string, ruleKind string, ruleLabel label.Label) {
	if lang != "sass" && lang != "ts" {
		log.Panicf("Unknown language: %q.", lang)
	}

	if rslv.sassImportsToDeps == nil {
		rslv.sassImportsToDeps = map[string]map[ruleKindAndLabel]bool{}
	}
	if rslv.tsImportsToDeps == nil {
		rslv.tsImportsToDeps = map[string]map[ruleKindAndLabel]bool{}
	}

	importsToDeps := rslv.sassImportsToDeps
	if lang == "ts" {
		importsToDeps = rslv.tsImportsToDeps
	}

	for _, importPath := range importPaths {
		if importsToDeps[importPath] == nil {
			importsToDeps[importPath] = map[ruleKindAndLabel]bool{}
		}
		rkal := ruleKindAndLabel{kind: ruleKind, label: ruleLabel}
		importsToDeps[importPath][rkal] = true
	}
}

// findRuleThatProvidesImport returns the rule that provides the given import, provided it was
// indexed via an earlier call to indexImportsProvidedByRule.
func (rslv *Resolver) findRuleThatProvidesImport(lang string, importPath string, fromRuleKind string, fromRuleLabel label.Label) ruleKindAndLabel {
	if lang != "sass" && lang != "ts" {
		log.Panicf("Unknown language: %q.", lang)
	}

	importsToDeps := rslv.sassImportsToDeps
	if lang == "ts" {
		importsToDeps = rslv.tsImportsToDeps
	}

	var candidates []ruleKindAndLabel
	if importsToDeps[importPath] != nil {
		for c := range importsToDeps[importPath] {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		gazelleIgnoreMsg := ""
		if lang == "ts" {
			gazelleIgnoreMsg = `; if this is expected, add "// gazelle:ignore" at the end of the import statement to make this warning go away`
		}
		log.Printf("Could not find any rules that satisfy import %q from %s (%s)%s", importPath, fromRuleLabel, fromRuleKind, gazelleIgnoreMsg)
		return noRuleKindAndLabel
	}

	if len(candidates) > 1 {
		log.Printf("Multiple rules satisfy import %q from %s (%s): %s (%s), %s (%s)", importPath, fromRuleLabel, fromRuleKind, candidates[0].label, candidates[0].kind, candidates[1].label, candidates[1].kind)
		return noRuleKindAndLabel
	}

	return candidates[0]
}

// Name implements the resolve.Resolver interface.
//
// Interface documentation:
//
// Name returns the name of the language. This should be a prefix of the
// kinds of rules generated by the language, e.g., "go" for the Go extension
// since it generates "go_library" rules.
func (rslv *Resolver) Name() string {
	return gazelleExtensionName
}

// Imports implements the resolve.Resolver interface.
//
// Imports extracts and indexes the imports provided by the given rule. Gazelle calls this method
// once for each rule in the repository that this Gazelle extension understands (i.e. all front-end
// rules).
//
// For example, if Imports is passed a ts_library rule with label "//path/to:my_lib" and sources
// "foo.ts" and "bar.ts", then presumably said rule could satisfy TypeScript imports such as
// "import * from 'path/to/foo'" or "import 'path/to/bar'". In this example, Imports should return
// a slice with two resolve.ImportSpec structs, one for each of "path/to/foo" and "path/to/bar".
//
// Gazelle uses the returned resolve.ImportSpec structs to build a resolve.RuleIndex struct, which
// maps imports (e.g. "path/to/foo") to the labels of the rules that provide them (e.g.
// "//path/to:my_lib"). This index is passed to the Resolve method (implemented below), in which we
// resolve the dependencies of all the rules generated by this Gazelle extension (i.e. we populate
// their deps attributes).
//
// However, the resolve.RuleIndex struct is insufficient to resolve the dependencies of rules such
// as sk_element, which has multiple types of dependencies (ts_deps, sass_deps, sk_element_deps).
// We need to know the kind of a dependency (e.g. "ts_library", "sass_library", "sk_element"), in
// addition to its label, before we can add it to one of the *_deps arguments, but the
// resolve.RuleIndex struct only provides the latter.
//
// For this reason, this Gazelle extension ignores the resolve.RuleIndex struct. Instead, we build
// our own index with all the information we need (see fields sassImportsToDeps and
// tsImportsToDeps).
//
// Therefore, this method always returns an empty slice, which results in an empty
// resolve.RuleIndex, but that is OK because we do not use it.
func (rslv *Resolver) Imports(c *config.Config, r *rule.Rule, f *rule.File) []resolve.ImportSpec {
	ruleLabel := label.New(c.RepoName, f.Pkg, r.Name())

	switch r.Kind() {
	case "ts_library":
		importPaths := extractTypeScriptImportsProvidedByRule(f.Pkg, r, "srcs")
		rslv.indexImportsProvidedByRule("ts", importPaths, r.Kind(), ruleLabel)
	case "sass_library":
		importPaths := extractSassImportsProvidedByRule(f.Pkg, r, "srcs")
		rslv.indexImportsProvidedByRule("sass", importPaths, r.Kind(), ruleLabel)
	case "sk_element":
		tsImportPaths := extractTypeScriptImportsProvidedByRule(f.Pkg, r, "ts_srcs")
		sassImportPaths := extractSassImportsProvidedByRule(f.Pkg, r, "sass_srcs")
		rslv.indexImportsProvidedByRule("ts", tsImportPaths, r.Kind(), ruleLabel)
		rslv.indexImportsProvidedByRule("sass", sassImportPaths, r.Kind(), ruleLabel)
	}

	return nil
}

// extractTypeScriptImportsProvidedByRule takes a rule with TypeScript sources (e.g. "ts_library",
// "sk_element", etc.) and returns the paths of the imports that the source files may satisfy.
func extractTypeScriptImportsProvidedByRule(pkg string, r *rule.Rule, srcsAttr string) []string {
	var importPaths []string
	for _, src := range r.AttrStrings(srcsAttr) {
		if !strings.HasSuffix(src, ".ts") {
			log.Printf("Rule %s of kind %s contains a non-TypeScript file in its %s attribute: %s", label.New("", pkg, r.Name()).String(), r.Kind(), srcsAttr, src)
			continue
		}

		importPaths = append(importPaths, path.Join(pkg, strings.TrimSuffix(src, path.Ext(src))))

		// An index.ts file may also be imported as its parent folder's "main" module:
		//
		//     // The two following imports are equivalent.
		//     import 'path/to/module/index';
		//     import 'path/to/module';
		//
		// Reference:
		// https://www.typescriptlang.org/docs/handbook/module-resolution.html#how-typescript-resolves-modules.
		if src == "index.ts" {
			importPaths = append(importPaths, pkg)
		}
	}
	return importPaths
}

// extractTypeScriptImportsProvidedByRule takes a rule with Sass sources (e.g. "sass_library",
// "sk_element", etc.) and returns the paths of the imports that the source files may satisfy.
func extractSassImportsProvidedByRule(pkg string, r *rule.Rule, srcsAttr string) []string {
	var importPaths []string
	for _, src := range r.AttrStrings(srcsAttr) {
		if !strings.HasSuffix(src, ".scss") && !strings.HasSuffix(src, ".css") {
			log.Printf("Rule %s of kind %s contains a non-Sass file in its %s attribute: %s", label.New("", pkg, r.Name()).String(), r.Kind(), srcsAttr, src)
			continue
		}
		importPaths = append(importPaths, path.Join(pkg, strings.TrimSuffix(src, path.Ext(src))))
	}
	return importPaths
}

// Embeds implements the resolve.Resolver interface.
func (rslv *Resolver) Embeds(*rule.Rule, label.Label) []label.Label { return nil }

// Resolve implements the resolve.Resolver interface.
//
// Resolve takes a (rule, ImportsParsedFromRuleSources) pair generated by Language.GenerateRules()
// and resolves the rule's dependencies based on its imports. It populates the rule's deps argument
// (or ts_deps, sass_deps and sk_element_deps arguments in the case of sk_element and sk_page rules)
// with the result of mapping each import to the label of a rule that provides the import. It does
// so by leveraging the index built in the Imports method.
//
// Gazelle calls this method once for each (rule, ImportsParsedFromRuleSources) pair generated by
// successive calls to Language.GenerateRules(). Gazelle calls this method after all imports in the
// repository have been indexed via successive calls to the Imports method.
func (rslv *Resolver) Resolve(c *config.Config, _ *resolve.RuleIndex, _ *repo.RemoteCache, r *rule.Rule, imports interface{}, from label.Label) {
	importsFromRuleSources := imports.(common.ImportsParsedFromRuleSources)

	switch r.Kind() {
	case "karma_test":
		fallthrough
	case "nodejs_test":
		fallthrough
	case "sk_element_puppeteer_test":
		fallthrough
	case "ts_library":
		var deps []label.Label
		for _, importPath := range importsFromRuleSources.GetTypeScriptImports() {
			for _, ruleKindAndLabel := range rslv.resolveDepsForTypeScriptImport(r.Kind(), from, importPath, c.RepoRoot) {
				deps = append(deps, ruleKindAndLabel.label)
			}
		}
		setDeps(r, from, "deps", deps)

	case "sass_library":
		var deps []label.Label
		for _, importPath := range importsFromRuleSources.GetSassImports() {
			ruleKindAndLabel := rslv.resolveDepForSassImport(r.Kind(), from, importPath)
			if ruleKindAndLabel == noRuleKindAndLabel {
				continue // No rule satisfies the current Sass import. A warning has already been logged.
			}
			dep := ruleKindAndLabel.label
			if ruleKindAndLabel.kind == "sk_element" {
				// Ensure that the target name is explicit ("//my/package:package" vs "//my/package") before
				// appending the known suffix for the sass_library target generated by the sk_element macro.
				if dep.Name == "" {
					dep.Name = dep.Pkg
				}
				dep.Name = dep.Name + "_styles"
			}
			deps = append(deps, dep)
		}
		setDeps(r, from, "deps", deps)

	case "sk_element":
		fallthrough
	case "sk_page":
		var skElementDeps, tsDeps, sassDeps []label.Label
		for _, importPath := range importsFromRuleSources.GetTypeScriptImports() {
			for _, ruleKindAndLabel := range rslv.resolveDepsForTypeScriptImport(r.Kind(), from, importPath, c.RepoRoot) {
				if ruleKindAndLabel.kind == "sk_element" {
					skElementDeps = append(skElementDeps, ruleKindAndLabel.label)
				} else {
					tsDeps = append(tsDeps, ruleKindAndLabel.label)
				}
			}
		}
		for _, importPath := range importsFromRuleSources.GetSassImports() {
			ruleKindAndLabel := rslv.resolveDepForSassImport(r.Kind(), from, importPath)
			if ruleKindAndLabel == noRuleKindAndLabel {
				continue // No rule satisfies the current Sass import. A warning has already been logged.
			}
			if ruleKindAndLabel.kind == "sk_element" {
				skElementDeps = append(skElementDeps, ruleKindAndLabel.label)
			} else {
				sassDeps = append(sassDeps, ruleKindAndLabel.label)
			}
		}
		setDeps(r, from, "sk_element_deps", skElementDeps)
		setDeps(r, from, "ts_deps", tsDeps)
		setDeps(r, from, "sass_deps", sassDeps)
	}
}

// setDeps sets the dependencies of a rule.
func setDeps(r *rule.Rule, l label.Label, depsAttr string, deps []label.Label) {
	r.DelAttr(depsAttr)

	var depsAsStrings []string
	for _, dep := range deps {
		dep = dep.Rel(l.Repo, l.Pkg)
		// Filter out self-imports (e.g. when an sk_element has files index.ts and foo-sk.ts, and file
		// foo-sk.ts is imported from index.ts).
		if dep.Relative && dep.Name == r.Name() {
			continue
		}
		depsAsStrings = append(depsAsStrings, dep.String())
	}

	if len(depsAsStrings) > 0 {
		depsAsStrings = util.SSliceDedup(depsAsStrings)
		sort.Strings(depsAsStrings)
		r.SetAttr(depsAttr, depsAsStrings)
	}
}

// resolveDepForSassImport returns the label of the rule that resolves the given Sass import.
//
// Due to the way rules_js works, we do not support Sass and CSS imports directly from NPM.
// Instead, please use the copy_file_from_npm_pkg Bazel macro to create a local copy of those
// files, then import them as if they were regular source files.
//
// Note that any such files added to a sass_library's "srcs" attribute or to a sk_element's
// "sass_srcs" attribute should include a "# keep" comment, or Gazelle will delete them. Example:
//
//     sass_library(
//         name = "my_lib",
//         srcs = [
//             "my_lib.scss",
//             "stylesheet_copied_from_npm.css",  # keep
//         ],
//     )
//
// For details, please see the copy_file_from_npm_pkg macro docstring.

func (rslv *Resolver) resolveDepForSassImport(ruleKind string, ruleLabel label.Label, importPath string) ruleKindAndLabel {
	// Sass always resolves imports relative to the current file first, so we normalize the import
	// path relative to the current directory, e.g. "../bar" imported from "myapp/foo" becomes
	// "myapp/bar".
	//
	// Reference:
	// https://sass-lang.com/documentation/at-rules/use#load-paths
	// https://sass-lang.com/documentation/at-rules/import#load-paths
	normalizedImportPath := path.Join(ruleLabel.Pkg, strings.TrimSuffix(importPath, path.Ext(importPath)))

	return rslv.findRuleThatProvidesImport("sass", normalizedImportPath, ruleKind, ruleLabel)
}

// resolveDepsForTypeScriptImport returns the labels of the rules that resolve the given TypeScript
// import.
//
// If the import refers to an NPM package with a separate types declaration (e.g. "foo" and
// "@types/foo"), the labels for both dependencies will be returned.
func (rslv *Resolver) resolveDepsForTypeScriptImport(ruleKind string, ruleLabel label.Label, importPath string, repoRootDir string) []ruleKindAndLabel {
	// Is this an import of another source file in the repository?
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		// Normalize the import path, e.g. "../bar" imported from "myapp/foo" becomes "myapp/bar".
		normalizedImportPath := path.Join(ruleLabel.Pkg, importPath)

		rkal := rslv.findRuleThatProvidesImport("ts", normalizedImportPath, ruleKind, ruleLabel)
		if rkal == noRuleKindAndLabel {
			return nil
		}
		return []ruleKindAndLabel{rkal}
	}

	// The import must be either an NPM package or a built-in Node.js module.
	var moduleScope, moduleName, fullyQualifiedModuleName string
	if strings.HasPrefix(importPath, "@") {
		parts := strings.Split(importPath, "/")
		moduleScope = parts[0]                                    // e.g. @scope/my-module/foo/bar => @scope
		moduleName = parts[1]                                     // e.g. @scope/my-module/foo/bar => my-module
		fullyQualifiedModuleName = moduleScope + "/" + moduleName // e.g. @scope/my-module/foo/bar => @scope/my-module
	} else {
		moduleName = strings.Split(importPath, "/")[0] // e.g. my-module/foo/bar => my-module
		fullyQualifiedModuleName = moduleName
	}

	// Is this an import from an NPM package?
	if npmPackages := rslv.getNPMPackages(filepath.Join(repoRootDir, packageJsonPath)); npmPackages[fullyQualifiedModuleName] {
		var rkals []ruleKindAndLabel
		// Add as dependencies both the module and its type annotations package, if it exists.
		rkals = append(rkals, ruleKindAndLabel{
			kind:  "",                                                           // This dependency is not a rule (e.g. ts_library), so we leave the rule kind blank.
			label: label.New("", npmDepsBazelPackage, fullyQualifiedModuleName), // e.g. //npm_deps:puppeteer
		})

		// We assume that scoped packages (e.g. @google-web-components/google-chart) include type
		// annotations. If this ceases to be true, we will have to update the below ruleKindAndLabel.
		if moduleScope == "" {
			typesModuleName := "@types/" + moduleName // e.g. @types/my-module
			if npmPackages[typesModuleName] {
				rkals = append(rkals, ruleKindAndLabel{
					kind:  "",                                                  // This dependency is not a rule (e.g. ts_library), so we leave the rule kind blank.
					label: label.New("", npmDepsBazelPackage, typesModuleName), // e.g. //npm_deps:@types/puppeteer
				})
			}
		}

		return rkals
	}

	// Is this a built-in Node.js module?
	if builtInNodeJSModules[moduleName] {
		// Nothing to do - no need to add built-in modules as explicit dependencies.
		return nil
	}

	log.Printf("Unable to resolve import %q from %s (%s): no %q NPM package or built-in module found.", importPath, ruleLabel, ruleKind, moduleName)
	return nil
}

// getNPMPackages returns the set of NPM dependencies found in the package.json file.
func (rslv *Resolver) getNPMPackages(path string) map[string]bool {
	if rslv.npmPackages != nil {
		return rslv.npmPackages
	}

	var packageJSON struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	// Read in and unmarshall package.json file.
	b, err := os.ReadFile(path)
	if err != nil {
		log.Panicf("Error reading file %q: %v", path, err)
	}
	if err := json.Unmarshal(b, &packageJSON); err != nil {
		log.Panicf("Error parsing %s: %v", path, err)
	}

	// Extract all NPM packages found in the package.json file.
	rslv.npmPackages = map[string]bool{}
	for pkg := range packageJSON.Dependencies {
		rslv.npmPackages[pkg] = true
	}
	for pkg := range packageJSON.DevDependencies {
		rslv.npmPackages[pkg] = true
	}

	return rslv.npmPackages
}

// builtInNodeJSModules is a set of built-in Node.js modules.
//
// This set can be regenerated via the following command:
//
//	$ echo "require('module').builtinModules.forEach(m => console.log(m))" | nodejs
//
// See https://nodejs.org/api/module.html#module_module_builtinmodules.
var builtInNodeJSModules = map[string]bool{
	"_http_agent":         true,
	"_http_client":        true,
	"_http_common":        true,
	"_http_incoming":      true,
	"_http_outgoing":      true,
	"_http_server":        true,
	"_stream_duplex":      true,
	"_stream_passthrough": true,
	"_stream_readable":    true,
	"_stream_transform":   true,
	"_stream_wrap":        true,
	"_stream_writable":    true,
	"_tls_common":         true,
	"_tls_wrap":           true,
	"assert":              true,
	"async_hooks":         true,
	"buffer":              true,
	"child_process":       true,
	"cluster":             true,
	"console":             true,
	"constants":           true,
	"crypto":              true,
	"dgram":               true,
	"dns":                 true,
	"domain":              true,
	"events":              true,
	"fs":                  true,
	"http":                true,
	"http2":               true,
	"https":               true,
	"inspector":           true,
	"module":              true,
	"net":                 true,
	"os":                  true,
	"path":                true,
	"perf_hooks":          true,
	"process":             true,
	"punycode":            true,
	"querystring":         true,
	"readline":            true,
	"repl":                true,
	"stream":              true,
	"string_decoder":      true,
	"sys":                 true,
	"timers":              true,
	"tls":                 true,
	"trace_events":        true,
	"tty":                 true,
	"url":                 true,
	"util":                true,
	"v8":                  true,
	"vm":                  true,
	"wasi":                true,
	"worker_threads":      true,
	"zlib":                true,
}

var _ resolve.Resolver = &Resolver{}
