package module

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dev-dami/carv/pkg/ast"
	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
)

type Loader struct {
	basePath    string
	loadedFiles map[string]*Module
	config      *Config
}

type Module struct {
	Path      string
	Program   *ast.Program
	Exports   map[string]bool
	IsBuiltin bool
}

func NewLoader(basePath string) *Loader {
	absPath, _ := filepath.Abs(basePath)
	return &Loader{
		basePath:    absPath,
		loadedFiles: make(map[string]*Module),
	}
}

func (l *Loader) SetConfig(cfg *Config) {
	l.config = cfg
}

func (l *Loader) Load(importPath string, fromFile string) (*Module, error) {
	resolved, err := l.resolvePath(importPath, fromFile)
	if err != nil {
		return nil, err
	}

	if mod, ok := l.loadedFiles[resolved]; ok {
		return mod, nil
	}

	content, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}

	lex := lexer.New(string(content))
	p := parser.New(lex)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		return nil, &ParseError{Path: resolved, Errors: p.Errors()}
	}

	mod := &Module{
		Path:    resolved,
		Program: program,
		Exports: l.extractExports(program),
	}

	l.loadedFiles[resolved] = mod
	return mod, nil
}

func (l *Loader) resolvePath(importPath string, fromFile string) (string, error) {
	if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		baseDir := filepath.Dir(fromFile)
		if baseDir == "" {
			baseDir = l.basePath
		}
		resolved := filepath.Join(baseDir, importPath)
		if !strings.HasSuffix(resolved, ".carv") {
			resolved += ".carv"
		}
		return filepath.Abs(resolved)
	}

	if l.config != nil && l.config.Dependencies != nil {
		if dep, ok := l.config.Dependencies[importPath]; ok {
			return l.resolvePackage(importPath, dep)
		}
	}

	projectPath := filepath.Join(l.basePath, "src", importPath)
	if !strings.HasSuffix(projectPath, ".carv") {
		projectPath += ".carv"
	}
	if _, err := os.Stat(projectPath); err == nil {
		return projectPath, nil
	}

	modPath := filepath.Join(l.basePath, importPath)
	if !strings.HasSuffix(modPath, ".carv") {
		modPath += ".carv"
	}
	return modPath, nil
}

func (l *Loader) resolvePackage(name string, dep Dependency) (string, error) {
	pkgDir := filepath.Join(l.basePath, "carv_modules", name)

	if dep.Path != "" {
		pkgDir = dep.Path
		if !filepath.IsAbs(pkgDir) {
			pkgDir = filepath.Join(l.basePath, pkgDir)
		}
	}

	modFile := filepath.Join(pkgDir, "mod.carv")
	if _, err := os.Stat(modFile); err == nil {
		return modFile, nil
	}

	indexFile := filepath.Join(pkgDir, "index.carv")
	if _, err := os.Stat(indexFile); err == nil {
		return indexFile, nil
	}

	mainFile := filepath.Join(pkgDir, name+".carv")
	return mainFile, nil
}

func (l *Loader) extractExports(program *ast.Program) map[string]bool {
	exports := make(map[string]bool)

	for _, stmt := range program.Statements {
		switch s := stmt.(type) {
		case *ast.FunctionStatement:
			if s.Public {
				exports[s.Name.Value] = true
			}
		case *ast.ClassStatement:
			if s.Public {
				exports[s.Name.Value] = true
			}
		case *ast.ConstStatement:
			if s.Public {
				exports[s.Name.Value] = true
			}
		case *ast.LetStatement:
			if s.Public {
				exports[s.Name.Value] = true
			}
		}
	}

	return exports
}

func (l *Loader) GetLoadedModules() map[string]*Module {
	return l.loadedFiles
}

type ParseError struct {
	Path   string
	Errors []string
}

func (e *ParseError) Error() string {
	return "parse error in " + e.Path + ": " + strings.Join(e.Errors, "; ")
}
