package resolver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dev-dami/carv/pkg/module"
	"github.com/dev-dami/carv/pkg/semver"
)

// Resolver handles transitive dependency resolution.
type Resolver struct {
	rootDir    string
	modulesDir string
	locked     map[string]module.LockedPackage
	resolved   map[string]*ResolvedDep
	visiting   map[string]bool // cycle detection
}

// ResolvedDep represents a fully resolved dependency.
type ResolvedDep struct {
	Name     string
	Version  string
	Source   string
	Revision string
	Deps     []*ResolvedDep
}

// New creates a new Resolver for the given project root.
func New(rootDir string) *Resolver {
	return &Resolver{
		rootDir:    rootDir,
		modulesDir: filepath.Join(rootDir, "carv_modules"),
		locked:     make(map[string]module.LockedPackage),
		resolved:   make(map[string]*ResolvedDep),
		visiting:   make(map[string]bool),
	}
}

// Resolve resolves all dependencies from the given config, including transitive deps.
func (r *Resolver) Resolve(cfg *module.Config) ([]*ResolvedDep, error) {
	// Load existing lock file for reproducible installs.
	lf, err := module.LoadLock(r.rootDir)
	if err == nil {
		for _, p := range lf.Packages {
			r.locked[p.Name] = p
		}
	}

	var roots []*ResolvedDep
	for name, dep := range cfg.Dependencies {
		resolved, err := r.resolveOne(name, dep, 0)
		if err != nil {
			return nil, fmt.Errorf("resolving %q: %w", name, err)
		}
		roots = append(roots, resolved)
	}
	return roots, nil
}

func (r *Resolver) resolveOne(name string, dep module.Dependency, depth int) (*ResolvedDep, error) {
	if depth > 50 {
		return nil, fmt.Errorf("dependency resolution exceeded max depth (possible cycle)")
	}

	// Cycle detection.
	if r.visiting[name] {
		return nil, fmt.Errorf("circular dependency detected: %s", name)
	}
	r.visiting[name] = true
	defer delete(r.visiting, name)

	// Check if already resolved.
	if existing, ok := r.resolved[name]; ok {
		return existing, nil
	}

	var resolved *ResolvedDep
	var err error

	switch {
	case dep.Git != "":
		resolved, err = r.resolveGit(name, dep)
	case dep.Path != "":
		resolved, err = r.resolvePath(name, dep)
	default:
		return nil, fmt.Errorf("no source specified for %q", name)
	}
	if err != nil {
		return nil, err
	}

	r.resolved[name] = resolved
	return resolved, nil
}

func (r *Resolver) resolveGit(name string, dep module.Dependency) (*ResolvedDep, error) {
	destDir := filepath.Join(r.modulesDir, name)

	// Check if already cloned.
	alreadyCloned := false
	if _, err := os.Stat(destDir); err == nil {
		alreadyCloned = true
	}

	// Determine which ref to checkout.
	var ref string
	var version string

	if dep.Tag != "" {
		ref = dep.Tag
		version = strings.TrimPrefix(dep.Tag, "v")
	} else if dep.Branch != "" {
		ref = dep.Branch
		version = "branch:" + dep.Branch
	} else if dep.Version != "" {
		// Resolve version constraint against available tags.
		bestTag, err := r.resolveVersionConstraint(dep.Git, dep.Version)
		if err != nil {
			return nil, err
		}
		ref = bestTag
		version = strings.TrimPrefix(bestTag, "v")
	} else {
		// No version specified, use latest tag or default branch.
		latestTag, err := r.latestTag(dep.Git)
		if err != nil {
			ref = "" // use default branch
			version = "latest"
		} else {
			ref = latestTag
			version = strings.TrimPrefix(latestTag, "v")
		}
	}

	// Clone or update.
	if !alreadyCloned {
		if err := r.gitClone(dep.Git, ref, destDir); err != nil {
			return nil, err
		}
	} else if ref != "" {
		// Checkout the specific ref.
		if err := r.gitCheckout(destDir, ref); err != nil {
			return nil, err
		}
	}

	// Get revision.
	revision := r.gitHead(destDir)

	// Read transitive dependencies.
	transDeps, err := r.readTransitiveDeps(destDir, name)
	if err != nil {
		return nil, err
	}

	return &ResolvedDep{
		Name:     name,
		Version:  version,
		Source:   "git+" + dep.Git,
		Revision: revision,
		Deps:     transDeps,
	}, nil
}

func (r *Resolver) resolvePath(name string, dep module.Dependency) (*ResolvedDep, error) {
	srcPath := dep.Path
	if !filepath.IsAbs(srcPath) {
		srcPath = filepath.Join(r.rootDir, srcPath)
	}

	destDir := filepath.Join(r.modulesDir, name)

	_ = os.RemoveAll(destDir)

	if err := os.Symlink(srcPath, destDir); err != nil {
		if err := copyDir(srcPath, destDir); err != nil {
			return nil, fmt.Errorf("failed to link/copy %q: %w", srcPath, err)
		}
	}

	transDeps, err := r.readTransitiveDeps(destDir, name)
	if err != nil {
		return nil, err
	}

	return &ResolvedDep{
		Name:    name,
		Version: dep.Version,
		Source:  "path+" + dep.Path,
		Deps:    transDeps,
	}, nil
}

func (r *Resolver) readTransitiveDeps(pkgDir string, parentName string) ([]*ResolvedDep, error) {
	cfg, err := module.LoadConfig(pkgDir)
	if err != nil {
		return nil, fmt.Errorf("reading %q carv.toml: %w", parentName, err)
	}
	if cfg == nil || len(cfg.Dependencies) == 0 {
		return nil, nil
	}

	var deps []*ResolvedDep
	for name, dep := range cfg.Dependencies {
		resolved, err := r.resolveOne(name, dep, 1)
		if err != nil {
			return nil, err
		}
		deps = append(deps, resolved)
	}
	return deps, nil
}

func (r *Resolver) resolveVersionConstraint(gitURL, constraintStr string) (string, error) {
	cs, err := semver.ParseConstraints(constraintStr)
	if err != nil {
		return "", err
	}

	tags, err := r.gitTags(gitURL)
	if err != nil {
		return "", fmt.Errorf("fetching tags from %s: %w", gitURL, err)
	}

	// Sort tags by semver, highest first.
	type tagVer struct {
		tag string
		ver semver.Version
	}
	var valid []tagVer
	for _, tag := range tags {
		v, err := semver.ParseVersion(tag)
		if err != nil {
			continue
		}
		if cs.Matches(v) {
			valid = append(valid, tagVer{tag: tag, ver: v})
		}
	}

	if len(valid) == 0 {
		return "", fmt.Errorf("no tags in %s satisfy constraint %q", gitURL, constraintStr)
	}

	sort.Slice(valid, func(i, j int) bool {
		return valid[i].ver.Compare(valid[j].ver) > 0
	})

	return valid[0].tag, nil
}

func (r *Resolver) latestTag(gitURL string) (string, error) {
	tags, err := r.gitTags(gitURL)
	if err != nil {
		return "", err
	}

	type tagVer struct {
		tag string
		ver semver.Version
	}
	var parsed []tagVer
	for _, tag := range tags {
		v, err := semver.ParseVersion(tag)
		if err != nil {
			continue
		}
		parsed = append(parsed, tagVer{tag: tag, ver: v})
	}

	if len(parsed) == 0 {
		return "", fmt.Errorf("no semver tags found")
	}

	sort.Slice(parsed, func(i, j int) bool {
		return parsed[i].ver.Compare(parsed[j].ver) > 0
	})

	return parsed[0].tag, nil
}

// Git operations.

func (r *Resolver) gitClone(url, ref, dest string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH")
	}

	args := []string{"clone"}
	if ref != "" {
		args = append(args, "--branch", ref, "--single-branch")
	}
	args = append(args, "--depth", "1", url, dest)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *Resolver) gitCheckout(dir, ref string) error {
	cmd := exec.Command("git", "-C", dir, "fetch", "--depth", "1", "origin", ref)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()

	cmd = exec.Command("git", "-C", dir, "checkout", ref)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (r *Resolver) gitHead(dir string) string {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (r *Resolver) gitTags(url string) ([]string, error) {
	cmd := exec.Command("git", "ls-remote", "--tags", "--refs", url)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		ref := parts[1]
		// refs/tags/v1.0.0 -> v1.0.0
		tag := strings.TrimPrefix(ref, "refs/tags/")
		// Skip annotated tags (refs/tags/v1.0.0^{})
		if strings.HasSuffix(tag, "^{}") {
			continue
		}
		tags = append(tags, tag)
	}

	return tags, nil
}

// LockFile generates a lock file from resolved dependencies.
func (r *Resolver) LockFile() *module.LockFile {
	var packages []module.LockedPackage
	r.collectLocked(r.resolved, &packages)
	return &module.LockFile{Packages: packages}
}

func (r *Resolver) collectLocked(resolved map[string]*ResolvedDep, out *[]module.LockedPackage) {
	for _, dep := range resolved {
		*out = append(*out, module.LockedPackage{
			Name:     dep.Name,
			Version:  dep.Version,
			Source:   dep.Source,
			Revision: dep.Revision,
		})
		r.collectLockedFlat(dep.Deps, out)
	}
}

func (r *Resolver) collectLockedFlat(deps []*ResolvedDep, out *[]module.LockedPackage) {
	for _, dep := range deps {
		*out = append(*out, module.LockedPackage{
			Name:     dep.Name,
			Version:  dep.Version,
			Source:   dep.Source,
			Revision: dep.Revision,
		})
		r.collectLockedFlat(dep.Deps, out)
	}
}

// PrintTree prints the resolved dependency tree.
func PrintTree(deps []*ResolvedDep, prefix string) {
	for _, dep := range deps {
		fmt.Printf("%s%s %s (%s)\n", prefix, dep.Name, dep.Version, dep.Source)
		if len(dep.Deps) > 0 {
			PrintTree(dep.Deps, prefix+"  ")
		}
	}
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(targetPath, data, 0o644)
	})
}
