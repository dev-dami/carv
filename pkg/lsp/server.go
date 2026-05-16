package lsp

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dev-dami/carv/pkg/lexer"
	"github.com/dev-dami/carv/pkg/parser"
	"github.com/dev-dami/carv/pkg/types"
	"github.com/opa-oz/glsp"
	protocol "github.com/opa-oz/glsp/protocol_3_16"
	"github.com/opa-oz/glsp/server"
)

const serverName = "carv-lsp"
const serverVersion = "0.1.0"

var handler protocol.Handler

type serverState struct {
	documents map[protocol.DocumentUri]string
	mu        sync.RWMutex
}

var state *serverState

func RunServer() {
	state = &serverState{
		documents: make(map[protocol.DocumentUri]string),
	}

	handler = protocol.Handler{
		Initialize:  initialize,
		Initialized: initialized,
		Shutdown:    shutdown,
		SetTrace:    setTrace,
	}

	handler.TextDocumentDidOpen = textDocumentDidOpen
	handler.TextDocumentDidChange = textDocumentDidChange
	handler.TextDocumentDidClose = textDocumentDidClose
	handler.TextDocumentHover = textDocumentHover
	handler.TextDocumentDefinition = textDocumentDefinition
	handler.TextDocumentCompletion = textDocumentCompletion

	srv := server.NewServer(&handler, serverName, false)
	srv.RunStdio()
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := handler.CreateServerCapabilities()
	capabilities.HoverProvider = true
	capabilities.DefinitionProvider = true
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{"."},
	}

	v := serverVersion
	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    serverName,
			Version: &v,
		},
	}, nil
}

func initialized(context *glsp.Context, params *protocol.InitializedParams) error {
	return nil
}

func shutdown(context *glsp.Context) error {
	return nil
}

func setTrace(context *glsp.Context, params *protocol.SetTraceParams) error {
	protocol.SetTraceValue(params.Value)
	return nil
}

func textDocumentDidOpen(context *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	state.mu.Lock()
	state.documents[params.TextDocument.URI] = params.TextDocument.Text
	state.mu.Unlock()

	publishDiagnostics(context, params.TextDocument.URI, params.TextDocument.Text)
	return nil
}

func textDocumentDidChange(context *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	state.mu.Lock()
	if len(params.ContentChanges) > 0 {
		if whole, ok := params.ContentChanges[0].(protocol.TextDocumentContentChangeEventWhole); ok {
			state.documents[params.TextDocument.URI] = whole.Text
		}
	}
	state.mu.Unlock()

	state.mu.RLock()
	content := state.documents[params.TextDocument.URI]
	state.mu.RUnlock()

	publishDiagnostics(context, params.TextDocument.URI, content)
	return nil
}

func textDocumentDidClose(context *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	state.mu.Lock()
	delete(state.documents, params.TextDocument.URI)
	state.mu.Unlock()
	return nil
}

func textDocumentHover(context *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	state.mu.RLock()
	content := state.documents[params.TextDocument.URI]
	state.mu.RUnlock()
	if content == "" {
		return nil, nil
	}

	line := int(params.Position.Line) + 1
	col := int(params.Position.Character) + 1
	info := analyzeSource(content)
	if info == nil {
		return nil, nil
	}

	sym := info.symbolAt(line, col)
	if sym == nil {
		return nil, nil
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: fmt.Sprintf("**%s**\n\nType: `%s`", sym.Name, sym.TypeStr),
		},
	}, nil
}

func textDocumentDefinition(context *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	state.mu.RLock()
	content := state.documents[params.TextDocument.URI]
	state.mu.RUnlock()
	if content == "" {
		return nil, nil
	}

	line := int(params.Position.Line) + 1
	col := int(params.Position.Character) + 1
	info := analyzeSource(content)
	if info == nil {
		return nil, nil
	}

	sym := info.symbolAt(line, col)
	if sym == nil || sym.DefLine == 0 {
		return nil, nil
	}

	loc := protocol.Location{
		URI: params.TextDocument.URI,
		Range: protocol.Range{
			Start: protocol.Position{Line: protocol.UInteger(sym.DefLine - 1), Character: protocol.UInteger(sym.DefCol - 1)},
			End:   protocol.Position{Line: protocol.UInteger(sym.DefLine - 1), Character: protocol.UInteger(sym.DefCol - 1 + len(sym.Name))},
		},
	}
	return &loc, nil
}

func textDocumentCompletion(context *glsp.Context, params *protocol.CompletionParams) (any, error) {
	state.mu.RLock()
	content := state.documents[params.TextDocument.URI]
	state.mu.RUnlock()
	if content == "" {
		return []protocol.CompletionItem{}, nil
	}

	info := analyzeSource(content)
	if info == nil {
		return []protocol.CompletionItem{}, nil
	}

	var items []protocol.CompletionItem
	for name, sym := range info.symbols {
		kind := protocol.CompletionItemKindVariable
		if strings.Contains(sym.TypeStr, "fn") || strings.Contains(sym.TypeStr, "Function") {
			kind = protocol.CompletionItemKindFunction
		} else if strings.Contains(sym.TypeStr, "class") || strings.Contains(sym.TypeStr, "Class") {
			kind = protocol.CompletionItemKindClass
		} else if strings.Contains(sym.TypeStr, "interface") || strings.Contains(sym.TypeStr, "Interface") {
			kind = protocol.CompletionItemKindInterface
		}

		detail := sym.TypeStr
		insert := name
		items = append(items, protocol.CompletionItem{
			Label:      name,
			Kind:       &kind,
			Detail:     &detail,
			InsertText: &insert,
		})
	}

	for _, kw := range carvKeywords {
		k := protocol.CompletionItemKindKeyword
		kw := kw
		items = append(items, protocol.CompletionItem{
			Label:      kw,
			Kind:       &k,
			InsertText: &kw,
		})
	}

	return items, nil
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
	diagnostics []protocol.Diagnostic
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
		msg := e
		info.diagnostics = append(info.diagnostics, protocol.Diagnostic{
			Severity: ptrSeverity(protocol.DiagnosticSeverityError),
			Message:  msg,
		})
	}

	checker := types.NewChecker()
	checker.Check(program)

	for _, iss := range checker.ErrorIssues() {
		msg := iss.Message
		info.diagnostics = append(info.diagnostics, protocol.Diagnostic{
			Severity: ptrSeverity(protocol.DiagnosticSeverityError),
			Message:  msg,
			Range: protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(iss.Line - 1), Character: protocol.UInteger(iss.Column - 1)},
				End:   protocol.Position{Line: protocol.UInteger(iss.Line - 1), Character: protocol.UInteger(iss.Column)},
			},
		})
	}

	for _, iss := range checker.WarningIssues() {
		msg := iss.Message
		info.diagnostics = append(info.diagnostics, protocol.Diagnostic{
			Severity: ptrSeverity(protocol.DiagnosticSeverityWarning),
			Message:  msg,
			Range: protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(iss.Line - 1), Character: protocol.UInteger(iss.Column - 1)},
				End:   protocol.Position{Line: protocol.UInteger(iss.Line - 1), Character: protocol.UInteger(iss.Column)},
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

func publishDiagnostics(context *glsp.Context, uri protocol.DocumentUri, content string) {
	info := analyzeSource(content)

	diags := info.diagnostics
	if diags == nil {
		diags = []protocol.Diagnostic{}
	}

	context.Notify("textDocument/publishDiagnostics", protocol.PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diags,
	})
}

func ptrSeverity(s protocol.DiagnosticSeverity) *protocol.DiagnosticSeverity {
	return &s
}
