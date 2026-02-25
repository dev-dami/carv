package codegen

import (
	"fmt"
	"strings"

	"github.com/dev-dami/carv/pkg/ast"
	"github.com/dev-dami/carv/pkg/module"
	"github.com/dev-dami/carv/pkg/types"
)

type cgenVar struct {
	CType   string
	Mutable bool
	Owned   bool
}

type cgenScope struct {
	parent *cgenScope
	vars   map[string]*cgenVar
}

type interfaceInfo struct {
	name    string
	methods []*ast.MethodSignature
}

type implInfo struct {
	ifaceName string
	typeName  string
	methods   []*ast.MethodDecl
}

type capturedVar struct {
	Name  string
	CType string
}

type CGenerator struct {
	output          strings.Builder
	indent          int
	tempCounter     int
	arrayLengths    map[string]int
	scope           *cgenScope
	fnReturnTypes   map[string]string
	typeInfo        map[ast.Expression]types.Type
	preamble        []string
	inFunction      bool
	funcRetType     string
	interfaces      map[string]*interfaceInfo
	implList        []*implInfo
	closureCounter  int
	closureDefs     []string
	captureMap      map[string]string // varName -> "__env->varName" during closure function generation
	lastClosureType string
	hasAsync        bool
	asyncFns        map[string]*asyncFnInfo
	inAsyncFn       bool
	asyncFnName     string
	asyncStateID    int
	builtinAliases  map[string]string
}

type asyncFnInfo struct {
	Name       string
	Params     []paramInfo
	Locals     []paramInfo
	ReturnType string
}

type paramInfo struct {
	Name  string
	CType string
}

func NewCGenerator() *CGenerator {
	g := &CGenerator{
		arrayLengths:   make(map[string]int),
		fnReturnTypes:  make(map[string]string),
		interfaces:     make(map[string]*interfaceInfo),
		asyncFns:       make(map[string]*asyncFnInfo),
		builtinAliases: make(map[string]string),
	}
	g.scope = newScope(nil)
	return g
}

func (g *CGenerator) nextClosureID() int {
	id := g.closureCounter
	g.closureCounter++
	return id
}

func (g *CGenerator) SetTypeInfo(info map[ast.Expression]types.Type) {
	g.typeInfo = info
}

func (g *CGenerator) addPreamble(stmt string) {
	g.preamble = append(g.preamble, stmt)
}

func (g *CGenerator) flushPreamble() {
	for _, stmt := range g.preamble {
		g.writeln(stmt)
	}
	g.preamble = g.preamble[:0]
}

func checkerTypeToCString(t types.Type) string {
	if t == nil {
		return ""
	}
	switch {
	case t.Equals(types.Int):
		return "carv_int"
	case t.Equals(types.Float):
		return "carv_float"
	case t.Equals(types.Bool):
		return "carv_bool"
	case t.Equals(types.String):
		return "carv_string"
	case t.Equals(types.Char):
		return "carv_int"
	case t.Equals(types.Void):
		return "void"
	case t.Equals(types.Nil):
		return "void*"
	}
	if arr, ok := t.(*types.ArrayType); ok {
		elem := checkerTypeToCString(arr.Element)
		switch elem {
		case "carv_int":
			return "carv_int_array"
		case "carv_float":
			return "carv_float_array"
		case "carv_string":
			return "carv_string_array"
		case "carv_bool":
			return "carv_bool_array"
		}
		return "carv_int_array"
	}
	if _, ok := t.(*types.MapType); ok {
		return "carv_int"
	}
	if cls, ok := t.(*types.ClassType); ok {
		return cls.Name + "*"
	}
	if ref, ok := t.(*types.RefType); ok {
		if iface, ok := ref.Inner.(*types.InterfaceType); ok {
			if ref.Mutable {
				return iface.Name + "_mut_ref"
			}
			return iface.Name + "_ref"
		}
		inner := checkerTypeToCString(ref.Inner)
		if ref.Mutable {
			return inner + "*"
		}
		return "const " + inner + "*"
	}
	if iface, ok := t.(*types.InterfaceType); ok {
		return iface.Name + "_ref"
	}
	if _, ok := t.(*types.FunctionType); ok {
		return "void*"
	}
	if _, ok := t.(*types.FutureType); ok {
		return "void*"
	}
	return ""
}

func (g *CGenerator) resolveType(expr ast.Expression) string {
	if g.typeInfo != nil {
		if t, ok := g.typeInfo[expr]; ok {
			if cs := checkerTypeToCString(t); cs != "" {
				return cs
			}
		}
	}
	return g.inferExprType(expr)
}

func newScope(parent *cgenScope) *cgenScope {
	return &cgenScope{parent: parent, vars: make(map[string]*cgenVar)}
}

func (g *CGenerator) enterScope() {
	g.scope = newScope(g.scope)
}

func (g *CGenerator) exitScope() {
	if g.scope.parent != nil {
		g.scope = g.scope.parent
	}
}

func (g *CGenerator) emitScopeDrops() {
	if g.scope == nil {
		return
	}
	for name, v := range g.scope.vars {
		if v.Owned {
			switch v.CType {
			case "carv_string":
				g.writeln(fmt.Sprintf("carv_string_drop(&%s);", name))
			}
		}
	}
}

func (g *CGenerator) declareVar(name, ctype string, mutable, owned bool) {
	g.scope.vars[name] = &cgenVar{CType: ctype, Mutable: mutable, Owned: owned}
}

func (g *CGenerator) lookupVar(name string) *cgenVar {
	for s := g.scope; s != nil; s = s.parent {
		if v, ok := s.vars[name]; ok {
			return v
		}
	}
	return nil
}

func (g *CGenerator) getVarType(name string) string {
	if v := g.lookupVar(name); v != nil {
		return v.CType
	}
	return ""
}

func (g *CGenerator) collectFunctionReturnTypes(program *ast.Program) {
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionStatement); ok {
			g.enterScope()
			for _, p := range fn.Parameters {
				pType := g.typeToC(p.Type)
				g.declareVar(p.Name.Value, pType, false, false)
			}

			retType := g.inferFunctionReturnType(fn)
			g.fnReturnTypes[fn.Name.Value] = retType

			if retType == "carv_result" {
				okType, errType := g.inferResultPayloadTypes(fn.Body)
				g.declareVar(fn.Name.Value+"_result_ok", okType, false, false)
				g.declareVar(fn.Name.Value+"_result_err", errType, false, false)
			}
			g.exitScope()
		}
	}
}

func (g *CGenerator) collectAsyncFunctions(program *ast.Program) {
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionStatement); ok && fn.Async {
			g.hasAsync = true
			info := &asyncFnInfo{
				Name:       fn.Name.Value,
				ReturnType: g.inferFunctionReturnType(fn),
			}
			for _, p := range fn.Parameters {
				info.Params = append(info.Params, paramInfo{
					Name:  p.Name.Value,
					CType: g.typeToC(p.Type),
				})
			}
			g.collectAsyncLocals(fn.Body, info)
			g.asyncFns[fn.Name.Value] = info
		}
	}
}

func (g *CGenerator) collectAsyncLocals(body *ast.BlockStatement, info *asyncFnInfo) {
	if body == nil {
		return
	}
	seen := make(map[string]bool, len(info.Params))
	for _, p := range info.Params {
		seen[p.Name] = true
	}
	for _, l := range info.Locals {
		seen[l.Name] = true
	}
	g.collectAsyncLocalsFromBlock(body, info, seen)
}

func (g *CGenerator) collectAsyncLocalsFromBlock(body *ast.BlockStatement, info *asyncFnInfo, seen map[string]bool) {
	if body == nil {
		return
	}
	for _, stmt := range body.Statements {
		g.collectAsyncLocalsFromStatement(stmt, info, seen)
	}
}

func (g *CGenerator) collectAsyncLocalsFromStatement(stmt ast.Statement, info *asyncFnInfo, seen map[string]bool) {
	switch s := stmt.(type) {
	case *ast.LetStatement:
		if seen[s.Name.Value] {
			return
		}
		seen[s.Name.Value] = true
		info.Locals = append(info.Locals, paramInfo{
			Name:  s.Name.Value,
			CType: g.inferType(s.Value),
		})
	case *ast.ForStatement:
		if s.Init != nil {
			g.collectAsyncLocalsFromStatement(s.Init, info, seen)
		}
		g.collectAsyncLocalsFromBlock(s.Body, info, seen)
		if s.Post != nil {
			g.collectAsyncLocalsFromStatement(s.Post, info, seen)
		}
	case *ast.ForInStatement:
		g.collectAsyncLocalsFromBlock(s.Body, info, seen)
	case *ast.WhileStatement:
		g.collectAsyncLocalsFromBlock(s.Body, info, seen)
	case *ast.LoopStatement:
		g.collectAsyncLocalsFromBlock(s.Body, info, seen)
	case *ast.ExpressionStatement:
		if ifExpr, ok := s.Expression.(*ast.IfExpression); ok {
			g.collectAsyncLocalsFromBlock(ifExpr.Consequence, info, seen)
			g.collectAsyncLocalsFromBlock(ifExpr.Alternative, info, seen)
		}
	case *ast.BlockStatement:
		g.collectAsyncLocalsFromBlock(s, info, seen)
	}
}

func (g *CGenerator) inferResultPayloadTypes(body *ast.BlockStatement) (okType, errType string) {
	okType = "carv_int"
	errType = "carv_string"

	for _, stmt := range body.Statements {
		if ret, ok := stmt.(*ast.ReturnStatement); ok && ret.ReturnValue != nil {
			if okExpr, isOk := ret.ReturnValue.(*ast.OkExpression); isOk {
				okType = g.resolveType(okExpr.Value)
			} else if errExpr, isErr := ret.ReturnValue.(*ast.ErrExpression); isErr {
				errType = g.resolveType(errExpr.Value)
			}
		}
	}
	return
}

func (g *CGenerator) Generate(program *ast.Program) string {
	g.collectBuiltinModuleAliases(program)
	g.collectFunctionReturnTypes(program)
	g.collectInterfacesAndImpls(program)
	g.collectAsyncFunctions(program)
	g.emitRuntime()

	for _, stmt := range program.Statements {
		if cls, ok := stmt.(*ast.ClassStatement); ok {
			g.generateClassDecl(cls)
		}
	}

	g.generateInterfaceTypedefs()

	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionStatement); ok {
			g.generateFunctionDecl(fn)
		}
	}

	for _, stmt := range program.Statements {
		if cls, ok := stmt.(*ast.ClassStatement); ok {
			g.generateClassMethodDecls(cls)
		}
	}

	g.generateImplMethodDecls()

	g.writeln("")

	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionStatement); ok {
			g.generateFunction(fn)
		}
	}

	for _, stmt := range program.Statements {
		if cls, ok := stmt.(*ast.ClassStatement); ok {
			g.generateClassMethods(cls)
		}
	}

	g.generateImplMethods(program)
	g.generateImplWrappers()

	preMainOutput := g.output.String()
	g.output.Reset()

	var asyncMain *ast.FunctionStatement
	for _, stmt := range program.Statements {
		if fn, ok := stmt.(*ast.FunctionStatement); ok && fn.Async && g.safeName(fn.Name.Value) == "carv_main" {
			asyncMain = fn
			break
		}
	}

	g.writeln("")
	g.writeln("int main(void) {")
	g.indent++

	if asyncMain != nil {
		g.writeln("carv_loop loop;")
		g.writeln("carv_loop_init(&loop);")
		g.writeln("carv_main_frame* mf = carv_main();")
		g.writeln("carv_task main_task = { .poll = carv_main_poll, .drop = NULL, .frame = mf };")
		g.writeln("carv_loop_add_task(&loop, &main_task);")
		g.writeln("carv_loop_run(&loop);")
	} else {
		for _, stmt := range program.Statements {
			switch stmt.(type) {
			case *ast.FunctionStatement, *ast.ClassStatement, *ast.InterfaceStatement, *ast.ImplStatement:
				continue
			default:
				g.generateStatement(stmt)
			}
		}
	}

	g.writeln("carv_arena_free_all();")
	g.writeln("return 0;")
	g.indent--
	g.writeln("}")

	mainBody := g.output.String()
	g.output.Reset()

	g.output.WriteString(preMainOutput)
	for _, def := range g.closureDefs {
		g.writeln(def)
	}
	g.output.WriteString(mainBody)

	return g.output.String()
}

func (g *CGenerator) collectBuiltinModuleAliases(program *ast.Program) {
	for _, stmt := range program.Statements {
		req, ok := stmt.(*ast.RequireStatement)
		if !ok || req.Path == nil {
			continue
		}
		if !module.IsBuiltinModule(req.Path.Value) {
			continue
		}
		if req.Alias != nil {
			g.builtinAliases[req.Alias.Value] = req.Path.Value
			continue
		}
		if len(req.Names) == 0 && !req.All {
			g.builtinAliases[req.Path.Value] = req.Path.Value
		}
	}
}

func (g *CGenerator) emitRuntime() {
	g.writeln("#include <stdio.h>")
	g.writeln("#include <stdlib.h>")
	g.writeln("#include <string.h>")
	g.writeln("#include <stdbool.h>")
	g.writeln("#include <unistd.h>")
	g.writeln("#include <sys/types.h>")
	g.writeln("#include <sys/socket.h>")
	g.writeln("#include <netinet/in.h>")
	g.writeln("#include <arpa/inet.h>")
	g.writeln("")
	g.writeln("// Arena allocator for automatic memory management")
	g.writeln("#define CARV_ARENA_BLOCK_SIZE (1024 * 1024)  // 1MB blocks")
	g.writeln("typedef struct carv_arena_block {")
	g.writeln("    char* data;")
	g.writeln("    size_t used;")
	g.writeln("    size_t capacity;")
	g.writeln("    struct carv_arena_block* next;")
	g.writeln("} carv_arena_block;")
	g.writeln("")
	g.writeln("typedef struct {")
	g.writeln("    carv_arena_block* head;")
	g.writeln("    carv_arena_block* current;")
	g.writeln("} carv_arena;")
	g.writeln("")
	g.writeln("static carv_arena carv_global_arena = {NULL, NULL};")
	g.writeln("")
	g.writeln("static carv_arena_block* carv_arena_new_block(size_t min_size) {")
	g.writeln("    size_t size = min_size > CARV_ARENA_BLOCK_SIZE ? min_size : CARV_ARENA_BLOCK_SIZE;")
	g.writeln("    carv_arena_block* block = (carv_arena_block*)malloc(sizeof(carv_arena_block));")
	g.writeln("    block->data = (char*)malloc(size);")
	g.writeln("    block->used = 0;")
	g.writeln("    block->capacity = size;")
	g.writeln("    block->next = NULL;")
	g.writeln("    return block;")
	g.writeln("}")
	g.writeln("")
	g.writeln("static void* carv_arena_alloc(size_t size) {")
	g.writeln("    size = (size + 7) & ~7;  // 8-byte alignment")
	g.writeln("    if (!carv_global_arena.current || carv_global_arena.current->used + size > carv_global_arena.current->capacity) {")
	g.writeln("        carv_arena_block* block = carv_arena_new_block(size);")
	g.writeln("        if (carv_global_arena.current) {")
	g.writeln("            carv_global_arena.current->next = block;")
	g.writeln("        } else {")
	g.writeln("            carv_global_arena.head = block;")
	g.writeln("        }")
	g.writeln("        carv_global_arena.current = block;")
	g.writeln("    }")
	g.writeln("    void* ptr = carv_global_arena.current->data + carv_global_arena.current->used;")
	g.writeln("    carv_global_arena.current->used += size;")
	g.writeln("    return ptr;")
	g.writeln("}")
	g.writeln("")
	g.writeln("static void carv_arena_free_all(void) {")
	g.writeln("    carv_arena_block* block = carv_global_arena.head;")
	g.writeln("    while (block) {")
	g.writeln("        carv_arena_block* next = block->next;")
	g.writeln("        free(block->data);")
	g.writeln("        free(block);")
	g.writeln("        block = next;")
	g.writeln("    }")
	g.writeln("    carv_global_arena.head = NULL;")
	g.writeln("    carv_global_arena.current = NULL;")
	g.writeln("}")
	g.writeln("")
	g.writeln("typedef long long carv_int;")
	g.writeln("typedef double carv_float;")
	g.writeln("typedef bool carv_bool;")
	g.writeln("typedef struct { char* data; size_t len; bool owned; } carv_string;")
	g.writeln("")
	g.writeln("// Create string from C string literal (NOT owned - never freed)")
	g.writeln("static carv_string carv_string_lit(const char* s) {")
	g.writeln("    return (carv_string){(char*)s, strlen(s), false};")
	g.writeln("}")
	g.writeln("")
	g.writeln("// Create owned string from heap allocation")
	g.writeln("static carv_string carv_string_own(char* data, size_t len) {")
	g.writeln("    return (carv_string){data, len, true};")
	g.writeln("}")
	g.writeln("")
	g.writeln("// Clone a string (always returns owned copy)")
	g.writeln("static carv_string carv_string_clone(carv_string s) {")
	g.writeln("    if (!s.data) return (carv_string){NULL, 0, false};")
	g.writeln("    char* copy = (char*)carv_arena_alloc(s.len + 1);")
	g.writeln("    memcpy(copy, s.data, s.len + 1);")
	g.writeln("    return (carv_string){copy, s.len, true};")
	g.writeln("}")
	g.writeln("")
	g.writeln("// Move ownership (source zeroed)")
	g.writeln("static carv_string carv_string_move(carv_string* s) {")
	g.writeln("    carv_string out = *s;")
	g.writeln("    s->data = NULL;")
	g.writeln("    s->len = 0;")
	g.writeln("    s->owned = false;")
	g.writeln("    return out;")
	g.writeln("}")
	g.writeln("")
	g.writeln("// Drop a string (free if owned)")
	g.writeln("static void carv_string_drop(carv_string* s) {")
	g.writeln("    // Note: with arena allocator, we don't actually free individual strings")
	g.writeln("    // This is a no-op for now but the interface exists for future per-alloc freeing")
	g.writeln("    s->data = NULL;")
	g.writeln("    s->len = 0;")
	g.writeln("    s->owned = false;")
	g.writeln("}")
	g.writeln("")
	g.writeln("static carv_string carv_strdup_str(const char* s) {")
	g.writeln("    size_t len = strlen(s) + 1;")
	g.writeln("    char* copy = (char*)carv_arena_alloc(len);")
	g.writeln("    memcpy(copy, s, len);")
	g.writeln("    return (carv_string){copy, len - 1, true};")
	g.writeln("}")
	g.writeln("")
	g.writeln("typedef struct { carv_int* data; carv_int len; carv_int cap; } carv_int_array;")
	g.writeln("typedef struct { carv_float* data; carv_int len; carv_int cap; } carv_float_array;")
	g.writeln("typedef struct { carv_string* data; carv_int len; carv_int cap; } carv_string_array;")
	g.writeln("typedef struct { carv_bool* data; carv_int len; carv_int cap; } carv_bool_array;")
	g.writeln("")
	g.writeln("carv_int_array carv_new_int_array(carv_int len) {")
	g.writeln("    carv_int_array arr;")
	g.writeln("    arr.data = (carv_int*)carv_arena_alloc(len * sizeof(carv_int));")
	g.writeln("    arr.len = len;")
	g.writeln("    arr.cap = len;")
	g.writeln("    return arr;")
	g.writeln("}")
	g.writeln("")
	g.writeln("carv_float_array carv_new_float_array(carv_int len) {")
	g.writeln("    carv_float_array arr;")
	g.writeln("    arr.data = (carv_float*)carv_arena_alloc(len * sizeof(carv_float));")
	g.writeln("    arr.len = len;")
	g.writeln("    arr.cap = len;")
	g.writeln("    return arr;")
	g.writeln("}")
	g.writeln("")
	g.writeln("carv_string_array carv_new_string_array(carv_int len) {")
	g.writeln("    carv_string_array arr;")
	g.writeln("    arr.data = (carv_string*)carv_arena_alloc(len * sizeof(carv_string));")
	g.writeln("    arr.len = len;")
	g.writeln("    arr.cap = len;")
	g.writeln("    return arr;")
	g.writeln("}")
	g.writeln("")
	g.writeln("void carv_print_int(carv_int x) { printf(\"%lld\\n\", x); }")
	g.writeln("void carv_print_float(carv_float x) { printf(\"%g\\n\", x); }")
	g.writeln("void carv_print_bool(carv_bool x) { printf(\"%s\\n\", x ? \"true\" : \"false\"); }")
	g.writeln("void carv_print_string(carv_string x) { printf(\"%s\\n\", x.data); }")
	g.writeln("")
	g.writeln("void carv_print_int_array(carv_int_array arr) {")
	g.writeln("    printf(\"[\");")
	g.writeln("    for (carv_int i = 0; i < arr.len; i++) {")
	g.writeln("        if (i > 0) printf(\", \");")
	g.writeln("        printf(\"%lld\", arr.data[i]);")
	g.writeln("    }")
	g.writeln("    printf(\"]\");")
	g.writeln("}")
	g.writeln("")
	g.writeln("void carv_print_float_array(carv_float_array arr) {")
	g.writeln("    printf(\"[\");")
	g.writeln("    for (carv_int i = 0; i < arr.len; i++) {")
	g.writeln("        if (i > 0) printf(\", \");")
	g.writeln("        printf(\"%g\", arr.data[i]);")
	g.writeln("    }")
	g.writeln("    printf(\"]\");")
	g.writeln("}")
	g.writeln("")
	g.writeln("void carv_print_string_array(carv_string_array arr) {")
	g.writeln("    printf(\"[\");")
	g.writeln("    for (carv_int i = 0; i < arr.len; i++) {")
	g.writeln("        if (i > 0) printf(\", \");")
	g.writeln("        printf(\"%s\", arr.data[i].data);")
	g.writeln("    }")
	g.writeln("    printf(\"]\");")
	g.writeln("}")
	g.writeln("")
	g.writeln("void carv_print_bool_array(carv_bool_array arr) {")
	g.writeln("    printf(\"[\");")
	g.writeln("    for (carv_int i = 0; i < arr.len; i++) {")
	g.writeln("        if (i > 0) printf(\", \");")
	g.writeln("        printf(\"%s\", arr.data[i] ? \"true\" : \"false\");")
	g.writeln("    }")
	g.writeln("    printf(\"]\");")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_string carv_read_file(carv_string path) {")
	g.writeln("    FILE* f = fopen(path.data, \"rb\");")
	g.writeln("    if (!f) return (carv_string){NULL, 0, false};")
	g.writeln("    fseek(f, 0, SEEK_END);")
	g.writeln("    long len = ftell(f);")
	g.writeln("    fseek(f, 0, SEEK_SET);")
	g.writeln("    char* buf = (char*)carv_arena_alloc(len + 1);")
	g.writeln("    size_t rd = fread(buf, 1, len, f);")
	g.writeln("    buf[rd] = '\\0';")
	g.writeln("    fclose(f);")
	g.writeln("    return carv_string_own(buf, rd);")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_bool carv_write_file(carv_string path, carv_string content) {")
	g.writeln("    FILE* f = fopen(path.data, \"wb\");")
	g.writeln("    if (!f) return false;")
	g.writeln("    size_t written = fwrite(content.data, 1, content.len, f);")
	g.writeln("    fclose(f);")
	g.writeln("    return written == content.len;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_bool carv_file_exists(carv_string path) {")
	g.writeln("    FILE* f = fopen(path.data, \"r\");")
	g.writeln("    if (f) { fclose(f); return true; }")
	g.writeln("    return false;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_int carv_tcp_listen(carv_string host, carv_int port) {")
	g.writeln("    int fd = socket(AF_INET, SOCK_STREAM, 0);")
	g.writeln("    if (fd < 0) return -1;")
	g.writeln("    int opt = 1;")
	g.writeln("    setsockopt(fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));")
	g.writeln("    struct sockaddr_in addr;")
	g.writeln("    memset(&addr, 0, sizeof(addr));")
	g.writeln("    addr.sin_family = AF_INET;")
	g.writeln("    addr.sin_port = htons((uint16_t)port);")
	g.writeln("    if (!host.data || host.len == 0 || strcmp(host.data, \"0.0.0.0\") == 0) {")
	g.writeln("        addr.sin_addr.s_addr = INADDR_ANY;")
	g.writeln("    } else if (inet_pton(AF_INET, host.data, &addr.sin_addr) <= 0) {")
	g.writeln("        close(fd);")
	g.writeln("        return -1;")
	g.writeln("    }")
	g.writeln("    if (bind(fd, (struct sockaddr*)&addr, sizeof(addr)) < 0) {")
	g.writeln("        close(fd);")
	g.writeln("        return -1;")
	g.writeln("    }")
	g.writeln("    if (listen(fd, 16) < 0) {")
	g.writeln("        close(fd);")
	g.writeln("        return -1;")
	g.writeln("    }")
	g.writeln("    return fd;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_int carv_tcp_accept(carv_int listener_fd) {")
	g.writeln("    int conn_fd = accept((int)listener_fd, NULL, NULL);")
	g.writeln("    if (conn_fd < 0) return -1;")
	g.writeln("    return conn_fd;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_string carv_tcp_read(carv_int conn_fd, carv_int max_bytes) {")
	g.writeln("    if (max_bytes <= 0) return carv_strdup_str(\"\");")
	g.writeln("    char* buf = (char*)carv_arena_alloc((size_t)max_bytes + 1);")
	g.writeln("    ssize_t n = recv((int)conn_fd, buf, (size_t)max_bytes, 0);")
	g.writeln("    if (n <= 0) {")
	g.writeln("        buf[0] = '\\0';")
	g.writeln("        return carv_string_own(buf, 0);")
	g.writeln("    }")
	g.writeln("    buf[n] = '\\0';")
	g.writeln("    return carv_string_own(buf, (size_t)n);")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_int carv_tcp_write(carv_int conn_fd, carv_string data) {")
	g.writeln("    if (!data.data || data.len == 0) return 0;")
	g.writeln("    ssize_t n = send((int)conn_fd, data.data, data.len, 0);")
	g.writeln("    if (n < 0) return -1;")
	g.writeln("    return (carv_int)n;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_bool carv_tcp_close(carv_int fd) {")
	g.writeln("    return close((int)fd) == 0;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_string_array carv_split(carv_string str, carv_string sep) {")
	g.writeln("    carv_string_array arr = {NULL, 0, 0};")
	g.writeln("    if (!str.data || !sep.data) return arr;")
	g.writeln("    size_t sep_len = sep.len;")
	g.writeln("    if (sep_len == 0) {")
	g.writeln("        arr = carv_new_string_array(1);")
	g.writeln("        arr.data[0] = carv_string_clone(str);")
	g.writeln("        return arr;")
	g.writeln("    }")
	g.writeln("    // Count occurrences")
	g.writeln("    carv_int count = 1;")
	g.writeln("    char* p = str.data;")
	g.writeln("    while ((p = strstr(p, sep.data)) != NULL) { count++; p += sep_len; }")
	g.writeln("    arr = carv_new_string_array(count);")
	g.writeln("    // Split")
	g.writeln("    char* start = str.data;")
	g.writeln("    carv_int idx = 0;")
	g.writeln("    while ((p = strstr(start, sep.data)) != NULL) {")
	g.writeln("        size_t part_len = p - start;")
	g.writeln("        char* part = (char*)carv_arena_alloc(part_len + 1);")
	g.writeln("        memcpy(part, start, part_len);")
	g.writeln("        part[part_len] = '\\0';")
	g.writeln("        arr.data[idx] = carv_string_own(part, part_len);")
	g.writeln("        idx++;")
	g.writeln("        start = p + sep_len;")
	g.writeln("    }")
	g.writeln("    size_t tail_len = (size_t)(str.data + str.len - start);")
	g.writeln("    char* tail = (char*)carv_arena_alloc(tail_len + 1);")
	g.writeln("    memcpy(tail, start, tail_len);")
	g.writeln("    tail[tail_len] = '\\0';")
	g.writeln("    arr.data[idx] = carv_string_own(tail, tail_len);")
	g.writeln("    return arr;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_string carv_join(carv_string_array arr, carv_string sep) {")
	g.writeln("    if (arr.len == 0) return carv_strdup_str(\"\");")
	g.writeln("    size_t sep_len = sep.data ? sep.len : 0;")
	g.writeln("    size_t total_len = 0;")
	g.writeln("    for (carv_int i = 0; i < arr.len; i++) {")
	g.writeln("        if (arr.data[i].data) total_len += arr.data[i].len;")
	g.writeln("    }")
	g.writeln("    if (arr.len > 0) total_len += sep_len * (size_t)(arr.len - 1);")
	g.writeln("    char* result = (char*)carv_arena_alloc(total_len + 1);")
	g.writeln("    result[0] = '\\0';")
	g.writeln("    for (carv_int i = 0; i < arr.len; i++) {")
	g.writeln("        if (i > 0 && sep.data) strcat(result, sep.data);")
	g.writeln("        if (arr.data[i].data) strcat(result, arr.data[i].data);")
	g.writeln("    }")
	g.writeln("    return carv_string_own(result, total_len);")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_string carv_trim(carv_string str) {")
	g.writeln("    if (!str.data) return carv_strdup_str(\"\");")
	g.writeln("    char* start = str.data;")
	g.writeln("    char* end = str.data + str.len;")
	g.writeln("    while (start < end && (*start == ' ' || *start == '\\t' || *start == '\\n' || *start == '\\r')) start++;")
	g.writeln("    if (start == end) return carv_strdup_str(\"\");")
	g.writeln("    char* last = end - 1;")
	g.writeln("    while (last > start && (*last == ' ' || *last == '\\t' || *last == '\\n' || *last == '\\r')) last--;")
	g.writeln("    size_t len = (size_t)(last - start + 1);")
	g.writeln("    char* result = (char*)carv_arena_alloc(len + 1);")
	g.writeln("    memcpy(result, start, len);")
	g.writeln("    result[len] = '\\0';")
	g.writeln("    return carv_string_own(result, len);")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_string carv_substr(carv_string str, carv_int start, carv_int end) {")
	g.writeln("    if (!str.data) return carv_strdup_str(\"\");")
	g.writeln("    size_t str_len = str.len;")
	g.writeln("    if (start < 0) start = 0;")
	g.writeln("    if (end < 0) end = (carv_int)str_len;")
	g.writeln("    if ((size_t)start >= str_len) return carv_strdup_str(\"\");")
	g.writeln("    if ((size_t)end > str_len) end = (carv_int)str_len;")
	g.writeln("    if (end <= start) return carv_strdup_str(\"\");")
	g.writeln("    size_t len = (size_t)(end - start);")
	g.writeln("    char* result = (char*)carv_arena_alloc(len + 1);")
	g.writeln("    memcpy(result, str.data + start, len);")
	g.writeln("    result[len] = '\\0';")
	g.writeln("    return carv_string_own(result, len);")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_string carv_int_to_string(carv_int val) {")
	g.writeln("    char* buf = (char*)carv_arena_alloc(32);")
	g.writeln("    int len = snprintf(buf, 32, \"%lld\", val);")
	g.writeln("    return carv_string_own(buf, len);")
	g.writeln("}")
	g.writeln("")
	g.writeln("carv_string carv_float_to_string(carv_float val) {")
	g.writeln("    char* buf = (char*)carv_arena_alloc(64);")
	g.writeln("    int len = snprintf(buf, 64, \"%g\", val);")
	g.writeln("    return carv_string_own(buf, len);")
	g.writeln("}")
	g.writeln("")
	g.writeln("carv_string carv_bool_to_string(carv_bool val) {")
	g.writeln("    return carv_strdup_str(val ? \"true\" : \"false\");")
	g.writeln("}")
	g.writeln("")
	g.writeln("carv_string carv_concat(carv_string a, carv_string b) {")
	g.writeln("    size_t total = a.len + b.len;")
	g.writeln("    char* result = (char*)carv_arena_alloc(total + 1);")
	g.writeln("    memcpy(result, a.data, a.len);")
	g.writeln("    memcpy(result + a.len, b.data, b.len + 1);")
	g.writeln("    return carv_string_own(result, total);")
	g.writeln("}")
	g.writeln("")

	g.writeln("typedef enum { CARV_TYPE_INT, CARV_TYPE_FLOAT, CARV_TYPE_BOOL, CARV_TYPE_STRING } carv_type_tag;")
	g.writeln("typedef struct { carv_bool is_ok; carv_type_tag ok_tag; carv_type_tag err_tag; union { carv_int ok_int; carv_float ok_float; carv_bool ok_bool; carv_string ok_str; void* ok_ptr; } ok; union { carv_string err_str; carv_int err_code; } err; } carv_result;")
	g.writeln("")
	g.writeln("carv_result carv_ok_int(carv_int val) { carv_result r; r.is_ok = true; r.ok_tag = CARV_TYPE_INT; r.ok.ok_int = val; return r; }")
	g.writeln("carv_result carv_ok_float(carv_float val) { carv_result r; r.is_ok = true; r.ok_tag = CARV_TYPE_FLOAT; r.ok.ok_float = val; return r; }")
	g.writeln("carv_result carv_ok_bool(carv_bool val) { carv_result r; r.is_ok = true; r.ok_tag = CARV_TYPE_BOOL; r.ok.ok_bool = val; return r; }")
	g.writeln("carv_result carv_ok_str(carv_string val) { carv_result r; r.is_ok = true; r.ok_tag = CARV_TYPE_STRING; r.ok.ok_str = val; return r; }")
	g.writeln("carv_result carv_err_str(carv_string val) { carv_result r; r.is_ok = false; r.err_tag = CARV_TYPE_STRING; r.err.err_str = val; return r; }")
	g.writeln("carv_result carv_err_code(carv_int val) { carv_result r; r.is_ok = false; r.err_tag = CARV_TYPE_INT; r.err.err_code = val; return r; }")
	g.writeln("")

	if g.hasAsync {
		g.emitEventLoopRuntime()
	}
}

func (g *CGenerator) emitEventLoopRuntime() {
	g.writeln("typedef struct carv_loop carv_loop;")
	g.writeln("typedef struct carv_task {")
	g.writeln("    bool (*poll)(void*, carv_loop*);")
	g.writeln("    void (*drop)(void*);")
	g.writeln("    void* frame;")
	g.writeln("} carv_task;")
	g.writeln("")
	g.writeln("struct carv_loop {")
	g.writeln("    carv_task** ready;")
	g.writeln("    int ready_count;")
	g.writeln("    int ready_cap;")
	g.writeln("};")
	g.writeln("")
	g.writeln("static void carv_loop_init(carv_loop* loop) {")
	g.writeln("    loop->ready = NULL;")
	g.writeln("    loop->ready_count = 0;")
	g.writeln("    loop->ready_cap = 0;")
	g.writeln("}")
	g.writeln("")
	g.writeln("static void carv_loop_add_task(carv_loop* loop, carv_task* task) {")
	g.writeln("    if (loop->ready_count >= loop->ready_cap) {")
	g.writeln("        int newcap = loop->ready_cap == 0 ? 4 : loop->ready_cap * 2;")
	g.writeln("        loop->ready = (carv_task**)realloc(loop->ready, newcap * sizeof(carv_task*));")
	g.writeln("        loop->ready_cap = newcap;")
	g.writeln("    }")
	g.writeln("    loop->ready[loop->ready_count++] = task;")
	g.writeln("}")
	g.writeln("")
	g.writeln("static void carv_loop_run(carv_loop* loop) {")
	g.writeln("    while (loop->ready_count > 0) {")
	g.writeln("        for (int i = 0; i < loop->ready_count; ) {")
	g.writeln("            carv_task* t = loop->ready[i];")
	g.writeln("            if (t->poll(t->frame, loop)) {")
	g.writeln("                if (t->drop) t->drop(t->frame);")
	g.writeln("                loop->ready[i] = loop->ready[--loop->ready_count];")
	g.writeln("            } else {")
	g.writeln("                i++;")
	g.writeln("            }")
	g.writeln("        }")
	g.writeln("    }")
	g.writeln("    if (loop->ready) free(loop->ready);")
	g.writeln("}")
	g.writeln("")
}

func (g *CGenerator) generateFunctionDecl(fn *ast.FunctionStatement) {
	fnName := g.safeName(fn.Name.Value)
	params := g.paramsToC(fn.Parameters)
	if fn.Async {
		frameName := fnName + "_frame"
		g.writeln(fmt.Sprintf("typedef struct %s %s;", frameName, frameName))
		g.writeln(fmt.Sprintf("%s* %s(%s);", frameName, fnName, params))
		return
	}
	retType := g.inferFunctionReturnType(fn)
	g.writeln(fmt.Sprintf("%s %s(%s);", retType, fnName, params))
}

func (g *CGenerator) inferFunctionReturnType(fn *ast.FunctionStatement) string {
	if fn.ReturnType != nil {
		return g.typeToC(fn.ReturnType)
	}
	if g.functionReturnsResult(fn.Body) {
		return "carv_result"
	}
	retType := g.inferReturnTypeFromBody(fn.Body)
	if retType != "" {
		return retType
	}
	return "void"
}

func (g *CGenerator) functionReturnsResult(body *ast.BlockStatement) bool {
	for _, stmt := range body.Statements {
		if ret, ok := stmt.(*ast.ReturnStatement); ok && ret.ReturnValue != nil {
			switch ret.ReturnValue.(type) {
			case *ast.OkExpression, *ast.ErrExpression:
				return true
			}
		}
	}
	return false
}

func (g *CGenerator) inferReturnTypeFromBody(body *ast.BlockStatement) string {
	for _, stmt := range body.Statements {
		if ret, ok := stmt.(*ast.ReturnStatement); ok && ret.ReturnValue != nil {
			return g.resolveType(ret.ReturnValue)
		}
	}
	return ""
}

func (g *CGenerator) safeName(name string) string {
	reserved := map[string]bool{
		"double": true, "float": true, "int": true, "char": true,
		"void": true, "long": true, "short": true, "auto": true,
		"break": true, "case": true, "const": true, "continue": true,
		"default": true, "do": true, "else": true, "enum": true,
		"extern": true, "for": true, "goto": true, "if": true,
		"register": true, "return": true, "signed": true, "sizeof": true,
		"static": true, "struct": true, "switch": true, "typedef": true,
		"union": true, "unsigned": true, "volatile": true, "while": true,
		"main": true,
	}
	if reserved[name] {
		return "carv_" + name
	}
	return name
}

func (g *CGenerator) generateAsyncFunction(fn *ast.FunctionStatement) {
	fnName := g.safeName(fn.Name.Value)
	info := g.asyncFns[fn.Name.Value]
	retType := info.ReturnType
	frameName := fnName + "_frame"
	pollName := fnName + "_poll"

	var frameDef strings.Builder
	frameDef.WriteString(fmt.Sprintf("struct %s {\n    int __state;\n", frameName))
	for _, p := range info.Params {
		frameDef.WriteString(fmt.Sprintf("    %s %s;\n", p.CType, p.Name))
	}
	for _, l := range info.Locals {
		frameDef.WriteString(fmt.Sprintf("    %s %s;\n", l.CType, l.Name))
	}
	if retType != "void" {
		frameDef.WriteString(fmt.Sprintf("    %s __result;\n", retType))
	}
	frameDef.WriteString("    void* __sub_future;\n")
	frameDef.WriteString("};\n")

	var pollFn strings.Builder
	pollFn.WriteString(fmt.Sprintf("static bool %s(void* __raw_frame, carv_loop* __loop) {\n", pollName))
	pollFn.WriteString(fmt.Sprintf("    %s* f = (%s*)__raw_frame;\n", frameName, frameName))
	pollFn.WriteString("    switch (f->__state) {\n")
	pollFn.WriteString("    case 0:\n")

	oldOutput := g.output
	oldIndent := g.indent
	oldInFunction := g.inFunction
	oldFuncRetType := g.funcRetType

	g.output = strings.Builder{}
	g.indent = 2
	g.inFunction = true
	g.funcRetType = retType
	g.inAsyncFn = true
	g.asyncFnName = fn.Name.Value
	g.asyncStateID = 0
	g.enterScope()

	for _, p := range fn.Parameters {
		pType := g.typeToC(p.Type)
		g.declareVar(p.Name.Value, pType, false, false)
	}
	for _, l := range info.Locals {
		g.declareVar(l.Name, l.CType, true, false)
	}

	for _, stmt := range fn.Body.Statements {
		g.generateAsyncStatement(stmt)
	}

	if retType != "void" {
		g.writeln("return true;")
	} else {
		g.writeln("return true;")
	}

	pollFn.WriteString(g.output.String())
	pollFn.WriteString("    }\n    return true;\n}\n")

	g.exitScope()
	g.inAsyncFn = false

	g.output = oldOutput
	g.indent = oldIndent
	g.inFunction = oldInFunction
	g.funcRetType = oldFuncRetType

	g.writeln(frameDef.String())
	g.writeln(pollFn.String())

	frameType := frameName + "*"
	g.writeln(fmt.Sprintf("%s %s(%s) {", frameType, fnName, g.paramsToC(fn.Parameters)))
	g.indent++
	g.writeln(fmt.Sprintf("%s* f = (%s*)carv_arena_alloc(sizeof(%s));", frameName, frameName, frameName))
	g.writeln("f->__state = 0;")
	for _, p := range fn.Parameters {
		pName := p.Name.Value
		g.writeln(fmt.Sprintf("f->%s = %s;", pName, pName))
	}
	g.writeln("return f;")
	g.indent--
	g.writeln("}")
	g.writeln("")
}

func (g *CGenerator) generateAsyncStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.LetStatement:
		value := g.generateAsyncExpression(s.Value)
		g.flushPreamble()
		g.writeln(fmt.Sprintf("f->%s = %s;", s.Name.Value, value))
	case *ast.ReturnStatement:
		if s.ReturnValue != nil {
			value := g.generateAsyncExpression(s.ReturnValue)
			g.flushPreamble()
			g.writeln(fmt.Sprintf("f->__result = %s;", value))
		}
		g.writeln("return true;")
	case *ast.ExpressionStatement:
		if ifExpr, ok := s.Expression.(*ast.IfExpression); ok {
			g.generateAsyncIfStatement(ifExpr)
		} else {
			expr := g.generateAsyncExpression(s.Expression)
			g.flushPreamble()
			if expr != "" {
				g.writeln(expr + ";")
			}
		}
	default:
		g.generateStatement(stmt)
	}
}

func (g *CGenerator) generateAsyncIfStatement(e *ast.IfExpression) {
	cond := g.generateAsyncExpression(e.Condition)
	g.flushPreamble()
	g.writeln(fmt.Sprintf("if (%s) {", cond))
	g.indent++
	for _, stmt := range e.Consequence.Statements {
		g.generateAsyncStatement(stmt)
	}
	g.indent--
	if e.Alternative != nil {
		g.writeln("} else {")
		g.indent++
		for _, stmt := range e.Alternative.Statements {
			g.generateAsyncStatement(stmt)
		}
		g.indent--
	}
	g.writeln("}")
}

func (g *CGenerator) generateAsyncExpression(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return g.generateExpression(expr)
	case *ast.AwaitExpression:
		return g.generateAwaitExpression(e)
	default:
		return g.generateExpression(expr)
	}
}

func (g *CGenerator) asyncFrameVarRef(name string) (string, bool) {
	if !g.inAsyncFn {
		return "", false
	}
	info := g.asyncFns[g.asyncFnName]
	if info == nil {
		return "", false
	}
	for _, p := range info.Params {
		if p.Name == name {
			return fmt.Sprintf("f->%s", name), true
		}
	}
	for _, l := range info.Locals {
		if l.Name == name {
			return fmt.Sprintf("f->%s", name), true
		}
	}
	return "", false
}

func (g *CGenerator) generateAwaitExpression(e *ast.AwaitExpression) string {
	subFuture := g.generateAsyncExpression(e.Value)
	g.flushPreamble()

	g.asyncStateID++
	nextState := g.asyncStateID

	g.writeln(fmt.Sprintf("f->__sub_future = %s;", subFuture))
	g.writeln(fmt.Sprintf("f->__state = %d;", nextState))
	g.writeln("return false;")
	g.writeln(fmt.Sprintf("case %d:", nextState))

	if call, ok := e.Value.(*ast.CallExpression); ok {
		if ident, ok := call.Function.(*ast.Identifier); ok {
			subFnName := g.safeName(ident.Value)
			pollFn := subFnName + "_poll"
			frameTy := subFnName + "_frame"
			g.writeln(fmt.Sprintf("if (!%s(f->__sub_future, __loop)) return false;", pollFn))
			return fmt.Sprintf("((%s*)f->__sub_future)->__result", frameTy)
		}
	}

	return "0"
}

func (g *CGenerator) generateFunction(fn *ast.FunctionStatement) {
	if fn.Async {
		g.generateAsyncFunction(fn)
		return
	}

	retType := g.inferFunctionReturnType(fn)
	params := g.paramsToC(fn.Parameters)
	fnName := g.safeName(fn.Name.Value)
	g.writeln(fmt.Sprintf("%s %s(%s) {", retType, fnName, params))
	g.indent++
	g.enterScope()

	g.inFunction = true
	g.funcRetType = retType

	if retType != "void" {
		g.writeln(fmt.Sprintf("%s __carv_retval = %s;", retType, g.zeroValue(retType)))
	}

	for _, p := range fn.Parameters {
		pType := g.typeToC(p.Type)
		g.declareVar(p.Name.Value, pType, false, false)
	}

	for _, stmt := range fn.Body.Statements {
		g.generateStatement(stmt)
	}

	g.writeln("__carv_exit:;")
	g.emitScopeDrops()
	if retType != "void" {
		g.writeln("return __carv_retval;")
	}

	g.exitScope()
	g.inFunction = false
	g.indent--
	g.writeln("}")
	g.writeln("")
}

func (g *CGenerator) analyzeCaptured(fn *ast.FunctionLiteral) []capturedVar {
	paramSet := make(map[string]bool)
	for _, p := range fn.Parameters {
		paramSet[p.Name.Value] = true
	}

	seen := make(map[string]bool)
	var captures []capturedVar
	g.walkForCaptures(fn.Body, paramSet, seen, &captures)
	return captures
}

func (g *CGenerator) walkForCaptures(node ast.Node, params map[string]bool, seen map[string]bool, captures *[]capturedVar) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *ast.Identifier:
		name := n.Value
		if params[name] || seen[name] {
			return
		}
		if v := g.lookupVar(name); v != nil {
			seen[name] = true
			*captures = append(*captures, capturedVar{Name: name, CType: v.CType})
		}
	case *ast.BlockStatement:
		for _, stmt := range n.Statements {
			g.walkForCaptures(stmt, params, seen, captures)
		}
	case *ast.LetStatement:
		g.walkForCaptures(n.Value, params, seen, captures)
	case *ast.ConstStatement:
		g.walkForCaptures(n.Value, params, seen, captures)
	case *ast.ReturnStatement:
		if n.ReturnValue != nil {
			g.walkForCaptures(n.ReturnValue, params, seen, captures)
		}
	case *ast.ExpressionStatement:
		g.walkForCaptures(n.Expression, params, seen, captures)
	case *ast.ForStatement:
		g.walkForCaptures(n.Init, params, seen, captures)
		g.walkForCaptures(n.Condition, params, seen, captures)
		g.walkForCaptures(n.Post, params, seen, captures)
		g.walkForCaptures(n.Body, params, seen, captures)
	case *ast.ForInStatement:
		g.walkForCaptures(n.Iterable, params, seen, captures)
		g.walkForCaptures(n.Body, params, seen, captures)
	case *ast.WhileStatement:
		g.walkForCaptures(n.Condition, params, seen, captures)
		g.walkForCaptures(n.Body, params, seen, captures)
	case *ast.InfixExpression:
		g.walkForCaptures(n.Left, params, seen, captures)
		g.walkForCaptures(n.Right, params, seen, captures)
	case *ast.PrefixExpression:
		g.walkForCaptures(n.Right, params, seen, captures)
	case *ast.CallExpression:
		g.walkForCaptures(n.Function, params, seen, captures)
		for _, arg := range n.Arguments {
			g.walkForCaptures(arg, params, seen, captures)
		}
	case *ast.MemberExpression:
		g.walkForCaptures(n.Object, params, seen, captures)
	case *ast.IndexExpression:
		g.walkForCaptures(n.Left, params, seen, captures)
		g.walkForCaptures(n.Index, params, seen, captures)
	case *ast.AssignExpression:
		g.walkForCaptures(n.Left, params, seen, captures)
		g.walkForCaptures(n.Right, params, seen, captures)
	case *ast.IfExpression:
		g.walkForCaptures(n.Condition, params, seen, captures)
		g.walkForCaptures(n.Consequence, params, seen, captures)
		g.walkForCaptures(n.Alternative, params, seen, captures)
	case *ast.PipeExpression:
		g.walkForCaptures(n.Left, params, seen, captures)
		g.walkForCaptures(n.Right, params, seen, captures)
	case *ast.ArrayLiteral:
		for _, elem := range n.Elements {
			g.walkForCaptures(elem, params, seen, captures)
		}
	case *ast.BorrowExpression:
		g.walkForCaptures(n.Value, params, seen, captures)
	case *ast.DerefExpression:
		g.walkForCaptures(n.Value, params, seen, captures)
	case *ast.FunctionLiteral:
		// Don't descend into nested closures
	}
}

func (g *CGenerator) generateClosureExpression(fn *ast.FunctionLiteral) string {
	id := g.nextClosureID()
	captures := g.analyzeCaptured(fn)

	envName := fmt.Sprintf("__closure_%d_env", id)
	fnName := fmt.Sprintf("__closure_%d_fn", id)
	closureType := fmt.Sprintf("__closure_%d", id)

	retType := g.closureReturnType(fn)
	paramTypes := g.closureParamTypes(fn)

	// Emit env struct
	var envDef strings.Builder
	envDef.WriteString("typedef struct { ")
	for _, c := range captures {
		envDef.WriteString(fmt.Sprintf("%s %s; ", c.CType, c.Name))
	}
	envDef.WriteString(fmt.Sprintf("} %s;", envName))
	g.closureDefs = append(g.closureDefs, envDef.String())

	// Emit fn_ptr typedef (fat pointer)
	var fnPtrSig strings.Builder
	fnPtrSig.WriteString("void*")
	for _, pt := range paramTypes {
		fnPtrSig.WriteString(", " + pt)
	}
	closureTypeDef := fmt.Sprintf("typedef struct { void* env; %s (*fn_ptr)(%s); } %s;",
		retType, fnPtrSig.String(), closureType)
	g.closureDefs = append(g.closureDefs, closureTypeDef)

	// Emit lambda-lifted function
	var liftedFn strings.Builder
	liftedFn.WriteString(fmt.Sprintf("static %s %s(%s* __env", retType, fnName, envName))
	for _, p := range fn.Parameters {
		pType := g.typeToC(p.Type)
		liftedFn.WriteString(fmt.Sprintf(", %s %s", pType, p.Name.Value))
	}
	liftedFn.WriteString(") {\n")

	oldOutput := g.output
	oldIndent := g.indent
	oldInFunction := g.inFunction
	oldFuncRetType := g.funcRetType
	oldCaptureMap := g.captureMap

	g.output = strings.Builder{}
	g.indent = 1
	g.inFunction = true
	g.funcRetType = retType
	g.enterScope()

	g.captureMap = make(map[string]string)
	for _, c := range captures {
		g.captureMap[c.Name] = fmt.Sprintf("__env->%s", c.Name)
	}
	for _, p := range fn.Parameters {
		pType := g.typeToC(p.Type)
		g.declareVar(p.Name.Value, pType, p.Mutable, false)
	}

	if retType != "void" {
		g.writeln(fmt.Sprintf("%s __carv_retval = %s;", retType, g.zeroValue(retType)))
	}

	for _, stmt := range fn.Body.Statements {
		g.generateStatement(stmt)
	}

	g.writeln("__carv_exit:;")
	g.emitScopeDrops()
	if retType != "void" {
		g.writeln("return __carv_retval;")
	}

	g.exitScope()
	g.captureMap = oldCaptureMap

	liftedFn.WriteString(g.output.String())
	liftedFn.WriteString("}\n")
	g.closureDefs = append(g.closureDefs, liftedFn.String())

	g.output = oldOutput
	g.indent = oldIndent
	g.inFunction = oldInFunction
	g.funcRetType = oldFuncRetType

	// At call site: allocate env, populate, build closure struct
	envVar := fmt.Sprintf("__env_%d", id)
	clVar := fmt.Sprintf("__cl_%d", id)

	g.writeln(fmt.Sprintf("%s* %s = (%s*)carv_arena_alloc(sizeof(%s));", envName, envVar, envName, envName))
	for _, c := range captures {
		if g.isMoveType(c.CType) {
			g.writeln(fmt.Sprintf("%s->%s = carv_string_move(&%s);", envVar, c.Name, c.Name))
		} else {
			g.writeln(fmt.Sprintf("%s->%s = %s;", envVar, c.Name, g.safeName(c.Name)))
		}
	}
	g.writeln(fmt.Sprintf("%s %s = { .env = %s, .fn_ptr = %s };", closureType, clVar, envVar, fnName))
	g.declareVar(clVar, closureType, false, false)
	g.lastClosureType = closureType

	return clVar
}

func (g *CGenerator) closureReturnType(fn *ast.FunctionLiteral) string {
	if fn.ReturnType != nil {
		return g.typeToC(fn.ReturnType)
	}
	return "void"
}

func (g *CGenerator) closureParamTypes(fn *ast.FunctionLiteral) []string {
	var pts []string
	for _, p := range fn.Parameters {
		pts = append(pts, g.typeToC(p.Type))
	}
	return pts
}

func (g *CGenerator) isMoveType(ctype string) bool {
	return ctype == "carv_string" || strings.HasSuffix(ctype, "_array") || strings.HasSuffix(ctype, "*")
}

func (g *CGenerator) generateClassDecl(cls *ast.ClassStatement) {
	className := cls.Name.Value
	g.writeln(fmt.Sprintf("typedef struct %s %s;", className, className))
	g.writeln(fmt.Sprintf("struct %s {", className))
	g.indent++

	for _, field := range cls.Fields {
		fieldType := g.typeToC(field.Type)
		g.writeln(fmt.Sprintf("%s %s;", fieldType, field.Name.Value))
	}

	g.indent--
	g.writeln("};")
	g.writeln("")

	g.writeln(fmt.Sprintf("%s* %s_new(void) {", className, className))
	g.indent++
	g.writeln(fmt.Sprintf("%s* self = (%s*)carv_arena_alloc(sizeof(%s));", className, className, className))
	for _, field := range cls.Fields {
		if field.Default != nil {
			defaultVal := g.generateExpression(field.Default)
			g.flushPreamble()
			g.writeln(fmt.Sprintf("self->%s = %s;", field.Name.Value, defaultVal))
		} else {
			fieldType := g.typeToC(field.Type)
			g.writeln(fmt.Sprintf("self->%s = %s;", field.Name.Value, g.zeroValue(fieldType)))
		}
	}
	g.writeln("return self;")
	g.indent--
	g.writeln("}")
	g.writeln("")
}

func (g *CGenerator) zeroValue(cType string) string {
	switch cType {
	case "carv_int":
		return "0"
	case "carv_float":
		return "0.0"
	case "carv_bool":
		return "false"
	case "carv_string":
		return "(carv_string){NULL, 0, false}"
	default:
		return "0"
	}
}

func (g *CGenerator) generateClassMethodDecls(cls *ast.ClassStatement) {
	className := cls.Name.Value
	for _, method := range cls.Methods {
		retType := g.typeToC(method.ReturnType)
		params := g.methodParamsToC(className, method.Receiver, method.Parameters)
		g.writeln(fmt.Sprintf("%s %s_%s(%s);", retType, className, method.Name.Value, params))
	}
}

func (g *CGenerator) methodParamsToC(className string, recv ast.ReceiverKind, params []*ast.Parameter) string {
	parts := []string{}
	switch recv {
	case ast.RecvRef:
		parts = append(parts, fmt.Sprintf("const %s* self", className))
	case ast.RecvMutRef, ast.RecvValue:
		parts = append(parts, fmt.Sprintf("%s* self", className))
	}
	for _, p := range params {
		pType := g.typeToC(p.Type)
		parts = append(parts, fmt.Sprintf("%s %s", pType, p.Name.Value))
	}
	return strings.Join(parts, ", ")
}

func (g *CGenerator) generateClassMethods(cls *ast.ClassStatement) {
	className := cls.Name.Value
	for _, method := range cls.Methods {
		retType := g.typeToC(method.ReturnType)
		params := g.methodParamsToC(className, method.Receiver, method.Parameters)
		g.writeln(fmt.Sprintf("%s %s_%s(%s) {", retType, className, method.Name.Value, params))
		g.indent++
		g.enterScope()

		g.inFunction = true
		g.funcRetType = retType

		if retType != "void" {
			g.writeln(fmt.Sprintf("%s __carv_retval = %s;", retType, g.zeroValue(retType)))
		}

		for _, stmt := range method.Body.Statements {
			g.generateStatement(stmt)
		}

		g.writeln("__carv_exit:;")
		g.emitScopeDrops()
		if retType != "void" {
			g.writeln("return __carv_retval;")
		}

		g.exitScope()
		g.inFunction = false
		g.indent--
		g.writeln("}")
		g.writeln("")
	}
}

func (g *CGenerator) generateStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.LetStatement:
		g.generateLetStatement(s)
	case *ast.ConstStatement:
		g.generateConstStatement(s)
	case *ast.ReturnStatement:
		g.generateReturnStatement(s)
	case *ast.ExpressionStatement:
		if ifExpr, ok := s.Expression.(*ast.IfExpression); ok {
			g.generateIfStatement(ifExpr)
		} else {
			g.generateExpressionStatement(s)
		}
	case *ast.ForStatement:
		g.generateForStatement(s)
	case *ast.ForInStatement:
		g.generateForInStatement(s)
	case *ast.WhileStatement:
		g.generateWhileStatement(s)
	case *ast.BreakStatement:
		g.writeln("break;")
	case *ast.ContinueStatement:
		g.writeln("continue;")
	case *ast.BlockStatement:
		g.generateBlockStatement(s)
	}
}

func (g *CGenerator) generateLetStatement(s *ast.LetStatement) {
	varType := g.inferType(s.Value)
	varName := s.Name.Value
	g.lastClosureType = ""
	value := g.generateExpression(s.Value)
	g.flushPreamble()

	if g.lastClosureType != "" {
		varType = g.lastClosureType
	}

	if arr, ok := s.Value.(*ast.ArrayLiteral); ok {
		varType = g.getArrayType(g.inferArrayElemType(s.Value))
		g.arrayLengths[varName] = len(arr.Elements)
	}

	isOwned := false
	if varType == "carv_string" || strings.HasSuffix(varType, "_array") || strings.HasSuffix(varType, "*") {
		isOwned = true
	}
	g.declareVar(varName, varType, s.Mutable, isOwned)

	if varType == "carv_result" {
		okType := g.inferResultOkType(s.Value)
		errType := g.inferResultErrType(s.Value)
		g.declareVar(varName+"_result_ok", okType, false, false)
		g.declareVar(varName+"_result_err", errType, false, false)
	}

	g.writeln(fmt.Sprintf("%s %s = %s;", varType, varName, value))
}

func (g *CGenerator) generateConstStatement(s *ast.ConstStatement) {
	varType := g.inferType(s.Value)
	varName := s.Name.Value
	value := g.generateExpression(s.Value)
	g.flushPreamble()
	g.writeln(fmt.Sprintf("const %s %s = %s;", varType, varName, value))
}

func (g *CGenerator) generateReturnStatement(s *ast.ReturnStatement) {
	if g.inFunction {
		if s.ReturnValue != nil {
			value := g.generateExpression(s.ReturnValue)
			g.flushPreamble()
			g.writeln(fmt.Sprintf("__carv_retval = %s;", value))
		}
		g.writeln("goto __carv_exit;")
		return
	}

	if s.ReturnValue == nil {
		g.writeln("return;")
		return
	}

	value := g.generateExpression(s.ReturnValue)
	g.flushPreamble()
	g.writeln(fmt.Sprintf("return %s;", value))
}

func (g *CGenerator) generateExpressionStatement(s *ast.ExpressionStatement) {
	expr := g.generateExpression(s.Expression)
	g.flushPreamble()
	if expr != "" {
		g.writeln(expr + ";")
	}
}

func (g *CGenerator) generateForStatement(s *ast.ForStatement) {
	init := ""
	if s.Init != nil {
		if let, ok := s.Init.(*ast.LetStatement); ok {
			varType := g.inferType(let.Value)
			value := g.generateExpression(let.Value)
			init = fmt.Sprintf("%s %s = %s", varType, let.Name.Value, value)
		}
	}
	g.flushPreamble()

	g.write("for (")
	if init != "" {
		g.writeRaw(init + "; ")
	} else {
		g.writeRaw("; ")
	}

	if s.Condition != nil {
		g.writeRaw(g.generateExpression(s.Condition))
	}
	g.writeRaw("; ")

	if s.Post != nil {
		if es, ok := s.Post.(*ast.ExpressionStatement); ok {
			g.writeRaw(g.generateExpression(es.Expression))
		}
	}

	g.writeRaw(") {\n")
	g.indent++
	g.enterScope()

	for _, stmt := range s.Body.Statements {
		g.generateStatement(stmt)
	}

	g.exitScope()
	g.indent--
	g.writeln("}")
}

func (g *CGenerator) generateForInStatement(s *ast.ForInStatement) {
	iterName := s.Value.Value
	iterableExpr := g.generateExpression(s.Iterable)
	g.flushPreamble()

	idxVar := fmt.Sprintf("__idx_%d", g.tempCounter)
	g.tempCounter++

	g.writeln(fmt.Sprintf("for (carv_int %s = 0; %s < %s.len; %s++) {", idxVar, idxVar, iterableExpr, idxVar))
	g.indent++
	g.enterScope()

	elemType := g.inferArrayElemType(s.Iterable)
	g.writeln(fmt.Sprintf("%s %s = %s.data[%s];", elemType, iterName, iterableExpr, idxVar))

	for _, stmt := range s.Body.Statements {
		g.generateStatement(stmt)
	}

	g.exitScope()
	g.indent--
	g.writeln("}")
}

func (g *CGenerator) generateWhileStatement(s *ast.WhileStatement) {
	cond := g.generateExpression(s.Condition)
	g.flushPreamble()
	g.writeln(fmt.Sprintf("while (%s) {", cond))
	g.indent++

	for _, stmt := range s.Body.Statements {
		g.generateStatement(stmt)
	}

	g.indent--
	g.writeln("}")
}

func (g *CGenerator) generateBlockStatement(s *ast.BlockStatement) {
	g.writeln("{")
	g.indent++
	g.enterScope()

	for _, stmt := range s.Statements {
		g.generateStatement(stmt)
	}

	g.exitScope()
	g.indent--
	g.writeln("}")
}

func (g *CGenerator) generateIfStatement(e *ast.IfExpression) {
	cond := g.generateExpression(e.Condition)
	g.flushPreamble()
	g.writeln(fmt.Sprintf("if (%s) {", cond))
	g.indent++

	for _, stmt := range e.Consequence.Statements {
		g.generateStatement(stmt)
	}

	g.indent--

	if e.Alternative != nil {
		g.writeln("} else {")
		g.indent++
		for _, stmt := range e.Alternative.Statements {
			g.generateStatement(stmt)
		}
		g.indent--
	}

	g.writeln("}")
}

func (g *CGenerator) generateExpression(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return fmt.Sprintf("%d", e.Value)
	case *ast.FloatLiteral:
		return fmt.Sprintf("%f", e.Value)
	case *ast.StringLiteral:
		return fmt.Sprintf("carv_string_lit(\"%s\")", g.escapeString(e.Value))
	case *ast.BoolLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case *ast.NilLiteral:
		return "NULL"
	case *ast.Identifier:
		if g.captureMap != nil {
			if mapped, ok := g.captureMap[e.Value]; ok {
				return mapped
			}
		}
		if mapped, ok := g.asyncFrameVarRef(e.Value); ok {
			return mapped
		}
		return g.safeName(e.Value)
	case *ast.ArrayLiteral:
		return g.generateArrayLiteral(e)
	case *ast.PrefixExpression:
		return g.generatePrefixExpression(e)
	case *ast.InfixExpression:
		return g.generateInfixExpression(e)
	case *ast.PipeExpression:
		return g.generatePipeExpression(e)
	case *ast.AssignExpression:
		return g.generateAssignExpression(e)
	case *ast.CallExpression:
		return g.generateCallExpression(e)
	case *ast.IfExpression:
		return g.generateIfExpression(e)
	case *ast.IndexExpression:
		return g.generateIndexExpression(e)
	case *ast.MemberExpression:
		return g.generateMemberExpression(e)
	case *ast.NewExpression:
		return g.generateNewExpression(e)
	case *ast.OkExpression:
		return g.generateOkExpression(e)
	case *ast.ErrExpression:
		return g.generateErrExpression(e)
	case *ast.TryExpression:
		return g.generateTryExpression(e)
	case *ast.MatchExpression:
		return g.generateMatchExpression(e)
	case *ast.InterpolatedString:
		return g.generateInterpolatedString(e)
	case *ast.BorrowExpression:
		inner := g.generateExpression(e.Value)
		return "(&" + inner + ")"
	case *ast.DerefExpression:
		inner := g.generateExpression(e.Value)
		return "(*" + inner + ")"
	case *ast.CastExpression:
		return g.generateCastExpression(e)
	case *ast.FunctionLiteral:
		return g.generateClosureExpression(e)
	}
	return ""
}

func (g *CGenerator) generatePrefixExpression(e *ast.PrefixExpression) string {
	right := g.generateExpression(e.Right)
	return fmt.Sprintf("(%s%s)", e.Operator, right)
}

func (g *CGenerator) generateInfixExpression(e *ast.InfixExpression) string {
	left := g.generateExpression(e.Left)
	right := g.generateExpression(e.Right)
	if e.Operator == "+" {
		leftType := g.resolveType(e.Left)
		rightType := g.resolveType(e.Right)
		if leftType == "carv_string" && rightType == "carv_string" {
			return fmt.Sprintf("carv_concat(%s, %s)", left, right)
		}
	}
	return fmt.Sprintf("(%s %s %s)", left, e.Operator, right)
}

func (g *CGenerator) generatePipeExpression(e *ast.PipeExpression) string {
	left := g.generateExpression(e.Left)
	leftType := g.resolveType(e.Left)

	switch right := e.Right.(type) {
	case *ast.Identifier:
		fnName := g.safeName(right.Value)
		if fnName == "print" || fnName == "println" {
			return g.generatePrintExpr(left, leftType)
		}
		if varType := g.getVarType(right.Value); strings.HasPrefix(varType, "__closure_") {
			return fmt.Sprintf("%s.fn_ptr(%s.env, %s)", fnName, fnName, left)
		}
		return fmt.Sprintf("%s(%s)", fnName, left)
	case *ast.CallExpression:
		if ident, ok := right.Function.(*ast.Identifier); ok {
			fnName := g.safeName(ident.Value)
			if fnName == "print" || fnName == "println" {
				return g.generatePrintExpr(left, leftType)
			}
			if varType := g.getVarType(ident.Value); strings.HasPrefix(varType, "__closure_") {
				args := []string{fnName + ".env", left}
				for _, arg := range right.Arguments {
					args = append(args, g.generateExpression(arg))
				}
				return fmt.Sprintf("%s.fn_ptr(%s)", fnName, strings.Join(args, ", "))
			}
			args := []string{left}
			for _, arg := range right.Arguments {
				args = append(args, g.generateExpression(arg))
			}
			return fmt.Sprintf("%s(%s)", fnName, strings.Join(args, ", "))
		}
		fn := g.generateExpression(right.Function)
		args := []string{left}
		for _, arg := range right.Arguments {
			args = append(args, g.generateExpression(arg))
		}
		return fmt.Sprintf("%s(%s)", fn, strings.Join(args, ", "))
	default:
		return left
	}
}

func (g *CGenerator) generatePrintExpr(val string, valType string) string {
	switch valType {
	case "carv_int":
		return fmt.Sprintf("(printf(\"%%lld\\n\", %s))", val)
	case "carv_float":
		return fmt.Sprintf("(printf(\"%%g\\n\", %s))", val)
	case "carv_bool":
		return fmt.Sprintf("(printf(\"%%s\\n\", %s ? \"true\" : \"false\"))", val)
	case "carv_string":
		return fmt.Sprintf("(printf(\"%%s\\n\", %s.data))", val)
	default:
		return fmt.Sprintf("(printf(\"%%lld\\n\", (carv_int)%s))", val)
	}
}

func (g *CGenerator) generateAssignExpression(e *ast.AssignExpression) string {
	left := g.generateExpression(e.Left)
	right := g.generateExpression(e.Right)

	switch e.Operator {
	case "=":
		return fmt.Sprintf("%s = %s", left, right)
	case "+=":
		return fmt.Sprintf("%s += %s", left, right)
	case "-=":
		return fmt.Sprintf("%s -= %s", left, right)
	case "*=":
		return fmt.Sprintf("%s *= %s", left, right)
	case "/=":
		return fmt.Sprintf("%s /= %s", left, right)
	}
	return fmt.Sprintf("%s = %s", left, right)
}

func (g *CGenerator) generateCallExpression(e *ast.CallExpression) string {
	if member, ok := e.Function.(*ast.MemberExpression); ok {
		if lowered, ok := g.generateBuiltinModuleCall(member, e.Arguments); ok {
			return lowered
		}
		return g.generateMethodCall(member, e.Arguments)
	}

	fn := g.generateExpression(e.Function)

	if fn == "print" || fn == "println" {
		return g.generatePrintCall(e)
	}

	if fn == "clone" && len(e.Arguments) == 1 {
		arg := g.generateExpression(e.Arguments[0])
		argType := g.resolveType(e.Arguments[0])
		switch argType {
		case "carv_string":
			return fmt.Sprintf("carv_string_clone(%s)", arg)
		default:
			return arg
		}
	}

	if fn == "len" && len(e.Arguments) == 1 {
		arg := g.generateExpression(e.Arguments[0])
		argType := g.resolveType(e.Arguments[0])
		if argType == "carv_string" {
			return fmt.Sprintf("(carv_int)%s.len", arg)
		}
		return fmt.Sprintf("%s.len", arg)
	}

	if fn == "read_file" && len(e.Arguments) == 1 {
		arg := g.generateExpression(e.Arguments[0])
		return fmt.Sprintf("carv_read_file(%s)", arg)
	}

	if fn == "write_file" && len(e.Arguments) == 2 {
		path := g.generateExpression(e.Arguments[0])
		content := g.generateExpression(e.Arguments[1])
		return fmt.Sprintf("carv_write_file(%s, %s)", path, content)
	}

	if fn == "file_exists" && len(e.Arguments) == 1 {
		arg := g.generateExpression(e.Arguments[0])
		return fmt.Sprintf("carv_file_exists(%s)", arg)
	}

	if fn == "tcp_listen" && len(e.Arguments) == 2 {
		host := g.generateExpression(e.Arguments[0])
		port := g.generateExpression(e.Arguments[1])
		return fmt.Sprintf("carv_tcp_listen(%s, %s)", host, port)
	}

	if fn == "tcp_accept" && len(e.Arguments) == 1 {
		listener := g.generateExpression(e.Arguments[0])
		return fmt.Sprintf("carv_tcp_accept(%s)", listener)
	}

	if fn == "tcp_read" && len(e.Arguments) == 2 {
		conn := g.generateExpression(e.Arguments[0])
		maxBytes := g.generateExpression(e.Arguments[1])
		return fmt.Sprintf("carv_tcp_read(%s, %s)", conn, maxBytes)
	}

	if fn == "tcp_write" && len(e.Arguments) == 2 {
		conn := g.generateExpression(e.Arguments[0])
		data := g.generateExpression(e.Arguments[1])
		return fmt.Sprintf("carv_tcp_write(%s, %s)", conn, data)
	}

	if fn == "tcp_close" && len(e.Arguments) == 1 {
		fd := g.generateExpression(e.Arguments[0])
		return fmt.Sprintf("carv_tcp_close(%s)", fd)
	}

	if fn == "split" && len(e.Arguments) == 2 {
		str := g.generateExpression(e.Arguments[0])
		sep := g.generateExpression(e.Arguments[1])
		return fmt.Sprintf("carv_split(%s, %s)", str, sep)
	}

	if fn == "join" && len(e.Arguments) == 2 {
		arr := g.generateExpression(e.Arguments[0])
		sep := g.generateExpression(e.Arguments[1])
		return fmt.Sprintf("carv_join(%s, %s)", arr, sep)
	}

	if fn == "trim" && len(e.Arguments) == 1 {
		arg := g.generateExpression(e.Arguments[0])
		return fmt.Sprintf("carv_trim(%s)", arg)
	}

	if fn == "substr" && len(e.Arguments) >= 2 {
		str := g.generateExpression(e.Arguments[0])
		start := g.generateExpression(e.Arguments[1])
		end := "-1"
		if len(e.Arguments) == 3 {
			end = g.generateExpression(e.Arguments[2])
		}
		return fmt.Sprintf("carv_substr(%s, %s, %s)", str, start, end)
	}

	if ident, ok := e.Function.(*ast.Identifier); ok {
		if varType := g.getVarType(ident.Value); strings.HasPrefix(varType, "__closure_") {
			clName := g.safeName(ident.Value)
			var callArgs []string
			callArgs = append(callArgs, clName+".env")
			for _, arg := range e.Arguments {
				callArgs = append(callArgs, g.generateExpression(arg))
			}
			return fmt.Sprintf("%s.fn_ptr(%s)", clName, strings.Join(callArgs, ", "))
		}
	}

	var args []string
	for _, arg := range e.Arguments {
		args = append(args, g.generateExpression(arg))
	}
	return fmt.Sprintf("%s(%s)", fn, strings.Join(args, ", "))
}

func (g *CGenerator) generateBuiltinModuleCall(member *ast.MemberExpression, args []ast.Expression) (string, bool) {
	ident, ok := member.Object.(*ast.Identifier)
	if !ok {
		return "", false
	}
	moduleName, ok := g.builtinAliases[ident.Value]
	if !ok || !module.IsBuiltinModule(moduleName) {
		return "", false
	}

	switch member.Member.Value {
	case "tcp_listen":
		if len(args) != 2 {
			return "", false
		}
		host := g.generateExpression(args[0])
		port := g.generateExpression(args[1])
		return fmt.Sprintf("carv_tcp_listen(%s, %s)", host, port), true
	case "tcp_accept":
		if len(args) != 1 {
			return "", false
		}
		listener := g.generateExpression(args[0])
		return fmt.Sprintf("carv_tcp_accept(%s)", listener), true
	case "tcp_read":
		if len(args) != 2 {
			return "", false
		}
		conn := g.generateExpression(args[0])
		maxBytes := g.generateExpression(args[1])
		return fmt.Sprintf("carv_tcp_read(%s, %s)", conn, maxBytes), true
	case "tcp_write":
		if len(args) != 2 {
			return "", false
		}
		conn := g.generateExpression(args[0])
		data := g.generateExpression(args[1])
		return fmt.Sprintf("carv_tcp_write(%s, %s)", conn, data), true
	case "tcp_close":
		if len(args) != 1 {
			return "", false
		}
		fd := g.generateExpression(args[0])
		return fmt.Sprintf("carv_tcp_close(%s)", fd), true
	default:
		return "", false
	}
}

func (g *CGenerator) generateMethodCall(member *ast.MemberExpression, args []ast.Expression) string {
	obj := g.generateExpression(member.Object)
	methodName := member.Member.Value
	if methodName == "clone" && len(args) == 0 {
		objType := g.resolveType(member.Object)
		switch objType {
		case "carv_string":
			return fmt.Sprintf("carv_string_clone(%s)", obj)
		default:
			return fmt.Sprintf("%s /* clone not yet implemented for %s */", obj, objType)
		}
	}

	objCType := g.resolveType(member.Object)
	if g.isInterfaceRefType(objCType) {
		var argStrs []string
		argStrs = append(argStrs, obj+".data")
		for _, arg := range args {
			argStrs = append(argStrs, g.generateExpression(arg))
		}
		return fmt.Sprintf("%s.vt->%s(%s)", obj, methodName, strings.Join(argStrs, ", "))
	}

	className := g.inferClassName(member.Object)
	if className == "" {
		className = "Unknown"
	}

	var argStrs []string
	argStrs = append(argStrs, obj)
	for _, arg := range args {
		argStrs = append(argStrs, g.generateExpression(arg))
	}

	return fmt.Sprintf("%s_%s(%s)", className, methodName, strings.Join(argStrs, ", "))
}

func (g *CGenerator) isInterfaceRefType(ctype string) bool {
	if strings.HasSuffix(ctype, "_ref") {
		name := strings.TrimSuffix(ctype, "_ref")
		if _, ok := g.interfaces[name]; ok {
			return true
		}
	}
	if strings.HasSuffix(ctype, "_mut_ref") {
		name := strings.TrimSuffix(ctype, "_mut_ref")
		if _, ok := g.interfaces[name]; ok {
			return true
		}
	}
	return false
}

func (g *CGenerator) inferClassName(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		t := g.getVarType(e.Value)
		return strings.TrimSuffix(t, "*")
	case *ast.NewExpression:
		if named, ok := e.Type.(*ast.NamedType); ok {
			return named.Name.Value
		}
	}
	return ""
}

func (g *CGenerator) generatePrintCall(e *ast.CallExpression) string {
	if len(e.Arguments) == 0 {
		return "printf(\"\\n\")"
	}

	var parts []string
	for i, arg := range e.Arguments {
		if i > 0 {
			parts = append(parts, "printf(\" \")")
		}

		argStr := g.generateExpression(arg)
		argType := g.resolveType(arg)

		if arr, ok := arg.(*ast.ArrayLiteral); ok {
			if len(arr.Elements) > 0 {
				elemType := g.resolveType(arr.Elements[0])
				parts = append(parts, g.generateArrayPrint(argStr, elemType))
			} else {
				parts = append(parts, fmt.Sprintf("carv_print_int_array(%s)", argStr))
			}
			continue
		}

		switch argType {
		case "carv_int_array":
			parts = append(parts, fmt.Sprintf("carv_print_int_array(%s)", argStr))
		case "carv_float_array":
			parts = append(parts, fmt.Sprintf("carv_print_float_array(%s)", argStr))
		case "carv_string_array":
			parts = append(parts, fmt.Sprintf("carv_print_string_array(%s)", argStr))
		case "carv_bool_array":
			parts = append(parts, fmt.Sprintf("carv_print_bool_array(%s)", argStr))
		case "carv_int":
			parts = append(parts, fmt.Sprintf("printf(\"%%lld\", %s)", argStr))
		case "carv_float":
			parts = append(parts, fmt.Sprintf("printf(\"%%g\", %s)", argStr))
		case "carv_bool":
			parts = append(parts, fmt.Sprintf("printf(\"%%s\", %s ? \"true\" : \"false\")", argStr))
		case "carv_string":
			parts = append(parts, fmt.Sprintf("printf(\"%%s\", %s.data)", argStr))
		default:
			parts = append(parts, fmt.Sprintf("printf(\"%%lld\", (carv_int)%s)", argStr))
		}
	}

	parts = append(parts, "printf(\"\\n\")")
	return "(" + strings.Join(parts, ", ") + ")"
}

func (g *CGenerator) generateArrayPrint(argStr string, elemType string) string {
	switch elemType {
	case "carv_int":
		return fmt.Sprintf("carv_print_int_array(%s)", argStr)
	case "carv_float":
		return fmt.Sprintf("carv_print_float_array(%s)", argStr)
	case "carv_string":
		return fmt.Sprintf("carv_print_string_array(%s)", argStr)
	case "carv_bool":
		return fmt.Sprintf("carv_print_bool_array(%s)", argStr)
	default:
		return fmt.Sprintf("carv_print_int_array(%s)", argStr)
	}
}

func (g *CGenerator) generateIfExpression(e *ast.IfExpression) string {
	cond := g.generateExpression(e.Condition)
	tempName := fmt.Sprintf("__if_%d", g.tempCounter)
	g.tempCounter++

	ifType := g.inferIfExprType(e)

	var consResult, altResult string
	if len(e.Consequence.Statements) > 0 {
		if last, ok := e.Consequence.Statements[len(e.Consequence.Statements)-1].(*ast.ExpressionStatement); ok {
			consResult = g.generateExpression(last.Expression)
		} else if ret, ok := e.Consequence.Statements[len(e.Consequence.Statements)-1].(*ast.ReturnStatement); ok && ret.ReturnValue != nil {
			consResult = g.generateExpression(ret.ReturnValue)
		}
	}
	if consResult == "" {
		consResult = g.zeroValue(ifType)
	}

	if e.Alternative != nil && len(e.Alternative.Statements) > 0 {
		if last, ok := e.Alternative.Statements[len(e.Alternative.Statements)-1].(*ast.ExpressionStatement); ok {
			altResult = g.generateExpression(last.Expression)
		} else if ret, ok := e.Alternative.Statements[len(e.Alternative.Statements)-1].(*ast.ReturnStatement); ok && ret.ReturnValue != nil {
			altResult = g.generateExpression(ret.ReturnValue)
		}
	}
	if altResult == "" {
		altResult = g.zeroValue(ifType)
	}

	g.addPreamble(fmt.Sprintf("%s %s;", ifType, tempName))
	g.addPreamble(fmt.Sprintf("if (%s) { %s = %s; } else { %s = %s; }",
		cond, tempName, consResult, tempName, altResult))
	return tempName
}

func (g *CGenerator) inferIfExprType(e *ast.IfExpression) string {
	if len(e.Consequence.Statements) > 0 {
		if last, ok := e.Consequence.Statements[len(e.Consequence.Statements)-1].(*ast.ExpressionStatement); ok {
			return g.resolveType(last.Expression)
		}
	}
	return "carv_int"
}

func (g *CGenerator) generateMemberExpression(e *ast.MemberExpression) string {
	obj := g.generateExpression(e.Object)
	member := e.Member.Value
	objCType := g.resolveType(e.Object)
	if g.isInterfaceRefType(objCType) {
		return fmt.Sprintf("%s.%s", obj, member)
	}
	return fmt.Sprintf("%s->%s", obj, member)
}

func (g *CGenerator) generateNewExpression(e *ast.NewExpression) string {
	if named, ok := e.Type.(*ast.NamedType); ok {
		className := named.Name.Value
		return fmt.Sprintf("%s_new()", className)
	}
	return "NULL"
}

func (g *CGenerator) generateIndexExpression(e *ast.IndexExpression) string {
	left := g.generateExpression(e.Left)
	index := g.generateExpression(e.Index)
	return fmt.Sprintf("%s.data[%s]", left, index)
}

func (g *CGenerator) generateArrayLiteral(e *ast.ArrayLiteral) string {
	if len(e.Elements) == 0 {
		return "carv_new_int_array(0)"
	}

	elemType := g.resolveType(e.Elements[0])
	arrayType := g.getArrayType(elemType)
	tempName := fmt.Sprintf("__arr_%d", g.tempCounter)
	g.tempCounter++

	g.arrayLengths[tempName] = len(e.Elements)

	var elements []string
	for _, elem := range e.Elements {
		elements = append(elements, g.generateExpression(elem))
	}

	return fmt.Sprintf("(%s){(%s[]){%s}, %d, %d}",
		arrayType,
		elemType,
		strings.Join(elements, ", "),
		len(e.Elements),
		len(e.Elements))
}

func (g *CGenerator) getArrayType(elemType string) string {
	switch elemType {
	case "carv_int":
		return "carv_int_array"
	case "carv_float":
		return "carv_float_array"
	case "carv_string":
		return "carv_string_array"
	case "carv_bool":
		return "carv_bool_array"
	default:
		return "carv_int_array"
	}
}

func (g *CGenerator) generateInterpolatedString(e *ast.InterpolatedString) string {
	if len(e.Parts) == 0 {
		return "carv_string_lit(\"\")"
	}

	if len(e.Parts) == 1 {
		return g.convertToString(e.Parts[0])
	}

	var result string
	for i, part := range e.Parts {
		partStr := g.convertToString(part)
		if i == 0 {
			result = partStr
		} else {
			result = fmt.Sprintf("carv_concat(%s, %s)", result, partStr)
		}
	}
	return result
}

func (g *CGenerator) convertToString(expr ast.Expression) string {
	if str, ok := expr.(*ast.StringLiteral); ok {
		return fmt.Sprintf("carv_string_lit(\"%s\")", g.escapeString(str.Value))
	}

	exprStr := g.generateExpression(expr)
	exprType := g.resolveType(expr)

	switch exprType {
	case "carv_string":
		return exprStr
	case "carv_int":
		return fmt.Sprintf("carv_int_to_string(%s)", exprStr)
	case "carv_float":
		return fmt.Sprintf("carv_float_to_string(%s)", exprStr)
	case "carv_bool":
		return fmt.Sprintf("carv_bool_to_string(%s)", exprStr)
	default:
		return fmt.Sprintf("carv_int_to_string((carv_int)%s)", exprStr)
	}
}

func (g *CGenerator) inferArrayElemType(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.ArrayLiteral:
		if len(e.Elements) > 0 {
			return g.resolveType(e.Elements[0])
		}
	case *ast.Identifier:
		return "carv_int"
	}
	return "carv_int"
}

func (g *CGenerator) typeToC(typeExpr ast.TypeExpr) string {
	if typeExpr == nil {
		return "void"
	}

	switch t := typeExpr.(type) {
	case *ast.BasicType:
		switch t.Name {
		case "int":
			return "carv_int"
		case "float":
			return "carv_float"
		case "bool":
			return "carv_bool"
		case "string":
			return "carv_string"
		case "void":
			return "void"
		}
	case *ast.RefType:
		if named, ok := t.Inner.(*ast.NamedType); ok {
			if _, isIface := g.interfaces[named.Name.Value]; isIface {
				if t.Mutable {
					return named.Name.Value + "_mut_ref"
				}
				return named.Name.Value + "_ref"
			}
		}
		inner := g.typeToC(t.Inner)
		if t.Mutable {
			return inner + "*"
		}
		return "const " + inner + "*"
	case *ast.NamedType:
		if _, isIface := g.interfaces[t.Name.Value]; isIface {
			return t.Name.Value + "_ref"
		}
		return t.Name.Value + "*"
	}
	return "void"
}

func (g *CGenerator) paramsToC(params []*ast.Parameter) string {
	if len(params) == 0 {
		return "void"
	}

	var parts []string
	for _, p := range params {
		pType := g.typeToC(p.Type)
		parts = append(parts, fmt.Sprintf("%s %s", pType, p.Name.Value))
	}
	return strings.Join(parts, ", ")
}

func (g *CGenerator) inferType(expr ast.Expression) string {
	return g.resolveType(expr)
}

func (g *CGenerator) inferExprType(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return "carv_int"
	case *ast.FloatLiteral:
		return "carv_float"
	case *ast.StringLiteral:
		return "carv_string"
	case *ast.InterpolatedString:
		return "carv_string"
	case *ast.BoolLiteral:
		return "carv_bool"
	case *ast.NewExpression:
		if named, ok := e.Type.(*ast.NamedType); ok {
			return named.Name.Value + "*"
		}
		return "void*"
	case *ast.InfixExpression:
		if e.Operator == "<" || e.Operator == ">" || e.Operator == "<=" ||
			e.Operator == ">=" || e.Operator == "==" || e.Operator == "!=" ||
			e.Operator == "&&" || e.Operator == "||" {
			return "carv_bool"
		}
		leftType := g.inferExprType(e.Left)
		rightType := g.inferExprType(e.Right)
		if leftType == "carv_float" || rightType == "carv_float" {
			return "carv_float"
		}
		return "carv_int"
	case *ast.PrefixExpression:
		if e.Operator == "!" {
			return "carv_bool"
		}
		return g.inferExprType(e.Right)
	case *ast.Identifier:
		if t := g.getVarType(e.Value); t != "" {
			return t
		}
		return "carv_int"
	case *ast.CallExpression:
		return g.inferCallType(e)
	case *ast.ArrayLiteral:
		if len(e.Elements) > 0 {
			return g.getArrayType(g.inferExprType(e.Elements[0]))
		}
		return "carv_int_array"
	case *ast.OkExpression, *ast.ErrExpression:
		return "carv_result"
	case *ast.TryExpression:
		return g.inferResultOkType(e.Value)
	case *ast.BorrowExpression:
		inner := g.inferExprType(e.Value)
		if e.Mutable {
			return inner + "*"
		}
		return "const " + inner + "*"
	case *ast.DerefExpression:
		inner := g.inferExprType(e.Value)
		if strings.HasSuffix(inner, "*") {
			return strings.TrimSuffix(inner, "*")
		}
		return inner
	case *ast.CastExpression:
		return g.typeToC(e.Type)
	case *ast.AwaitExpression:
		if call, ok := e.Value.(*ast.CallExpression); ok {
			if ident, ok := call.Function.(*ast.Identifier); ok {
				if info, exists := g.asyncFns[ident.Value]; exists {
					return info.ReturnType
				}
			}
		}
		return "void*"
	}
	return "carv_int"
}

func (g *CGenerator) inferCallType(e *ast.CallExpression) string {
	if ident, ok := e.Function.(*ast.Identifier); ok {
		if retType, exists := g.fnReturnTypes[ident.Value]; exists {
			return retType
		}
		switch ident.Value {
		case "read_file", "join", "trim", "substr":
			return "carv_string"
		case "split":
			return "carv_string_array"
		case "file_exists", "write_file":
			return "carv_bool"
		case "tcp_read":
			return "carv_string"
		case "tcp_close":
			return "carv_bool"
		case "tcp_listen", "tcp_accept", "tcp_write":
			return "carv_int"
		case "len":
			return "carv_int"
		}
	}
	return "carv_int"
}

func (g *CGenerator) generateOkExpression(e *ast.OkExpression) string {
	val := g.generateExpression(e.Value)
	valType := g.resolveType(e.Value)

	switch valType {
	case "carv_int":
		return fmt.Sprintf("carv_ok_int(%s)", val)
	case "carv_float":
		return fmt.Sprintf("carv_ok_float(%s)", val)
	case "carv_bool":
		return fmt.Sprintf("carv_ok_bool(%s)", val)
	case "carv_string":
		return fmt.Sprintf("carv_ok_str(%s)", val)
	default:
		return fmt.Sprintf("carv_ok_int(%s)", val)
	}
}

func (g *CGenerator) generateErrExpression(e *ast.ErrExpression) string {
	val := g.generateExpression(e.Value)
	valType := g.resolveType(e.Value)

	switch valType {
	case "carv_string":
		return fmt.Sprintf("carv_err_str(%s)", val)
	case "carv_int":
		return fmt.Sprintf("carv_err_code(%s)", val)
	default:
		return fmt.Sprintf("carv_err_str(%s)", val)
	}
}

func (g *CGenerator) generateTryExpression(e *ast.TryExpression) string {
	val := g.generateExpression(e.Value)
	tempName := fmt.Sprintf("__try_%d", g.tempCounter)
	g.tempCounter++

	okType := g.inferResultOkType(e.Value)
	okField := g.okFieldForType(okType)

	g.addPreamble(fmt.Sprintf("carv_result %s = %s;", tempName, val))
	if g.inFunction {
		g.addPreamble(fmt.Sprintf("if (!%s.is_ok) { __carv_retval = %s; goto __carv_exit; }", tempName, tempName))
	} else {
		g.addPreamble(fmt.Sprintf("if (!%s.is_ok) return %s;", tempName, tempName))
	}
	return fmt.Sprintf("%s.ok.%s", tempName, okField)
}

func (g *CGenerator) inferResultOkType(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.OkExpression:
		return g.resolveType(e.Value)
	case *ast.CallExpression:
		if ident, ok := e.Function.(*ast.Identifier); ok {
			if v := g.lookupVar(ident.Value + "_result_ok"); v != nil {
				return v.CType
			}
		}
	case *ast.Identifier:
		if v := g.lookupVar(e.Value + "_result_ok"); v != nil {
			return v.CType
		}
	}
	return "carv_int"
}

func (g *CGenerator) okFieldForType(cType string) string {
	switch cType {
	case "carv_int":
		return "ok_int"
	case "carv_float":
		return "ok_float"
	case "carv_bool":
		return "ok_bool"
	case "carv_string":
		return "ok_str"
	default:
		return "ok_int"
	}
}

func (g *CGenerator) generateMatchExpression(e *ast.MatchExpression) string {
	val := g.generateExpression(e.Value)
	tempName := fmt.Sprintf("__match_%d", g.tempCounter)
	g.tempCounter++
	resultName := fmt.Sprintf("__match_res_%d", g.tempCounter)
	g.tempCounter++

	okType := g.inferResultOkType(e.Value)
	okField := g.okFieldForType(okType)
	errType := g.inferResultErrType(e.Value)
	errField := g.errFieldForType(errType)

	resultType := "carv_int"
	if len(e.Arms) > 0 {
		resultType = g.resolveType(e.Arms[0].Body)
	}

	g.addPreamble(fmt.Sprintf("carv_result %s = %s;", tempName, val))
	g.addPreamble(fmt.Sprintf("%s %s;", resultType, resultName))

	for i, arm := range e.Arms {
		prefix := ""
		if i > 0 {
			prefix = "else "
		}

		g.enterScope()
		if ok, isOk := arm.Pattern.(*ast.OkExpression); isOk {
			if ident, isIdent := ok.Value.(*ast.Identifier); isIdent {
				g.declareVar(ident.Value, okType, false, false)
				g.addPreamble(fmt.Sprintf("%sif (%s.is_ok) { %s %s = %s.ok.%s;",
					prefix, tempName, okType, ident.Value, tempName, okField))
			} else {
				g.addPreamble(fmt.Sprintf("%sif (%s.is_ok) {", prefix, tempName))
			}
		} else if errExpr, isErr := arm.Pattern.(*ast.ErrExpression); isErr {
			if ident, isIdent := errExpr.Value.(*ast.Identifier); isIdent {
				g.declareVar(ident.Value, errType, false, false)
				g.addPreamble(fmt.Sprintf("%sif (!%s.is_ok) { %s %s = %s.err.%s;",
					prefix, tempName, errType, ident.Value, tempName, errField))
			} else {
				g.addPreamble(fmt.Sprintf("%sif (!%s.is_ok) {", prefix, tempName))
			}
		} else {
			g.addPreamble(fmt.Sprintf("%s{", prefix))
		}

		bodyExpr := g.generateExpression(arm.Body)
		g.addPreamble(fmt.Sprintf("%s = %s;", resultName, bodyExpr))
		g.addPreamble("}")
		g.exitScope()
	}

	return resultName
}

func (g *CGenerator) inferResultErrType(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.ErrExpression:
		return g.resolveType(e.Value)
	case *ast.CallExpression:
		if ident, ok := e.Function.(*ast.Identifier); ok {
			if v := g.lookupVar(ident.Value + "_result_err"); v != nil {
				return v.CType
			}
		}
	case *ast.Identifier:
		if v := g.lookupVar(e.Value + "_result_err"); v != nil {
			return v.CType
		}
	}
	return "carv_string"
}

func (g *CGenerator) errFieldForType(cType string) string {
	switch cType {
	case "carv_int":
		return "err_code"
	case "carv_string":
		return "err_str"
	default:
		return "err_str"
	}
}

func (g *CGenerator) collectInterfacesAndImpls(program *ast.Program) {
	for _, stmt := range program.Statements {
		if iface, ok := stmt.(*ast.InterfaceStatement); ok {
			g.interfaces[iface.Name.Value] = &interfaceInfo{
				name:    iface.Name.Value,
				methods: iface.Methods,
			}
		}
	}
	for _, stmt := range program.Statements {
		if impl, ok := stmt.(*ast.ImplStatement); ok {
			g.implList = append(g.implList, &implInfo{
				ifaceName: impl.Interface.Value,
				typeName:  impl.Type.Value,
				methods:   impl.Methods,
			})
		}
	}
}

func (g *CGenerator) generateInterfaceTypedefs() {
	for _, info := range g.interfaces {
		g.writeln("typedef struct {")
		g.indent++
		for _, sig := range info.methods {
			retType := g.methodSigReturnType(sig)
			params := g.vtableMethodParams(sig)
			g.writeln(fmt.Sprintf("%s (*%s)(%s);", retType, sig.Name.Value, params))
		}
		g.indent--
		g.writeln(fmt.Sprintf("} %s_vtable;", info.name))
		g.writeln("")
		g.writeln("typedef struct {")
		g.indent++
		g.writeln("const void* data;")
		g.writeln(fmt.Sprintf("const %s_vtable* vt;", info.name))
		g.indent--
		g.writeln(fmt.Sprintf("} %s_ref;", info.name))
		g.writeln("")
		g.writeln("typedef struct {")
		g.indent++
		g.writeln("void* data;")
		g.writeln(fmt.Sprintf("const %s_vtable* vt;", info.name))
		g.indent--
		g.writeln(fmt.Sprintf("} %s_mut_ref;", info.name))
		g.writeln("")
	}
}

func (g *CGenerator) methodSigReturnType(sig *ast.MethodSignature) string {
	if sig.ReturnType == nil {
		return "void"
	}
	return g.typeToC(sig.ReturnType)
}

func (g *CGenerator) vtableMethodParams(sig *ast.MethodSignature) string {
	parts := []string{}
	switch sig.Receiver {
	case ast.RecvRef:
		parts = append(parts, "const void* self")
	case ast.RecvMutRef, ast.RecvValue:
		parts = append(parts, "void* self")
	}
	for _, p := range sig.Parameters {
		pType := g.typeToC(p.Type)
		parts = append(parts, fmt.Sprintf("%s %s", pType, p.Name.Value))
	}
	return strings.Join(parts, ", ")
}

func (g *CGenerator) generateImplMethodDecls() {
	for _, impl := range g.implList {
		for _, method := range impl.methods {
			retType := g.typeToC(method.ReturnType)
			params := g.methodParamsToC(impl.typeName, method.Receiver, method.Parameters)
			g.writeln(fmt.Sprintf("%s %s_%s(%s);", retType, impl.typeName, method.Name.Value, params))
		}
	}
}

func (g *CGenerator) generateImplMethods(program *ast.Program) {
	for _, impl := range g.implList {
		for _, method := range impl.methods {
			retType := g.typeToC(method.ReturnType)
			params := g.methodParamsToC(impl.typeName, method.Receiver, method.Parameters)
			g.writeln(fmt.Sprintf("%s %s_%s(%s) {", retType, impl.typeName, method.Name.Value, params))
			g.indent++
			g.enterScope()

			g.inFunction = true
			g.funcRetType = retType

			if retType != "void" {
				g.writeln(fmt.Sprintf("%s __carv_retval = %s;", retType, g.zeroValue(retType)))
			}

			for _, p := range method.Parameters {
				pType := g.typeToC(p.Type)
				g.declareVar(p.Name.Value, pType, false, false)
			}

			for _, stmt := range method.Body.Statements {
				g.generateStatement(stmt)
			}

			g.writeln("__carv_exit:;")
			g.emitScopeDrops()
			if retType != "void" {
				g.writeln("return __carv_retval;")
			}

			g.exitScope()
			g.inFunction = false
			g.indent--
			g.writeln("}")
			g.writeln("")
		}
	}
}

func (g *CGenerator) generateImplWrappers() {
	for _, impl := range g.implList {
		iface, ok := g.interfaces[impl.ifaceName]
		if !ok {
			continue
		}

		for _, sig := range iface.methods {
			retType := g.methodSigReturnType(sig)
			wrapperParams := g.vtableMethodParams(sig)
			wrapperName := fmt.Sprintf("%s__%s__%s", impl.ifaceName, impl.typeName, sig.Name.Value)

			g.writeln(fmt.Sprintf("static %s %s(%s) {", retType, wrapperName, wrapperParams))
			g.indent++

			switch sig.Receiver {
			case ast.RecvRef:
				g.writeln(fmt.Sprintf("const %s* p = (const %s*)self;", impl.typeName, impl.typeName))
			default:
				g.writeln(fmt.Sprintf("%s* p = (%s*)self;", impl.typeName, impl.typeName))
			}

			var argParts []string
			argParts = append(argParts, "p")
			for _, p := range sig.Parameters {
				argParts = append(argParts, p.Name.Value)
			}

			call := fmt.Sprintf("%s_%s(%s)", impl.typeName, sig.Name.Value, strings.Join(argParts, ", "))
			if retType == "void" {
				g.writeln(call + ";")
			} else {
				g.writeln(fmt.Sprintf("return %s;", call))
			}

			g.indent--
			g.writeln("}")
			g.writeln("")
		}

		g.writeln(fmt.Sprintf("static const %s_vtable %s__%s__VT = {", impl.ifaceName, impl.ifaceName, impl.typeName))
		g.indent++
		for _, sig := range iface.methods {
			wrapperName := fmt.Sprintf("%s__%s__%s", impl.ifaceName, impl.typeName, sig.Name.Value)
			g.writeln(fmt.Sprintf(".%s = %s,", sig.Name.Value, wrapperName))
		}
		g.indent--
		g.writeln("};")
		g.writeln("")
	}
}

func (g *CGenerator) generateCastExpression(e *ast.CastExpression) string {
	val := g.generateExpression(e.Value)

	if refType, ok := e.Type.(*ast.RefType); ok {
		if named, ok := refType.Inner.(*ast.NamedType); ok {
			ifaceName := named.Name.Value
			if _, isIface := g.interfaces[ifaceName]; isIface {
				className := g.inferCastSourceClass(e.Value)
				if className != "" {
					if refType.Mutable {
						return fmt.Sprintf("(%s_mut_ref){ .data = %s, .vt = &%s__%s__VT }",
							ifaceName, val, ifaceName, className)
					}
					return fmt.Sprintf("(%s_ref){ .data = %s, .vt = &%s__%s__VT }",
						ifaceName, val, ifaceName, className)
				}
			}
		}
	}

	targetType := g.typeToC(e.Type)
	return fmt.Sprintf("(%s)%s", targetType, val)
}

func (g *CGenerator) inferCastSourceClass(expr ast.Expression) string {
	if borrow, ok := expr.(*ast.BorrowExpression); ok {
		return g.inferClassName(borrow.Value)
	}
	return g.inferClassName(expr)
}

func (g *CGenerator) escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}

func (g *CGenerator) writeln(s string) {
	g.write(s)
	g.output.WriteString("\n")
}

func (g *CGenerator) write(s string) {
	for i := 0; i < g.indent; i++ {
		g.output.WriteString("    ")
	}
	g.output.WriteString(s)
}

func (g *CGenerator) writeRaw(s string) {
	g.output.WriteString(s)
}
