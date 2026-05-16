package lsp

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/types"
	"github.com/owenrumney/go-lsp/lsp"
	"github.com/owenrumney/go-lsp/server"
)

const serverName = "carv-lsp"
const serverVersion = "0.1.0"

type Handler struct {
	documents map[lsp.DocumentURI]string
	mu        sync.RWMutex
	client    *server.Client
}

func NewHandler() *Handler {
	return &Handler{
		documents: make(map[lsp.DocumentURI]string),
	}
}

func (h *Handler) SetClient(client *server.Client) {
	h.client = client
}

func (h *Handler) Initialize(ctx context.Context, params *lsp.InitializeParams) (*lsp.InitializeResult, error) {
	trueVal := true
	return &lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptions{
				OpenClose: &trueVal,
				Change:    lsp.SyncFull,
			},
			HoverProvider:      &trueVal,
			DefinitionProvider: &trueVal,
			CompletionProvider: &lsp.CompletionOptions{
				TriggerCharacters: []string{"."},
			},
		},
		ServerInfo: &lsp.ServerInfo{
			Name:    serverName,
			Version: serverVersion,
		},
	}, nil
}

func (h *Handler) Initialized(ctx context.Context, params *lsp.InitializedParams) error {
	return nil
}

func (h *Handler) Shutdown(ctx context.Context) error {
	return nil
}

func (h *Handler) DidOpen(ctx context.Context, params *lsp.DidOpenTextDocumentParams) error {
	h.mu.Lock()
	h.documents[params.TextDocument.URI] = params.TextDocument.Text
	h.mu.Unlock()

	h.publishDiagnostics(ctx, params.TextDocument.URI, params.TextDocument.Text)
	return nil
}

func (h *Handler) DidChange(ctx context.Context, params *lsp.DidChangeTextDocumentParams) error {
	h.mu.Lock()
	if len(params.ContentChanges) > 0 {
		h.documents[params.TextDocument.URI] = params.ContentChanges[0].Text
	}
	h.mu.Unlock()

	h.mu.RLock()
	content := h.documents[params.TextDocument.URI]
	h.mu.RUnlock()

	h.publishDiagnostics(ctx, params.TextDocument.URI, content)
	return nil
}

func (h *Handler) DidClose(ctx context.Context, params *lsp.DidCloseTextDocumentParams) error {
	h.mu.Lock()
	delete(h.documents, params.TextDocument.URI)
	h.mu.Unlock()
	return nil
}

func (h *Handler) Hover(ctx context.Context, params *lsp.HoverParams) (*lsp.Hover, error) {
	h.mu.RLock()
	content := h.documents[params.TextDocument.URI]
	h.mu.RUnlock()
	if content == "" {
		return nil, nil
	}

	pos := lspPosToLineCol(params.Position)
	info := analyzeSource(content)
	if info == nil {
		return nil, nil
	}

	sym := info.symbolAt(pos.line, pos.col)
	if sym == nil {
		return nil, nil
	}

	return &lsp.Hover{
		Contents: lsp.MarkupContent{
			Kind:  lsp.Markdown,
			Value: fmt.Sprintf("**%s**\n\nType: `%s`", sym.Name, sym.TypeStr),
		},
	}, nil
}

func (h *Handler) Definition(ctx context.Context, params *lsp.DefinitionParams) ([]lsp.Location, error) {
	h.mu.RLock()
	content := h.documents[params.TextDocument.URI]
	h.mu.RUnlock()
	if content == "" {
		return nil, nil
	}

	pos := lspPosToLineCol(params.Position)
	info := analyzeSource(content)
	if info == nil {
		return nil, nil
	}

	sym := info.symbolAt(pos.line, pos.col)
	if sym == nil || sym.DefLine == 0 {
		return nil, nil
	}

	return []lsp.Location{{
		URI: params.TextDocument.URI,
		Range: lsp.Range{
			Start: lsp.Position{Line: sym.DefLine - 1, Character: sym.DefCol - 1},
			End:   lsp.Position{Line: sym.DefLine - 1, Character: sym.DefCol - 1 + len(sym.Name)},
		},
	}}, nil
}

func (h *Handler) Completion(ctx context.Context, params *lsp.CompletionParams) (*lsp.CompletionList, error) {
	h.mu.RLock()
	content := h.documents[params.TextDocument.URI]
	h.mu.RUnlock()
	if content == "" {
		return &lsp.CompletionList{}, nil
	}

	info := analyzeSource(content)
	if info == nil {
		return &lsp.CompletionList{}, nil
	}

	var items []lsp.CompletionItem
	for name, sym := range info.symbols {
		kind := lsp.CompletionItemKindVariable
		if strings.Contains(sym.TypeStr, "fn") || strings.Contains(sym.TypeStr, "Function") {
			kind = lsp.CompletionItemKindFunction
		} else if strings.Contains(sym.TypeStr, "class") || strings.Contains(sym.TypeStr, "Class") {
			kind = lsp.CompletionItemKindClass
		} else if strings.Contains(sym.TypeStr, "interface") || strings.Contains(sym.TypeStr, "Interface") {
			kind = lsp.CompletionItemKindInterface
		}

		k := kind
		items = append(items, lsp.CompletionItem{
			Label:      name,
			Kind:       &k,
			Detail:     sym.TypeStr,
			InsertText: name,
		})
	}

	for _, kw := range carvKeywords {
		k := lsp.CompletionItemKindKeyword
		items = append(items, lsp.CompletionItem{
			Label:      kw,
			Kind:       &k,
			InsertText: kw,
		})
	}

	return &lsp.CompletionList{Items: items}, nil
}

var carvKeywords = []string{
	"let", "mut", "const", "fn", "if", "else", "for", "in", "while", "loop",
	"return", "break", "continue", "match", "class", "interface", "impl",
	"async", "await", "spawn", "try", "new", "as", "self", "static",
	"volatile", "packed", "unsafe", "asm", "require", "true", "false", "nil",
	"void", "int", "bool", "string", "char", "any",
	"u8", "u16", "u32", "u64", "i8", "i16", "i32", "i64",
	"f32", "f64", "usize", "isize",
}

type symbolEntry struct {
	Name    string
	TypeStr string
	DefLine int
	DefCol  int
	UseLine int
	UseCol  int
}

type sourceInfo struct {
	symbols     map[string]*symbolEntry
	uses        []symbolEntry
	diagnostics []lsp.Diagnostic
}

func (si *sourceInfo) symbolAt(line, col int) *symbolEntry {
	for _, u := range si.uses {
		if u.UseLine == line && u.UseCol == col {
			if sym, ok := si.symbols[u.Name]; ok {
				return sym
			}
		}
	}
	for _, sym := range si.symbols {
		if sym.DefLine == line && sym.DefCol <= col && col <= sym.DefCol+len(sym.Name) {
			return sym
		}
	}
	return nil
}

func analyzeSource(src string) *sourceInfo {
	l := lexer.New(src)
	p := parser.New(l)
	program := p.ParseProgram()

	info := &sourceInfo{
		symbols: make(map[string]*symbolEntry),
	}

	for _, e := range p.Errors() {
		info.diagnostics = append(info.diagnostics, lsp.Diagnostic{
			Severity: severityPtr(lsp.SeverityError),
			Message:  e,
		})
	}

	checker := types.NewChecker()
	checker.Check(program)

	for _, iss := range checker.ErrorIssues() {
		info.diagnostics = append(info.diagnostics, lsp.Diagnostic{
			Severity: severityPtr(lsp.SeverityError),
			Message:  iss.Message,
			Range: lsp.Range{
				Start: lsp.Position{Line: iss.Line - 1, Character: iss.Column - 1},
				End:   lsp.Position{Line: iss.Line - 1, Character: iss.Column},
			},
		})
	}

	for _, iss := range checker.WarningIssues() {
		info.diagnostics = append(info.diagnostics, lsp.Diagnostic{
			Severity: severityPtr(lsp.SeverityWarning),
			Message:  iss.Message,
			Range: lsp.Range{
				Start: lsp.Position{Line: iss.Line - 1, Character: iss.Column - 1},
				End:   lsp.Position{Line: iss.Line - 1, Character: iss.Column},
			},
		})
	}

	for name, sym := range checker.RootScope().AllSymbols() {
		info.symbols[name] = &symbolEntry{
			Name:    name,
			TypeStr: sym.Type.String(),
			DefLine: sym.Line,
			DefCol:  sym.Column,
		}
	}

	return info
}

func (h *Handler) publishDiagnostics(ctx context.Context, uri lsp.DocumentURI, content string) {
	if h.client == nil {
		return
	}
	info := analyzeSource(content)

	diags := info.diagnostics
	if diags == nil {
		diags = []lsp.Diagnostic{}
	}

	_ = h.client.PublishDiagnostics(ctx, &lsp.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	})
}

func RunServer() error {
	h := NewHandler()
	srv := server.NewServer(h)
	return srv.Run(context.Background(), server.RunStdio())
}

func lspPosToLineCol(p lsp.Position) struct{ line, col int } {
	return struct{ line, col int }{line: p.Line + 1, col: p.Character + 1}
}

func severityPtr(s lsp.DiagnosticSeverity) *lsp.DiagnosticSeverity {
	return &s
}
