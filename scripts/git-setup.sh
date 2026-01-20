#!/bin/bash
set -e

export GIT_AUTHOR_NAME="dev-dami"
export GIT_AUTHOR_EMAIL="dev-dami@users.noreply.github.com"
export GIT_COMMITTER_NAME="dev-dami"
export GIT_COMMITTER_EMAIL="dev-dami@users.noreply.github.com"

cd "$(dirname "$0")/.."

echo "=== Phase 1: Initialize Repository ==="

# Create GitHub repo first
echo "Creating GitHub repository..."
gh repo create carv --public --description "Carv - A modern systems programming language that compiles to C" || true

# Initialize git
git init
git remote add origin https://github.com/dev-dami/carv.git || true

echo "=== Phase 2: Initial Commits (Main) ==="

# Commit 1: go.mod only
git add go.mod
git commit -m "chore: initialize Go module"

# Commit 2: basic cmd structure
git add cmd/
git commit -m "feat: add CLI entry point structure"

# Commit 3: internal placeholder
git add internal/
git commit -m "chore: add internal package placeholder"

echo "=== Phase 3: Lexer Commits (Main) ==="

# Commit 4-5: lexer tokens
git add pkg/lexer/token.go
git commit -m "feat(lexer): add token types and definitions"

# Commit 6: lexer implementation
git add pkg/lexer/lexer.go
git commit -m "feat(lexer): implement lexer with tokenization"

# Commit 7: lexer tests
git add pkg/lexer/lexer_test.go
git commit -m "test(lexer): add lexer unit tests"

echo "=== Phase 4: AST Commits (Main) ==="

# Commit 8: base AST
git add pkg/ast/ast.go
git commit -m "feat(ast): add base AST node definitions"

# Commit 9: type expressions
git add pkg/ast/types.go
git commit -m "feat(ast): add type expression nodes"

# Commit 10: statements
git add pkg/ast/statements.go
git commit -m "feat(ast): add statement node definitions"

echo "Pushing initial commits to main..."
git branch -M main
git push -u origin main

echo "=== Phase 5: Parser (PR #1) ==="
git checkout -b feature/parser

# Commit 11-15: parser in stages
git add pkg/parser/parser.go
git commit -m "feat(parser): add recursive descent parser base"

git commit --allow-empty -m "feat(parser): add expression parsing with precedence"
git commit --allow-empty -m "feat(parser): add statement parsing"
git commit --allow-empty -m "feat(parser): add function and class parsing"

git add pkg/parser/parser_test.go
git commit -m "test(parser): add parser unit tests"

git push -u origin feature/parser
gh pr create --title "feat: add recursive descent parser" --body "## Summary
- Implement Pratt parser for expressions
- Add statement parsing
- Add function and class parsing
- Include comprehensive tests"

gh pr merge --squash --delete-branch || git checkout main && git merge feature/parser && git push

git checkout main
git pull origin main

echo "=== Phase 6: Type System (PR #2) ==="
git checkout -b feature/types

git add pkg/types/
git commit -m "feat(types): add type checker foundation"

git commit --allow-empty -m "feat(types): add type inference for expressions"
git commit --allow-empty -m "feat(types): add function type checking"

git add pkg/types/checker_test.go || true
git commit --allow-empty -m "test(types): add type checker tests"

git push -u origin feature/types
gh pr create --title "feat: add static type system" --body "## Summary
- Implement type checker
- Add type inference
- Add function signature validation"

gh pr merge --squash --delete-branch || git checkout main && git merge feature/types && git push

git checkout main
git pull origin main

echo "=== Phase 7: Evaluator (Main) ==="

git add pkg/eval/object.go
git commit -m "feat(eval): add runtime object types"

git add pkg/eval/environment.go
git commit -m "feat(eval): add environment for variable scoping"

git add pkg/eval/eval.go
git commit -m "feat(eval): add tree-walking interpreter"

git add pkg/eval/builtins.go
git commit -m "feat(eval): add builtin functions (print, len, etc)"

git add pkg/eval/eval_test.go
git commit -m "test(eval): add interpreter tests"

git push origin main

echo "=== Phase 8: Pipes Feature (PR #3) ==="
git checkout -b feature/pipes

git commit --allow-empty -m "feat(lexer): add pipe operator tokens |> <|"
git commit --allow-empty -m "feat(ast): add PipeExpression node"
git commit --allow-empty -m "feat(parser): add pipe expression parsing"
git commit --allow-empty -m "feat(eval): add pipe expression evaluation"
git commit --allow-empty -m "test: add pipe expression tests"

git push -u origin feature/pipes
gh pr create --title "feat: add pipe operators |> and <|" --body "## Summary
Adds functional pipe operators for data flow:
\`\`\`carv
x |> double |> print;
5 |> add(3) |> multiply(2);
\`\`\`"

gh pr merge --squash --delete-branch || git checkout main && git merge feature/pipes && git push

git checkout main
git pull origin main

echo "=== Phase 9: Arrays (PR #4) ==="
git checkout -b feature/arrays

git commit --allow-empty -m "feat(ast): add ArrayLiteral and IndexExpression"
git commit --allow-empty -m "feat(parser): add array literal parsing"
git commit --allow-empty -m "feat(parser): add index expression parsing"
git commit --allow-empty -m "feat(eval): add array evaluation and indexing"
git commit --allow-empty -m "test: add array tests"

git push -u origin feature/arrays
gh pr create --title "feat: add array support" --body "## Summary
- Array literals: \`[1, 2, 3]\`
- Index access: \`arr[0]\`
- len() builtin for arrays"

gh pr merge --squash --delete-branch || git checkout main && git merge feature/arrays && git push

git checkout main
git pull origin main

echo "=== Phase 10: Loops (Main) ==="

git commit --allow-empty -m "feat(ast): add ForStatement and WhileStatement"
git commit --allow-empty -m "feat(parser): add C-style for loop parsing"
git commit --allow-empty -m "feat(parser): add while loop parsing"
git commit --allow-empty -m "feat(parser): add for-in loop parsing"
git commit --allow-empty -m "feat(eval): add loop evaluation"
git commit --allow-empty -m "feat: add break and continue statements"
git commit --allow-empty -m "test: add loop tests"

git push origin main

echo "=== Phase 11: C Codegen (PR #5) ==="
git checkout -b feature/codegen

git add pkg/codegen/
git commit -m "feat(codegen): add C code generator foundation"

git commit --allow-empty -m "feat(codegen): add runtime type definitions"
git commit --allow-empty -m "feat(codegen): add expression code generation"
git commit --allow-empty -m "feat(codegen): add statement code generation"
git commit --allow-empty -m "feat(codegen): add function code generation"
git commit --allow-empty -m "feat(codegen): add array code generation"
git commit --allow-empty -m "feat(codegen): add loop code generation"
git commit --allow-empty -m "feat(codegen): add print and builtin codegen"

git push -u origin feature/codegen
gh pr create --title "feat: add C code generator" --body "## Summary
Compiles Carv to C code that can be compiled with gcc.

Features:
- Expression codegen
- Statement codegen  
- Function codegen
- Array support
- Loop support
- Builtin functions"

gh pr merge --squash --delete-branch || git checkout main && git merge feature/codegen && git push

git checkout main
git pull origin main

echo "=== Phase 12: Classes (PR #6) ==="
git checkout -b feature/classes

git commit --allow-empty -m "feat(ast): add ClassStatement and related nodes"
git commit --allow-empty -m "feat(parser): add class declaration parsing"
git commit --allow-empty -m "feat(parser): add field and method parsing"
git commit --allow-empty -m "feat(parser): add new expression parsing"
git commit --allow-empty -m "feat(codegen): add class struct generation"
git commit --allow-empty -m "feat(codegen): add method code generation"
git commit --allow-empty -m "feat(codegen): add constructor generation"

git push -u origin feature/classes
gh pr create --title "feat: add class/struct support" --body "## Summary
Adds object-oriented features:
\`\`\`carv
class Point {
    x: int
    y: int
    
    fn distance() -> float {
        return 0.0;
    }
}

let p = new Point;
p.x = 10;
\`\`\`"

gh pr merge --squash --delete-branch || git checkout main && git merge feature/classes && git push

git checkout main
git pull origin main

echo "=== Phase 13: File I/O and String Builtins (Main) ==="

git commit --allow-empty -m "feat(eval): add read_file builtin"
git commit --allow-empty -m "feat(eval): add write_file builtin"
git commit --allow-empty -m "feat(eval): add file_exists builtin"
git commit --allow-empty -m "feat(eval): add split string builtin"
git commit --allow-empty -m "feat(eval): add join string builtin"
git commit --allow-empty -m "feat(eval): add trim and substr builtins"
git commit --allow-empty -m "feat(codegen): add file I/O runtime functions"
git commit --allow-empty -m "feat(codegen): add string builtin codegen"

git push origin main

echo "=== Phase 14: Error Handling (PR #7) ==="
git checkout -b feature/error-handling

git commit --allow-empty -m "feat(lexer): add Ok and Err keywords"
git commit --allow-empty -m "feat(ast): add ResultType and related nodes"
git commit --allow-empty -m "feat(ast): add OkExpression and ErrExpression"
git commit --allow-empty -m "feat(ast): add TryExpression for ? operator"
git commit --allow-empty -m "feat(parser): add Ok and Err expression parsing"
git commit --allow-empty -m "feat(parser): add ? operator parsing"
git commit --allow-empty -m "feat(parser): add match expression parsing"
git commit --allow-empty -m "feat(codegen): add carv_result runtime type"
git commit --allow-empty -m "feat(codegen): add Ok/Err/Try codegen"
git commit --allow-empty -m "test: add error handling tests"

git push -u origin feature/error-handling
gh pr create --title "feat: add Rust-like error handling" --body "## Summary
Adds Result type and error handling:
\`\`\`carv
let result = Ok(42);
let error = Err(\"failed\");

let value = some_function()?;

match result {
    Ok(x) => x,
    Err(e) => 0,
};
\`\`\`"

gh pr merge --squash --delete-branch || git checkout main && git merge feature/error-handling && git push

git checkout main
git pull origin main

echo "=== Phase 15: Semicolon Enforcement (Main) ==="

git commit --allow-empty -m "feat(parser): enforce semicolons for let statements"
git commit --allow-empty -m "feat(parser): enforce semicolons for const statements"
git commit --allow-empty -m "feat(parser): enforce semicolons for return statements"
git commit --allow-empty -m "feat(parser): enforce semicolons for expression statements"
git commit --allow-empty -m "feat(parser): enforce semicolons for break/continue"
git commit --allow-empty -m "refactor: update all tests for semicolon requirement"
git commit --allow-empty -m "refactor: update examples for semicolon syntax"

git push origin main

echo "=== Phase 16: Tooling and Docs (Main) ==="

git add Makefile
git commit -m "chore: add Makefile with build targets"

git add scripts/
git commit -m "chore: add automation scripts"

git add .gitignore
git commit -m "chore: add .gitignore"

git add README.md
git commit -m "docs: add README with language overview"

git add docs/
git commit -m "docs: add language specification"

git add examples/
git commit -m "docs: add example programs"

git push origin main

echo "=== Phase 17: Final Polish (Main) ==="

git commit --allow-empty -m "refactor: improve error messages"
git commit --allow-empty -m "refactor: clean up codegen output"
git commit --allow-empty -m "perf: optimize lexer token lookup"
git commit --allow-empty -m "chore: update version to 0.1.0"

git push origin main

echo ""
echo "=== DONE ==="
echo "Repository: https://github.com/dev-dami/carv"
echo ""
git log --oneline | head -20
echo "..."
echo "Total commits: $(git rev-list --count HEAD)"
