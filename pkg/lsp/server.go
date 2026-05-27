package lsp

import (
	"fmt"
	"strings"
	"sync"

	"github.com/dev-dami/carv/pkg/ast"
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
		Initialize:    initialize,
		Initialized:   initialized,
		Shutdown:      shutdown,
		SetTrace:      setTrace,
		CancelRequest: cancelRequest,
	}

	handler.TextDocumentDidOpen = textDocumentDidOpen
	handler.TextDocumentDidChange = textDocumentDidChange
	handler.TextDocumentDidClose = textDocumentDidClose
	handler.TextDocumentHover = textDocumentHover
	handler.TextDocumentDefinition = textDocumentDefinition
	handler.TextDocumentCompletion = textDocumentCompletion
	handler.TextDocumentDocumentSymbol = textDocumentDocumentSymbol
	handler.TextDocumentSignatureHelp = textDocumentSignatureHelp
	handler.TextDocumentReferences = textDocumentReferences
	handler.TextDocumentDocumentHighlight = textDocumentDocumentHighlight
	handler.TextDocumentCodeAction = textDocumentCodeAction
	handler.WorkspaceDidChangeWatchedFiles = workspaceDidChangeWatchedFiles

	srv := server.NewServer(&handler, serverName, false)
	_ = srv.RunStdio()
}

func initialize(context *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := handler.CreateServerCapabilities()
	capabilities.HoverProvider = true
	capabilities.DefinitionProvider = true
	capabilities.CompletionProvider = &protocol.CompletionOptions{
		TriggerCharacters: []string{"."},
	}
	capabilities.CodeActionProvider = &protocol.CodeActionOptions{
		CodeActionKinds: []protocol.CodeActionKind{protocol.CodeActionKindQuickFix},
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

func cancelRequest(context *glsp.Context, params *protocol.CancelParams) error {
	// $/cancelRequest is a notification from the client to cancel a pending request.
	// Since the server processes requests synchronously, cancellation is a no-op.
	return nil
}

func workspaceDidChangeWatchedFiles(context *glsp.Context, params *protocol.DidChangeWatchedFilesParams) error {
	// workspace/didChangeWatchedFiles is a notification. File changes in the workspace
	// are handled via textDocument/didChange for open documents, so this is a no-op.
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

func textDocumentCodeAction(context *glsp.Context, params *protocol.CodeActionParams) (any, error) {
	state.mu.RLock()
	content := state.documents[params.TextDocument.URI]
	state.mu.RUnlock()
	if content == "" {
		return nil, nil
	}

	var actions []protocol.CodeAction

	for _, diag := range params.Context.Diagnostics {
		msg := diag.Message

		// Quick fix: "cannot assign to immutable" / "cannot reassign" → suggest `mut`
		if strings.Contains(msg, "cannot assign") || strings.Contains(msg, "immutable") || strings.Contains(msg, "cannot reassign") {
			kind := protocol.CodeActionKindQuickFix
			trueVal := true
			actions = append(actions, protocol.CodeAction{
				Title:       "Add `mut` keyword to make variable mutable",
				Kind:        &kind,
				Diagnostics: []protocol.Diagnostic{diag},
				IsPreferred: &trueVal,
			})
		}

		// Quick fix: "undefined" → suggest adding `let`
		if strings.Contains(msg, "undefined:") {
			kind := protocol.CodeActionKindQuickFix
			actions = append(actions, protocol.CodeAction{
				Title:       "Declare variable with `let`",
				Kind:        &kind,
				Diagnostics: []protocol.Diagnostic{diag},
			})
		}

		// Quick fix: type mismatch → suggest casting
		if strings.Contains(msg, "cannot pass") && strings.Contains(msg, "as") {
			kind := protocol.CodeActionKindQuickFix
			actions = append(actions, protocol.CodeAction{
				Title:       "Add explicit `as` cast",
				Kind:        &kind,
				Diagnostics: []protocol.Diagnostic{diag},
			})
		}
	}

	// Fallback: if no errors exist, offer general diagnostics run
	hasErrors := false
	for _, d := range params.Context.Diagnostics {
		if d.Severity != nil && *d.Severity == protocol.DiagnosticSeverityError {
			hasErrors = true
			break
		}
	}
	if !hasErrors && len(params.Context.Diagnostics) > 0 {
		kind := protocol.CodeActionKindQuickFix
		actions = append(actions, protocol.CodeAction{
			Title:       "Clear diagnostics",
			Kind:        &kind,
			Diagnostics: params.Context.Diagnostics,
		})
	}

	return actions, nil
}

var carvKeywords = []string{
	"let", "mut", "const", "fn", "if", "else", "for", "in", "while", "loop",
	"return", "break", "continue", "match", "class", "interface", "impl",
	"async", "await", "spawn", "try", "new", "as", "self", "static",
	"volatile", "packed", "unsafe", "asm", "require", "true", "false", "nil",
	"void", "int", "float", "bool", "string", "char", "any", "Result", "Option",
	"u8", "u16", "u32", "u64", "i8", "i16", "i32", "i64",
	"f32", "f64", "usize", "isize",
	"pub", "priv", "Ok", "Err", "Some", "None", "function", "import", "export", "module", "from",
	"chan", "send", "recv", "select", "type", "is", "super",
}

// ---- AST helpers for use-site tracking ----

// collectUses walks the AST and records every identifier use position.
// This enables hover/definition/references on identifier use sites.
func collectUses(program *ast.Program, info *sourceInfo) {
	var walkStmts func([]ast.Statement)
	var walkExpr func(ast.Expression)

	walkStmts = func(stmts []ast.Statement) {
		for _, stmt := range stmts {
			switch s := stmt.(type) {
			case *ast.ExpressionStatement:
				walkExpr(s.Expression)
			case *ast.LetStatement:
				if s.Name != nil {
					recordUse(s.Name, info)
				}
				if s.Value != nil {
					walkExpr(s.Value)
				}
			case *ast.ConstStatement:
				if s.Name != nil {
					recordUse(s.Name, info)
				}
				if s.Value != nil {
					walkExpr(s.Value)
				}
			case *ast.ReturnStatement:
				if s.ReturnValue != nil {
					walkExpr(s.ReturnValue)
				}
			case *ast.BlockStatement:
				walkStmts(s.Statements)
			case *ast.FunctionStatement:
				if s.Body != nil {
					walkStmts(s.Body.Statements)
				}
				for _, p := range s.Parameters {
					recordUse(p.Name, info)
				}
			case *ast.ForStatement:
				if s.Init != nil {
					walkStmts([]ast.Statement{s.Init})
				}
				if s.Condition != nil {
					walkExpr(s.Condition)
				}
				if s.Post != nil {
					walkStmts([]ast.Statement{s.Post})
				}
				if s.Body != nil {
					walkStmts(s.Body.Statements)
				}
			case *ast.ForInStatement:
				if s.Iterable != nil {
					walkExpr(s.Iterable)
				}
				if s.Body != nil {
					walkStmts(s.Body.Statements)
				}
			case *ast.WhileStatement:
				if s.Condition != nil {
					walkExpr(s.Condition)
				}
				if s.Body != nil {
					walkStmts(s.Body.Statements)
				}
			case *ast.LoopStatement:
				if s.Body != nil {
					walkStmts(s.Body.Statements)
				}
			case *ast.ClassStatement:
				for _, m := range s.Methods {
					recordUse(m.Name, info)
					if m.Body != nil {
						walkStmts(m.Body.Statements)
					}
					for _, p := range m.Parameters {
						recordUse(p.Name, info)
					}
				}
			case *ast.ImplStatement:
				for _, m := range s.Methods {
					recordUse(m.Name, info)
					if m.Body != nil {
						walkStmts(m.Body.Statements)
					}
					for _, p := range m.Parameters {
						recordUse(p.Name, info)
					}
				}
			case *ast.InterfaceStatement:
				for _, m := range s.Methods {
					recordUse(m.Name, info)
				}
			case *ast.RequireStatement:
			}
		}
	}

	walkExpr = func(expr ast.Expression) {
		if expr == nil {
			return
		}
		switch e := expr.(type) {
		case *ast.Identifier:
			recordUse(e, info)
		case *ast.IntegerLiteral:
		case *ast.FloatLiteral:
		case *ast.StringLiteral:
		case *ast.CharLiteral:
		case *ast.BoolLiteral:
		case *ast.NilLiteral:
		case *ast.PrefixExpression:
			walkExpr(e.Right)
		case *ast.InfixExpression:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *ast.CallExpression:
			walkExpr(e.Function)
			for _, a := range e.Arguments {
				walkExpr(a)
			}
		case *ast.IndexExpression:
			walkExpr(e.Left)
			walkExpr(e.Index)
		case *ast.MemberExpression:
			walkExpr(e.Object)
		case *ast.AssignExpression:
			walkExpr(e.Left)
			walkExpr(e.Right)
		case *ast.IfExpression:
			walkExpr(e.Condition)
			if e.Consequence != nil {
				walkStmts(e.Consequence.Statements)
			}
			if e.Alternative != nil {
				walkStmts(e.Alternative.Statements)
			}
		case *ast.MatchExpression:
			walkExpr(e.Value)
			for _, arm := range e.Arms {
				walkExpr(arm.Pattern)
				walkExpr(arm.Body)
			}
		case *ast.FunctionLiteral:
			if e.Body != nil {
				walkStmts(e.Body.Statements)
			}
			for _, p := range e.Parameters {
				recordUse(p.Name, info)
			}
		case *ast.ArrayLiteral:
			for _, el := range e.Elements {
				walkExpr(el)
			}
		case *ast.MapLiteral:
			for k, v := range e.Pairs {
				walkExpr(k)
				walkExpr(v)
			}
		case *ast.TryExpression:
			walkExpr(e.Value)
		case *ast.CastExpression:
			walkExpr(e.Value)
		case *ast.IsExpression:
			walkExpr(e.Value)
		case *ast.OkExpression:
			walkExpr(e.Value)
		case *ast.ErrExpression:
			walkExpr(e.Value)
		case *ast.NewExpression:
			for _, a := range e.Arguments {
				walkExpr(a)
			}
		case *ast.AwaitExpression:
			walkExpr(e.Value)
		case *ast.SendExpression:
			walkExpr(e.Channel)
			walkExpr(e.Value)
		case *ast.RecvExpression:
			walkExpr(e.Channel)
		case *ast.SpawnExpression:
			if e.Body != nil {
				walkStmts(e.Body.Statements)
			}
		case *ast.InterpolatedString:
			for _, p := range e.Parts {
				walkExpr(p)
			}
		case *ast.BlockExpression:
			if e.Block != nil {
				walkStmts(e.Block.Statements)
			}
		}
	}

	walkStmts(program.Statements)
}

func recordUse(ident *ast.Identifier, info *sourceInfo) {
	if ident == nil {
		return
	}
	line, col := ident.Pos()
	info.uses = append(info.uses, symbolEntry{
		Name:    ident.Value,
		UseLine: line,
		UseCol:  col,
	})
	if _, exists := info.symbols[ident.Value]; !exists {
		info.symbols[ident.Value] = &symbolEntry{
			Name: ident.Value,
		}
	}
}

// ---- DocumentSymbol (outline) ----

func textDocumentDocumentSymbol(context *glsp.Context, params *protocol.DocumentSymbolParams) (any, error) {
	state.mu.RLock()
	content := state.documents[params.TextDocument.URI]
	state.mu.RUnlock()
	if content == "" {
		return []protocol.DocumentSymbol{}, nil
	}

	l := lexer.New(content)
	p := parser.New(l)
	program := p.ParseProgram()
	if program == nil {
		return []protocol.DocumentSymbol{}, nil
	}

	var symbols []protocol.DocumentSymbol
	for _, stmt := range program.Statements {
		if sym := declToSymbol(stmt); sym != nil {
			symbols = append(symbols, *sym)
		}
	}
	return symbols, nil
}

func declToSymbol(stmt ast.Statement) *protocol.DocumentSymbol {
	switch s := stmt.(type) {
	case *ast.FunctionStatement:
		line, col := s.Pos()
		return &protocol.DocumentSymbol{
			Name: s.Name.Value,
			Kind: protocol.SymbolKindFunction,
			Range: protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(line - 1), Character: protocol.UInteger(col - 1)},
				End:   rangeEndForBlock(s.Body),
			},
			SelectionRange: symbolRange(s.Name),
			Detail:         detailForFn(s),
		}
	case *ast.ClassStatement:
		line, col := s.Pos()
		var children []protocol.DocumentSymbol
		for _, m := range s.Methods {
			children = append(children, protocol.DocumentSymbol{
				Name: m.Name.Value,
				Kind: protocol.SymbolKindMethod,
				Range: protocol.Range{
					Start: posToRangeStart(m),
					End:   rangeEndForBlock(m.Body),
				},
				SelectionRange: symbolRange(m.Name),
				Detail:         detailForMethod(m),
			})
		}
		// Range covers from class keyword to end of last method or class line
		endPos := lineRange(s.Name).End
		if len(children) > 0 {
			lastChild := children[len(children)-1]
			endPos = lastChild.Range.End
		}
		return &protocol.DocumentSymbol{
			Name:           s.Name.Value,
			Kind:           protocol.SymbolKindClass,
			Range:          protocol.Range{Start: protocol.Position{Line: protocol.UInteger(line - 1), Character: protocol.UInteger(col - 1)}, End: endPos},
			SelectionRange: symbolRange(s.Name),
			Children:       children,
		}
	case *ast.InterfaceStatement:
		var children []protocol.DocumentSymbol
		for _, m := range s.Methods {
			children = append(children, protocol.DocumentSymbol{
				Name:           m.Name.Value,
				Kind:           protocol.SymbolKindMethod,
				Range:          lineRange(m.Name),
				SelectionRange: symbolRange(m.Name),
			})
		}
		line, col := s.Pos()
		endPos := lineRange(s.Name).End
		if len(children) > 0 {
			lastChild := children[len(children)-1]
			endPos = lastChild.Range.End
		}
		return &protocol.DocumentSymbol{
			Name:           s.Name.Value,
			Kind:           protocol.SymbolKindInterface,
			Range:          protocol.Range{Start: protocol.Position{Line: protocol.UInteger(line - 1), Character: protocol.UInteger(col - 1)}, End: endPos},
			SelectionRange: symbolRange(s.Name),
			Children:       children,
		}
	case *ast.ImplStatement:
		name := "impl"
		if s.Interface != nil {
			name = "impl " + s.Interface.Value + " for " + s.Type.Value
		} else {
			name = "impl " + s.Type.Value
		}
		var children []protocol.DocumentSymbol
		for _, m := range s.Methods {
			children = append(children, protocol.DocumentSymbol{
				Name: m.Name.Value,
				Kind: protocol.SymbolKindMethod,
				Range: protocol.Range{
					Start: posToRangeStart(m),
					End:   rangeEndForBlock(m.Body),
				},
				SelectionRange: symbolRange(m.Name),
				Detail:         detailForMethod(m),
			})
		}
		line, col := s.Pos()
		endPos := protocol.Position{Line: protocol.UInteger(line), Character: 0}
		if len(children) > 0 {
			lastChild := children[len(children)-1]
			endPos = lastChild.Range.End
		}
		return &protocol.DocumentSymbol{
			Name:           name,
			Kind:           protocol.SymbolKindNamespace,
			Range:          protocol.Range{Start: protocol.Position{Line: protocol.UInteger(line - 1), Character: protocol.UInteger(col - 1)}, End: endPos},
			SelectionRange: symbolRange(identifierForImpl(s)),
			Children:       children,
		}
	case *ast.TypeAliasStatement:
		return &protocol.DocumentSymbol{
			Name:           s.Name.Value,
			Kind:           protocol.SymbolKindTypeParameter,
			Range:          lineRange(s.Name),
			SelectionRange: symbolRange(s.Name),
		}
	case *ast.LetStatement:
		if s.Name == nil {
			return nil
		}
		return &protocol.DocumentSymbol{
			Name:           s.Name.Value,
			Kind:           protocol.SymbolKindVariable,
			Range:          lineRange(s.Name),
			SelectionRange: symbolRange(s.Name),
		}
	case *ast.ConstStatement:
		if s.Name == nil {
			return nil
		}
		return &protocol.DocumentSymbol{
			Name:           s.Name.Value,
			Kind:           protocol.SymbolKindConstant,
			Range:          lineRange(s.Name),
			SelectionRange: symbolRange(s.Name),
		}
	}
	return nil
}

func identifierForImpl(s *ast.ImplStatement) *ast.Identifier {
	if s.Interface != nil {
		return s.Interface
	}
	return s.Type
}

func detailForFn(s *ast.FunctionStatement) *string {
	d := "fn("
	for i, p := range s.Parameters {
		if i > 0 {
			d += ", "
		}
		if p.Type != nil {
			_, col := p.Type.Pos()
			_ = col
		}
		d += p.Name.Value
	}
	d += ")"
	return &d
}

func detailForMethod(m *ast.MethodDecl) *string {
	d := "fn("
	for i, p := range m.Parameters {
		if i > 0 {
			d += ", "
		}
		d += p.Name.Value
	}
	d += ")"
	return &d
}

func posToRangeStart(n ast.Node) protocol.Position {
	line, col := n.Pos()
	return protocol.Position{Line: protocol.UInteger(line - 1), Character: protocol.UInteger(col - 1)}
}

func rangeEndForBlock(body *ast.BlockStatement) protocol.Position {
	if body == nil {
		return protocol.Position{Line: 1000000, Character: 1000000}
	}
	if len(body.Statements) > 0 {
		last := body.Statements[len(body.Statements)-1]
		line, col := last.Pos()
		return protocol.Position{Line: protocol.UInteger(line + 1), Character: protocol.UInteger(col)}
	}
	line, col := body.Pos()
	return protocol.Position{Line: protocol.UInteger(line + 1), Character: protocol.UInteger(col)}
}

func rangeEndForStmts(stmts []ast.Statement) protocol.Position {
	if len(stmts) > 0 {
		last := stmts[len(stmts)-1]
		line, col := last.Pos()
		return protocol.Position{Line: protocol.UInteger(line + 1), Character: protocol.UInteger(col)}
	}
	return protocol.Position{Line: 10, Character: 0}
}

func symbolRange(name *ast.Identifier) protocol.Range {
	line, col := name.Pos()
	return protocol.Range{
		Start: protocol.Position{Line: protocol.UInteger(line - 1), Character: protocol.UInteger(col - 1)},
		End:   protocol.Position{Line: protocol.UInteger(line - 1), Character: protocol.UInteger(col - 1 + len(name.Value))},
	}
}

func lineRange(name *ast.Identifier) protocol.Range {
	line, _ := name.Pos()
	return protocol.Range{
		Start: protocol.Position{Line: protocol.UInteger(line - 1), Character: protocol.UInteger(0)},
		End:   protocol.Position{Line: protocol.UInteger(line), Character: protocol.UInteger(0)},
	}
}

// ---- SignatureHelp ----

func textDocumentSignatureHelp(context *glsp.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	state.mu.RLock()
	content := state.documents[params.TextDocument.URI]
	state.mu.RUnlock()
	if content == "" {
		return nil, nil
	}

	l := lexer.New(content)
	p := parser.New(l)
	program := p.ParseProgram()
	if program == nil {
		return nil, nil
	}

	line := int(params.Position.Line) + 1
	col := int(params.Position.Character) + 1

	fn := findCallAtPos(program, line, col)
	if fn == "" {
		return nil, nil
	}

	checker := types.NewChecker()
	checker.Check(program)

	sym, ok := checker.RootScope().LookupSymbol(fn)
	if !ok {
		return nil, nil
	}

	fnType, ok := sym.Type.(*types.FunctionType)
	if !ok {
		return nil, nil
	}

	paramsStr := ""
	labels := make([]protocol.ParameterInformation, len(fnType.Params))
	for i, p := range fnType.Params {
		label := fmt.Sprintf("%s: %s", paramName(i, fnType.Params), p.String())
		labels[i] = protocol.ParameterInformation{
			Label: label,
		}
		if i > 0 {
			paramsStr += ", "
		}
		paramsStr += label
	}

	sig := fmt.Sprintf("(%s)", paramsStr)
	active := uint32(0)
	return &protocol.SignatureHelp{
		Signatures: []protocol.SignatureInformation{
			{
				Label:      fn + sig,
				Parameters: labels,
			},
		},
		ActiveSignature: &active,
		ActiveParameter: &active,
	}, nil
}

func paramName(i int, params []types.Type) string {
	return fmt.Sprintf("arg%d", i)
}

// findCallAtPos walks the AST to find a function call at the given cursor position.
// Returns the function name string if found.
func findCallAtPos(program *ast.Program, line, col int) string {
	var found string
	var walk func([]ast.Statement)
	walk = func(stmts []ast.Statement) {
		if found != "" {
			return
		}
		for _, stmt := range stmts {
			if found != "" {
				return
			}
			switch s := stmt.(type) {
			case *ast.ExpressionStatement:
				exprCallAtPos(s.Expression, line, col, &found)
			case *ast.LetStatement:
				if s.Value != nil {
					exprCallAtPos(s.Value, line, col, &found)
				}
			case *ast.ReturnStatement:
				if s.ReturnValue != nil {
					exprCallAtPos(s.ReturnValue, line, col, &found)
				}
			case *ast.BlockStatement:
				walk(s.Statements)
			case *ast.FunctionStatement:
				if s.Body != nil {
					walk(s.Body.Statements)
				}
			case *ast.ForStatement:
				if s.Body != nil {
					walk(s.Body.Statements)
				}
			case *ast.ForInStatement:
				if s.Body != nil {
					walk(s.Body.Statements)
				}
			case *ast.WhileStatement:
				if s.Body != nil {
					walk(s.Body.Statements)
				}
			case *ast.LoopStatement:
				if s.Body != nil {
					walk(s.Body.Statements)
				}
			case *ast.ClassStatement:
				for _, m := range s.Methods {
					if m.Body != nil {
						walk(m.Body.Statements)
					}
				}
			case *ast.ImplStatement:
				for _, m := range s.Methods {
					if m.Body != nil {
						walk(m.Body.Statements)
					}
				}
			}
		}
	}
	walk(program.Statements)
	return found
}

func exprCallAtPos(expr ast.Expression, line, col int, found *string) {
	if expr == nil || *found != "" {
		return
	}

	switch e := expr.(type) {
	case *ast.CallExpression:
		ceLine, ceCol := e.Pos()
		// Check if cursor is within the call expression
		if ceLine == line && ceCol <= col && col <= ceCol+10 {
			if ident, ok := e.Function.(*ast.Identifier); ok {
				*found = ident.Value
				return
			}
		}
		// Still recurse into arguments
		for _, a := range e.Arguments {
			exprCallAtPos(a, line, col, found)
		}
	case *ast.InfixExpression:
		exprCallAtPos(e.Left, line, col, found)
		exprCallAtPos(e.Right, line, col, found)
	case *ast.PrefixExpression:
		exprCallAtPos(e.Right, line, col, found)
	case *ast.IfExpression:
		exprCallAtPos(e.Condition, line, col, found)
	case *ast.MatchExpression:
		exprCallAtPos(e.Value, line, col, found)
		for _, arm := range e.Arms {
			exprCallAtPos(arm.Body, line, col, found)
		}
	case *ast.FunctionLiteral:
		if e.Body != nil {
			walkBlockForCall(e.Body, line, col, found)
		}
	case *ast.BlockExpression:
		if e.Block != nil {
			walkBlockForCall(e.Block, line, col, found)
		}
	case *ast.ArrayLiteral:
		for _, el := range e.Elements {
			exprCallAtPos(el, line, col, found)
		}
	case *ast.TryExpression:
		exprCallAtPos(e.Value, line, col, found)
	case *ast.AwaitExpression:
		exprCallAtPos(e.Value, line, col, found)
	case *ast.CastExpression:
		exprCallAtPos(e.Value, line, col, found)
	case *ast.NewExpression:
		for _, a := range e.Arguments {
			exprCallAtPos(a, line, col, found)
		}
	}
}

func walkBlockForCall(block *ast.BlockStatement, line, col int, found *string) {
	for _, stmt := range block.Statements {
		if *found != "" {
			return
		}
		if es, ok := stmt.(*ast.ExpressionStatement); ok {
			exprCallAtPos(es.Expression, line, col, found)
		}
		if ret, ok := stmt.(*ast.ReturnStatement); ok {
			exprCallAtPos(ret.ReturnValue, line, col, found)
		}
		if bs, ok := stmt.(*ast.BlockStatement); ok {
			walkBlockForCall(bs, line, col, found)
		}
		if fs, ok := stmt.(*ast.FunctionStatement); ok && fs.Body != nil {
			walkBlockForCall(fs.Body, line, col, found)
		}
	}
}

// ---- References ----

func textDocumentReferences(context *glsp.Context, params *protocol.ReferenceParams) ([]protocol.Location, error) {
	state.mu.RLock()
	content := state.documents[params.TextDocument.URI]
	state.mu.RUnlock()
	if content == "" {
		return []protocol.Location{}, nil
	}

	info := analyzeSource(content)
	if info == nil {
		return []protocol.Location{}, nil
	}

	line := int(params.Position.Line) + 1
	col := int(params.Position.Character) + 1

	sym := info.symbolAt(line, col)
	if sym == nil {
		return []protocol.Location{}, nil
	}

	uri := params.TextDocument.URI
	var refs []protocol.Location

	// Add definition
	if sym.DefLine > 0 {
		refs = append(refs, protocol.Location{
			URI: uri,
			Range: protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(sym.DefLine - 1), Character: protocol.UInteger(sym.DefCol - 1)},
				End:   protocol.Position{Line: protocol.UInteger(sym.DefLine - 1), Character: protocol.UInteger(sym.DefCol - 1 + len(sym.Name))},
			},
		})
	}

	// Add all use sites
	for _, u := range info.uses {
		if u.Name == sym.Name {
			refs = append(refs, protocol.Location{
				URI: uri,
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(u.UseLine - 1), Character: protocol.UInteger(u.UseCol - 1)},
					End:   protocol.Position{Line: protocol.UInteger(u.UseLine - 1), Character: protocol.UInteger(u.UseCol - 1 + len(u.Name))},
				},
			})
		}
	}

	if len(refs) == 0 {
		return []protocol.Location{}, nil
	}
	return refs, nil
}

// ---- DocumentHighlight ----

func textDocumentDocumentHighlight(context *glsp.Context, params *protocol.DocumentHighlightParams) ([]protocol.DocumentHighlight, error) {
	state.mu.RLock()
	content := state.documents[params.TextDocument.URI]
	state.mu.RUnlock()
	if content == "" {
		return []protocol.DocumentHighlight{}, nil
	}

	info := analyzeSource(content)
	if info == nil {
		return []protocol.DocumentHighlight{}, nil
	}

	line := int(params.Position.Line) + 1
	col := int(params.Position.Character) + 1

	sym := info.symbolAt(line, col)
	if sym == nil {
		return []protocol.DocumentHighlight{}, nil
	}

	var highlights []protocol.DocumentHighlight

	// Add definition
	if sym.DefLine > 0 {
		highlights = append(highlights, protocol.DocumentHighlight{
			Range: protocol.Range{
				Start: protocol.Position{Line: protocol.UInteger(sym.DefLine - 1), Character: protocol.UInteger(sym.DefCol - 1)},
				End:   protocol.Position{Line: protocol.UInteger(sym.DefLine - 1), Character: protocol.UInteger(sym.DefCol - 1 + len(sym.Name))},
			},
			Kind: ptrHighlightKind(protocol.DocumentHighlightKindText),
		})
	}

	// Add all use sites
	for _, u := range info.uses {
		if u.Name == sym.Name && !(u.UseLine == sym.DefLine && u.UseCol == sym.DefCol) {
			highlights = append(highlights, protocol.DocumentHighlight{
				Range: protocol.Range{
					Start: protocol.Position{Line: protocol.UInteger(u.UseLine - 1), Character: protocol.UInteger(u.UseCol - 1)},
					End:   protocol.Position{Line: protocol.UInteger(u.UseLine - 1), Character: protocol.UInteger(u.UseCol - 1 + len(u.Name))},
				},
				Kind: ptrHighlightKind(protocol.DocumentHighlightKindRead),
			})
		}
	}

	if len(highlights) == 0 {
		return []protocol.DocumentHighlight{}, nil
	}
	return highlights, nil
}

func ptrHighlightKind(k protocol.DocumentHighlightKind) *protocol.DocumentHighlightKind {
	return &k
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

	collectUses(program, info)

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
