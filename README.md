# cpx

A minimal Go CLI for local competitive programming workflows.

## Features

- `cpx init` creates the `.cpx/` workspace
- `cpx new <problem> [count] [template]` creates a problem folder and sample pairs
- `cpx s <problem> [count]` appends more sample pairs to an existing problem
- `cpx run <problem>` compiles and runs the problem against saved samples
- `cpx version` prints the current cpx version
- `cpx doctor` checks compiler, workspace, config, and templates
- `.cpx/config.json` controls the active language, compiler standard, and default template
- `.cpx/templates/` can contain multiple templates for the same language

## Requirements

- Go to build `cpx`
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

## Quick start

```bash
cpx init
cpx new a
cpx s a
cpx run a
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
```

## Configuration

`cpx init` creates `.cpx/config.json`:

```json
{
  "language": "cpp",
  "standard": "c++17",
  "template": "main"
}
```

Current config fields:

- `language`: currently supports `cpp`
- `standard`: passed to the compiler as `-std=<value>` during `cpx run`
- `template`: default template name used by `cpx new`

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
