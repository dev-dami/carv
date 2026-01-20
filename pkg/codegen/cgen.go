package codegen

import (
	"fmt"
	"strings"

	"github.com/dev-dami/carv/pkg/ast"
)

type CGenerator struct {
	output       strings.Builder
	indent       int
	tempCounter  int
	functions    []string
	arrayLengths map[string]int
	varTypes     map[string]string
}

func NewCGenerator() *CGenerator {
	return &CGenerator{
		arrayLengths: make(map[string]int),
		varTypes:     make(map[string]string),
	}
}

func (g *CGenerator) getVarType(name string) string {
	if t, ok := g.varTypes[name]; ok {
		return t
	}
	return ""
}

func (g *CGenerator) Generate(program *ast.Program) string {
	g.emitRuntime()

	for _, stmt := range program.Statements {
		if cls, ok := stmt.(*ast.ClassStatement); ok {
			g.generateClassDecl(cls)
		}
	}

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

	g.writeln("")
	g.writeln("int main(void) {")
	g.indent++

	for _, stmt := range program.Statements {
		switch stmt.(type) {
		case *ast.FunctionStatement, *ast.ClassStatement:
			continue
		default:
			g.generateStatement(stmt)
		}
	}

	g.writeln("return 0;")
	g.indent--
	g.writeln("}")

	return g.output.String()
}

func (g *CGenerator) emitRuntime() {
	g.writeln("#include <stdio.h>")
	g.writeln("#include <stdlib.h>")
	g.writeln("#include <string.h>")
	g.writeln("#include <stdbool.h>")
	g.writeln("")
	g.writeln("typedef long long carv_int;")
	g.writeln("typedef double carv_float;")
	g.writeln("typedef bool carv_bool;")
	g.writeln("typedef char* carv_string;")
	g.writeln("")
	g.writeln("typedef struct { carv_int* data; carv_int len; carv_int cap; } carv_int_array;")
	g.writeln("typedef struct { carv_float* data; carv_int len; carv_int cap; } carv_float_array;")
	g.writeln("typedef struct { carv_string* data; carv_int len; carv_int cap; } carv_string_array;")
	g.writeln("typedef struct { carv_bool* data; carv_int len; carv_int cap; } carv_bool_array;")
	g.writeln("")
	g.writeln("carv_int_array carv_new_int_array(carv_int len) {")
	g.writeln("    carv_int_array arr;")
	g.writeln("    arr.data = (carv_int*)malloc(len * sizeof(carv_int));")
	g.writeln("    arr.len = len;")
	g.writeln("    arr.cap = len;")
	g.writeln("    return arr;")
	g.writeln("}")
	g.writeln("")
	g.writeln("carv_float_array carv_new_float_array(carv_int len) {")
	g.writeln("    carv_float_array arr;")
	g.writeln("    arr.data = (carv_float*)malloc(len * sizeof(carv_float));")
	g.writeln("    arr.len = len;")
	g.writeln("    arr.cap = len;")
	g.writeln("    return arr;")
	g.writeln("}")
	g.writeln("")
	g.writeln("carv_string_array carv_new_string_array(carv_int len) {")
	g.writeln("    carv_string_array arr;")
	g.writeln("    arr.data = (carv_string*)malloc(len * sizeof(carv_string));")
	g.writeln("    arr.len = len;")
	g.writeln("    arr.cap = len;")
	g.writeln("    return arr;")
	g.writeln("}")
	g.writeln("")
	g.writeln("void carv_print_int(carv_int x) { printf(\"%lld\\n\", x); }")
	g.writeln("void carv_print_float(carv_float x) { printf(\"%g\\n\", x); }")
	g.writeln("void carv_print_bool(carv_bool x) { printf(\"%s\\n\", x ? \"true\" : \"false\"); }")
	g.writeln("void carv_print_string(carv_string x) { printf(\"%s\\n\", x); }")
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

	g.writeln("carv_string carv_read_file(carv_string path) {")
	g.writeln("    FILE* f = fopen(path, \"rb\");")
	g.writeln("    if (!f) return NULL;")
	g.writeln("    fseek(f, 0, SEEK_END);")
	g.writeln("    long len = ftell(f);")
	g.writeln("    fseek(f, 0, SEEK_SET);")
	g.writeln("    char* buf = (char*)malloc(len + 1);")
	g.writeln("    if (!buf) { fclose(f); return NULL; }")
	g.writeln("    size_t read = fread(buf, 1, len, f);")
	g.writeln("    buf[read] = '\\0';")
	g.writeln("    fclose(f);")
	g.writeln("    return buf;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_bool carv_write_file(carv_string path, carv_string content) {")
	g.writeln("    FILE* f = fopen(path, \"wb\");")
	g.writeln("    if (!f) return false;")
	g.writeln("    size_t len = strlen(content);")
	g.writeln("    size_t written = fwrite(content, 1, len, f);")
	g.writeln("    fclose(f);")
	g.writeln("    return written == len;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_bool carv_file_exists(carv_string path) {")
	g.writeln("    FILE* f = fopen(path, \"r\");")
	g.writeln("    if (f) { fclose(f); return true; }")
	g.writeln("    return false;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_string_array carv_split(carv_string str, carv_string sep) {")
	g.writeln("    carv_string_array arr = {NULL, 0, 0};")
	g.writeln("    if (!str || !sep) return arr;")
	g.writeln("    size_t sep_len = strlen(sep);")
	g.writeln("    if (sep_len == 0) {")
	g.writeln("        arr = carv_new_string_array(1);")
	g.writeln("        arr.data[0] = strdup(str);")
	g.writeln("        return arr;")
	g.writeln("    }")
	g.writeln("    // Count occurrences")
	g.writeln("    carv_int count = 1;")
	g.writeln("    char* p = str;")
	g.writeln("    while ((p = strstr(p, sep)) != NULL) { count++; p += sep_len; }")
	g.writeln("    arr = carv_new_string_array(count);")
	g.writeln("    // Split")
	g.writeln("    char* start = str;")
	g.writeln("    carv_int idx = 0;")
	g.writeln("    while ((p = strstr(start, sep)) != NULL) {")
	g.writeln("        size_t part_len = p - start;")
	g.writeln("        arr.data[idx] = (char*)malloc(part_len + 1);")
	g.writeln("        memcpy(arr.data[idx], start, part_len);")
	g.writeln("        arr.data[idx][part_len] = '\\0';")
	g.writeln("        idx++;")
	g.writeln("        start = p + sep_len;")
	g.writeln("    }")
	g.writeln("    arr.data[idx] = strdup(start);")
	g.writeln("    return arr;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_string carv_join(carv_string_array arr, carv_string sep) {")
	g.writeln("    if (arr.len == 0) return strdup(\"\");")
	g.writeln("    size_t sep_len = sep ? strlen(sep) : 0;")
	g.writeln("    size_t total = 0;")
	g.writeln("    for (carv_int i = 0; i < arr.len; i++) {")
	g.writeln("        if (arr.data[i]) total += strlen(arr.data[i]);")
	g.writeln("    }")
	g.writeln("    total += sep_len * (arr.len - 1) + 1;")
	g.writeln("    char* result = (char*)malloc(total);")
	g.writeln("    result[0] = '\\0';")
	g.writeln("    for (carv_int i = 0; i < arr.len; i++) {")
	g.writeln("        if (i > 0 && sep) strcat(result, sep);")
	g.writeln("        if (arr.data[i]) strcat(result, arr.data[i]);")
	g.writeln("    }")
	g.writeln("    return result;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_string carv_trim(carv_string str) {")
	g.writeln("    if (!str) return strdup(\"\");")
	g.writeln("    while (*str && (*str == ' ' || *str == '\\t' || *str == '\\n' || *str == '\\r')) str++;")
	g.writeln("    if (*str == '\\0') return strdup(\"\");")
	g.writeln("    char* end = str + strlen(str) - 1;")
	g.writeln("    while (end > str && (*end == ' ' || *end == '\\t' || *end == '\\n' || *end == '\\r')) end--;")
	g.writeln("    size_t len = end - str + 1;")
	g.writeln("    char* result = (char*)malloc(len + 1);")
	g.writeln("    memcpy(result, str, len);")
	g.writeln("    result[len] = '\\0';")
	g.writeln("    return result;")
	g.writeln("}")
	g.writeln("")

	g.writeln("carv_string carv_substr(carv_string str, carv_int start, carv_int end) {")
	g.writeln("    if (!str) return strdup(\"\");")
	g.writeln("    size_t str_len = strlen(str);")
	g.writeln("    if (start < 0) start = 0;")
	g.writeln("    if (end < 0) end = str_len;")
	g.writeln("    if ((size_t)start >= str_len) return strdup(\"\");")
	g.writeln("    if ((size_t)end > str_len) end = str_len;")
	g.writeln("    if (end <= start) return strdup(\"\");")
	g.writeln("    size_t len = end - start;")
	g.writeln("    char* result = (char*)malloc(len + 1);")
	g.writeln("    memcpy(result, str + start, len);")
	g.writeln("    result[len] = '\\0';")
	g.writeln("    return result;")
	g.writeln("}")
	g.writeln("")

	g.writeln("typedef struct { carv_bool is_ok; union { carv_int ok_int; carv_string ok_str; void* ok_ptr; } ok; union { carv_string err_str; carv_int err_code; } err; } carv_result;")
	g.writeln("")
	g.writeln("carv_result carv_ok_int(carv_int val) { carv_result r; r.is_ok = true; r.ok.ok_int = val; return r; }")
	g.writeln("carv_result carv_ok_str(carv_string val) { carv_result r; r.is_ok = true; r.ok.ok_str = val; return r; }")
	g.writeln("carv_result carv_err_str(carv_string val) { carv_result r; r.is_ok = false; r.err.err_str = val; return r; }")
	g.writeln("carv_result carv_err_code(carv_int val) { carv_result r; r.is_ok = false; r.err.err_code = val; return r; }")
	g.writeln("")
}

func (g *CGenerator) generateFunctionDecl(fn *ast.FunctionStatement) {
	retType := g.typeToC(fn.ReturnType)
	params := g.paramsToC(fn.Parameters)
	fnName := g.safeName(fn.Name.Value)
	g.writeln(fmt.Sprintf("%s %s(%s);", retType, fnName, params))
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
	}
	if reserved[name] {
		return "carv_" + name
	}
	return name
}

func (g *CGenerator) generateFunction(fn *ast.FunctionStatement) {
	retType := g.typeToC(fn.ReturnType)
	params := g.paramsToC(fn.Parameters)
	fnName := g.safeName(fn.Name.Value)
	g.writeln(fmt.Sprintf("%s %s(%s) {", retType, fnName, params))
	g.indent++

	for _, stmt := range fn.Body.Statements {
		g.generateStatement(stmt)
	}

	if retType == "void" {
		g.writeln("return;")
	}

	g.indent--
	g.writeln("}")
	g.writeln("")
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
	g.writeln(fmt.Sprintf("%s* self = (%s*)malloc(sizeof(%s));", className, className, className))
	for _, field := range cls.Fields {
		if field.Default != nil {
			defaultVal := g.generateExpression(field.Default)
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
		return "NULL"
	default:
		return "0"
	}
}

func (g *CGenerator) generateClassMethodDecls(cls *ast.ClassStatement) {
	className := cls.Name.Value
	for _, method := range cls.Methods {
		retType := g.typeToC(method.ReturnType)
		params := g.methodParamsToC(className, method.Parameters)
		g.writeln(fmt.Sprintf("%s %s_%s(%s);", retType, className, method.Name.Value, params))
	}
}

func (g *CGenerator) methodParamsToC(className string, params []*ast.Parameter) string {
	parts := []string{fmt.Sprintf("%s* self", className)}
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
		params := g.methodParamsToC(className, method.Parameters)
		g.writeln(fmt.Sprintf("%s %s_%s(%s) {", retType, className, method.Name.Value, params))
		g.indent++

		for _, stmt := range method.Body.Statements {
			g.generateStatement(stmt)
		}

		if retType == "void" {
			g.writeln("return;")
		}

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
	value := g.generateExpression(s.Value)

	g.varTypes[varName] = varType

	if arr, ok := s.Value.(*ast.ArrayLiteral); ok {
		arrayType := g.getArrayType(g.inferArrayElemType(s.Value))
		g.arrayLengths[varName] = len(arr.Elements)
		g.varTypes[varName] = arrayType
		g.writeln(fmt.Sprintf("%s %s = %s;", arrayType, varName, value))
	} else {
		g.writeln(fmt.Sprintf("%s %s = %s;", varType, varName, value))
	}
}

func (g *CGenerator) generateConstStatement(s *ast.ConstStatement) {
	varType := g.inferType(s.Value)
	varName := s.Name.Value
	value := g.generateExpression(s.Value)
	g.writeln(fmt.Sprintf("const %s %s = %s;", varType, varName, value))
}

func (g *CGenerator) generateReturnStatement(s *ast.ReturnStatement) {
	if s.ReturnValue == nil {
		g.writeln("return;")
	} else {
		value := g.generateExpression(s.ReturnValue)
		g.writeln(fmt.Sprintf("return %s;", value))
	}
}

func (g *CGenerator) generateExpressionStatement(s *ast.ExpressionStatement) {
	expr := g.generateExpression(s.Expression)
	if expr != "" {
		g.writeln(expr + ";")
	}
}

func (g *CGenerator) generateForStatement(s *ast.ForStatement) {
	g.write("for (")

	if s.Init != nil {
		if let, ok := s.Init.(*ast.LetStatement); ok {
			varType := g.inferType(let.Value)
			value := g.generateExpression(let.Value)
			g.writeRaw(fmt.Sprintf("%s %s = %s; ", varType, let.Name.Value, value))
		}
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

	for _, stmt := range s.Body.Statements {
		g.generateStatement(stmt)
	}

	g.indent--
	g.writeln("}")
}

func (g *CGenerator) generateForInStatement(s *ast.ForInStatement) {
	iterName := s.Value.Value
	iterableExpr := g.generateExpression(s.Iterable)

	idxVar := fmt.Sprintf("__idx_%d", g.tempCounter)
	g.tempCounter++

	g.writeln(fmt.Sprintf("for (carv_int %s = 0; %s < %s.len; %s++) {", idxVar, idxVar, iterableExpr, idxVar))
	g.indent++

	elemType := g.inferArrayElemType(s.Iterable)
	g.writeln(fmt.Sprintf("%s %s = %s.data[%s];", elemType, iterName, iterableExpr, idxVar))

	for _, stmt := range s.Body.Statements {
		g.generateStatement(stmt)
	}

	g.indent--
	g.writeln("}")
}

func (g *CGenerator) generateWhileStatement(s *ast.WhileStatement) {
	cond := g.generateExpression(s.Condition)
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

	for _, stmt := range s.Statements {
		g.generateStatement(stmt)
	}

	g.indent--
	g.writeln("}")
}

func (g *CGenerator) generateIfStatement(e *ast.IfExpression) {
	cond := g.generateExpression(e.Condition)
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
		return fmt.Sprintf("\"%s\"", g.escapeString(e.Value))
	case *ast.BoolLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case *ast.NilLiteral:
		return "NULL"
	case *ast.Identifier:
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
	return fmt.Sprintf("(%s %s %s)", left, e.Operator, right)
}

func (g *CGenerator) generatePipeExpression(e *ast.PipeExpression) string {
	left := g.generateExpression(e.Left)
	leftType := g.inferExprType(e.Left)

	switch right := e.Right.(type) {
	case *ast.Identifier:
		fnName := g.safeName(right.Value)
		if fnName == "print" || fnName == "println" {
			return g.generatePrintExpr(left, leftType)
		}
		return fmt.Sprintf("%s(%s)", fnName, left)
	case *ast.CallExpression:
		if ident, ok := right.Function.(*ast.Identifier); ok {
			fnName := g.safeName(ident.Value)
			if fnName == "print" || fnName == "println" {
				return g.generatePrintExpr(left, leftType)
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
		return fmt.Sprintf("(printf(\"%%s\\n\", %s))", val)
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
		return g.generateMethodCall(member, e.Arguments)
	}

	fn := g.generateExpression(e.Function)

	if fn == "print" || fn == "println" {
		return g.generatePrintCall(e)
	}

	if fn == "len" && len(e.Arguments) == 1 {
		arg := g.generateExpression(e.Arguments[0])
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

	var args []string
	for _, arg := range e.Arguments {
		args = append(args, g.generateExpression(arg))
	}
	return fmt.Sprintf("%s(%s)", fn, strings.Join(args, ", "))
}

func (g *CGenerator) generateMethodCall(member *ast.MemberExpression, args []ast.Expression) string {
	obj := g.generateExpression(member.Object)
	methodName := member.Member.Value

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
		argType := g.inferExprType(arg)

		if _, ok := arg.(*ast.ArrayLiteral); ok {
			parts = append(parts, fmt.Sprintf("carv_print_int_array(%s)", argStr))
			continue
		}

		if ident, ok := arg.(*ast.Identifier); ok {
			if strings.HasPrefix(argType, "carv_int_array") || g.isArrayIdent(ident.Value) {
				parts = append(parts, fmt.Sprintf("carv_print_int_array(%s)", argStr))
				continue
			}
		}

		switch argType {
		case "carv_int":
			parts = append(parts, fmt.Sprintf("printf(\"%%lld\", %s)", argStr))
		case "carv_float":
			parts = append(parts, fmt.Sprintf("printf(\"%%g\", %s)", argStr))
		case "carv_bool":
			parts = append(parts, fmt.Sprintf("printf(\"%%s\", %s ? \"true\" : \"false\")", argStr))
		case "carv_string":
			parts = append(parts, fmt.Sprintf("printf(\"%%s\", %s)", argStr))
		default:
			parts = append(parts, fmt.Sprintf("printf(\"%%lld\", (carv_int)%s)", argStr))
		}
	}

	parts = append(parts, "printf(\"\\n\")")
	return "(" + strings.Join(parts, ", ") + ")"
}

func (g *CGenerator) isArrayIdent(name string) bool {
	_, ok := g.arrayLengths[name]
	return ok
}

func (g *CGenerator) generateIfExpression(e *ast.IfExpression) string {
	return ""
}

func (g *CGenerator) generateMemberExpression(e *ast.MemberExpression) string {
	obj := g.generateExpression(e.Object)
	member := e.Member.Value
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

	elemType := g.inferExprType(e.Elements[0])
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

func (g *CGenerator) inferArrayElemType(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.ArrayLiteral:
		if len(e.Elements) > 0 {
			return g.inferExprType(e.Elements[0])
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
	return g.inferExprType(expr)
}

func (g *CGenerator) inferExprType(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return "carv_int"
	case *ast.FloatLiteral:
		return "carv_float"
	case *ast.StringLiteral:
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
	case *ast.OkExpression, *ast.ErrExpression, *ast.TryExpression:
		return "carv_result"
	}
	return "carv_int"
}

func (g *CGenerator) inferCallType(e *ast.CallExpression) string {
	if ident, ok := e.Function.(*ast.Identifier); ok {
		switch ident.Value {
		case "read_file", "join", "trim", "substr":
			return "carv_string"
		case "split":
			return "carv_string_array"
		case "file_exists", "write_file":
			return "carv_bool"
		case "len":
			return "carv_int"
		}
	}
	return "carv_int"
}

func (g *CGenerator) inferResultExprType(expr ast.Expression) string {
	switch expr.(type) {
	case *ast.OkExpression, *ast.ErrExpression:
		return "carv_result"
	}
	return ""
}

func (g *CGenerator) generateOkExpression(e *ast.OkExpression) string {
	val := g.generateExpression(e.Value)
	valType := g.inferExprType(e.Value)

	switch valType {
	case "carv_int":
		return fmt.Sprintf("carv_ok_int(%s)", val)
	case "carv_string":
		return fmt.Sprintf("carv_ok_str(%s)", val)
	default:
		return fmt.Sprintf("carv_ok_int(%s)", val)
	}
}

func (g *CGenerator) generateErrExpression(e *ast.ErrExpression) string {
	val := g.generateExpression(e.Value)
	valType := g.inferExprType(e.Value)

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
	return fmt.Sprintf("({ carv_result %s = %s; if (!%s.is_ok) return %s; %s.ok.ok_int; })",
		tempName, val, tempName, tempName, tempName)
}

func (g *CGenerator) generateMatchExpression(e *ast.MatchExpression) string {
	val := g.generateExpression(e.Value)
	tempName := fmt.Sprintf("__match_%d", g.tempCounter)
	g.tempCounter++

	var result strings.Builder
	result.WriteString(fmt.Sprintf("({ carv_result %s = %s; ", tempName, val))

	for i, arm := range e.Arms {
		if i > 0 {
			result.WriteString(" else ")
		}

		if ok, isOk := arm.Pattern.(*ast.OkExpression); isOk {
			if ident, isIdent := ok.Value.(*ast.Identifier); isIdent {
				result.WriteString(fmt.Sprintf("if (%s.is_ok) { carv_int %s = %s.ok.ok_int; ",
					tempName, ident.Value, tempName))
			} else {
				result.WriteString(fmt.Sprintf("if (%s.is_ok) { ", tempName))
			}
		} else if errExpr, isErr := arm.Pattern.(*ast.ErrExpression); isErr {
			if ident, isIdent := errExpr.Value.(*ast.Identifier); isIdent {
				result.WriteString(fmt.Sprintf("if (!%s.is_ok) { carv_string %s = %s.err.err_str; ",
					tempName, ident.Value, tempName))
			} else {
				result.WriteString(fmt.Sprintf("if (!%s.is_ok) { ", tempName))
			}
		} else {
			result.WriteString("{ ")
		}

		bodyExpr := g.generateExpression(arm.Body)
		result.WriteString(fmt.Sprintf("%s; } ", bodyExpr))
	}

	result.WriteString("; })")
	return result.String()
}

func (g *CGenerator) escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
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
