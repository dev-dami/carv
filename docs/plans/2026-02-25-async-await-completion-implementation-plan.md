# Async/Await Completion Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make async/await compile and run end-to-end with passing parser/codegen tests and working `carv build` output.

**Architecture:** Keep the existing async state-machine model, but fix correctness at boundaries: parser acceptance for async methods, frame/type emission order, async main entrypoint wiring, and frame-local identifier resolution during async poll generation. Add compile-level regression tests that exercise generated C, not only string matching. Keep implementation minimal and localized to parser/codegen paths.

**Tech Stack:** Go (`testing`), Carv parser/type-checker/codegen, GCC for emitted C verification.

### Task 1: Add Parser RED Tests for Async Methods

**Files:**
- Modify: `pkg/parser/parser_test.go`
- Run: `go test ./pkg/parser -run AsyncMethod -v`

**Step 1: Write failing tests**
1. Add a test asserting `class` methods accept `async fn` and set `MethodDecl.Async = true`.
2. Add a test asserting `impl` methods accept `async fn` and set `MethodDecl.Async = true`.

**Step 2: Run test to verify it fails**
Run: `go test ./pkg/parser -run AsyncMethod -v`  
Expected: FAIL due to parser not accepting `async fn` in class/impl blocks.

### Task 2: Add Codegen RED Tests That Compile Emitted C

**Files:**
- Modify: `pkg/codegen/cgen_test.go`
- Run: `go test ./pkg/codegen -run Async -v`

**Step 1: Write failing tests**
1. Add a test that emits C for async function(s) and asserts frame typedef appears before async function declarations that reference it.
2. Add a test that emits C for `async fn carv_main` and verifies runtime bootstrap references `carv_main` consistently.
3. Add a test that emits C for async local usage after `await` and asserts frame-style access (`f->local`) in generated poll code.
4. Add a test that compiles emitted C via `gcc` and expects success for minimal async program.

**Step 2: Run test to verify it fails**
Run: `go test ./pkg/codegen -run Async -v`  
Expected: FAIL for at least one compile/wiring/identifier issue.

### Task 3: Implement Parser Async Method Support

**Files:**
- Modify: `pkg/parser/parser.go`
- Test: `pkg/parser/parser_test.go`

**Step 1: Write minimal implementation**
1. In class body parse loop, accept `async fn` before method declaration.
2. In impl body parse loop, accept `async fn` before impl method declaration.
3. Set `MethodDecl.Async = true` when method begins with `async fn`.

**Step 2: Verify**
Run: `go test ./pkg/parser -run AsyncMethod -v`  
Expected: PASS.

### Task 4: Implement Async Codegen Correctness Fixes

**Files:**
- Modify: `pkg/codegen/cgen.go`
- Test: `pkg/codegen/cgen_test.go`

**Step 1: Fix declaration/definition order**
1. Emit async frame typedefs before async function declarations that use frame pointer types.
2. Keep poll functions and constructors emitted in an order that compiles in C.

**Step 2: Fix async entrypoint consistency**
1. Detect and bootstrap only `async fn carv_main`.
2. Keep sync `fn main` handling separate and avoid symbol conflicts.

**Step 3: Fix async identifier emission**
1. Ensure async expression/call paths inside poll code use frame-qualified locals/params (`f->...`).

**Step 4: Fix async local collection**
1. Collect frame locals recursively from nested blocks/branches/loops.

**Step 5: Verify targeted tests**
Run: `go test ./pkg/codegen -run Async -v`  
Expected: PASS.

### Task 5: Full Verification

**Files:**
- Run-only verification across repo

**Step 1: Run full Go test suite**
Run: `go test ./...`  
Expected: PASS.

**Step 2: End-to-end build verification**
1. Build minimal async program: `go run ./cmd/carv build <tmp-async-program>.carv`
2. Execute resulting binary and confirm expected output.

**Step 3: Regression confidence**
1. Confirm no parser/type-checker regressions.
2. Confirm emitted async C compiles cleanly with GCC.

