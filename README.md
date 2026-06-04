# cpx

A minimal Go CLI for local competitive programming workflows.

## Features

- `cpx init` creates the `.cpx/` workspace
- `cpx new <problem> [count]` creates a problem folder and sample pairs
- `cpx s <problem> [count]` appends more sample pairs to an existing problem
- `cpx run <problem>` compiles and runs the problem against saved samples
- `.cpx/config.json` controls the active language and compiler standard

## Requirements

- Go to build `cpx`
- A C++ compiler in `PATH` to run problems (`g++`, `clang++`, or `c++`)

## Install

Install with Go:

```bash
go install github.com/rizqishq/cpx@latest
```

Or with curl:

```bash
curl -fsSL https://raw.githubusercontent.com/rizqishq/cpx/master/install.sh | sh
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
  "standard": "c++17"
}
```

Current config fields:

- `language`: currently supports `cpp`
- `standard`: passed to the compiler as `-std=<value>` during `cpx run`
