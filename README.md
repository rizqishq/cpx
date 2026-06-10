# cpx

A minimal Go CLI for local competitive programming workflows.

## Features

- `cpx init` creates the `.cpx/` workspace
- `cpx new <problem> [count] [template]` creates a problem folder and sample pairs
- `cpx contest <contest> <count> [samples] [template]` scaffolds contest problem folders from `a` onward
- `cpx s <problem> [count]` appends more sample pairs to an existing problem
- `cpx run <problem>` compiles and runs the problem against saved samples
- `cpx watch <problem>` reruns a problem when its source, config, or sample files change
- `cpx version` prints the current cpx version
- `cpx doctor` checks compiler, workspace, config, and templates
- `.cpx/config.json` controls the active language, compiler standard, default template, compiler flags, and run/watch behavior
- `.cpx/templates/` can contain multiple templates for the same language

## Requirements

- A C++ compiler in `PATH` to run problems (`g++`, `clang++`, or `c++`)

## Install

Unix:

```bash
curl -fsSL https://raw.githubusercontent.com/rizqishq/cpx/master/install.sh | sh
```

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/rizqishq/cpx/master/install.ps1 | iex
```

After installing, verify the binary on your PATH:

```bash
cpx version
```

## Quick start

```bash
cpx init
cpx new a
cpx contest abc 3
cpx s a
cpx run a
cpx watch a
```

This creates a workspace like:

```text
.cpx/
  config.json
  templates/
    main.cpp

a/
  main.cpp
  samples/
    1.in
    1.out

abc/
  a/
    main.cpp
    samples/
      1.in
      1.out
  b/
    main.cpp
    samples/
      1.in
      1.out
  c/
    main.cpp
    samples/
      1.in
      1.out
```

## Configuration

`cpx init` creates `.cpx/config.json`:

```json
{
  "language": "cpp",
  "standard": "c++17",
  "template": "main",
  "compilerFlags": [],
  "runTimeoutMs": 5000,
  "stopOnFirstFail": true,
  "diffContextLines": 1,
  "watchIntervalMs": 500
}
```

Current config fields:

- `language`: currently supports `cpp`
- `standard`: passed to the compiler as `-std=<value>` during `cpx run`
- `template`: default template name used by `cpx new`
- `compilerFlags`: optional extra compiler arguments appended during `cpx run`
- `runTimeoutMs`: runtime timeout per sample in milliseconds during `cpx run`
- `stopOnFirstFail`: whether `cpx run` stops at the first failed sample
- `diffContextLines`: how many context lines to show around the first mismatch in `Diff preview`
- `watchIntervalMs`: polling interval in milliseconds for `cpx watch`

Example with extra compiler flags and custom run/watch behavior:

```json
{
  "language": "cpp",
  "standard": "c++17",
  "template": "main",
  "compilerFlags": ["-Wall", "-Wextra", "-DLOCAL"],
  "runTimeoutMs": 1500,
  "stopOnFirstFail": false,
  "diffContextLines": 0,
  "watchIntervalMs": 100
}
```

## Multiple templates

Templates live under `.cpx/templates/` and are matched by template name plus source extension.

Examples:

- `.cpx/templates/main.cpp`
- `.cpx/templates/fast.cpp`
- `.cpx/templates/debug.cpp`

Use the default template from config:

```bash
cpx new a
```

Override it for one problem:

```bash
cpx new b fast
cpx new c 3 debug
```

## Watch mode

Rerun a problem automatically whenever its source file, workspace config, or sample files change:

```bash
cpx watch a
```

You can also watch problems inside a contest:

```bash
cpx watch abc/a
```

## Contest scaffolding

Create a contest folder with problems `a` through `c`:

```bash
cpx contest abc 3
```

Run or update a problem inside the contest with its path-like problem name:

```bash
cpx run abc/a
cpx s abc/a
```

Create a contest with 2 sample pairs per problem:

```bash
cpx contest abc 3 2
```

Use a specific template for every problem in the contest:

```bash
cpx contest abc 3 fast
cpx contest abc 3 2 debug
```
