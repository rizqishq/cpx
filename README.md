# cpx

A minimal Go-based CLI for local competitive programming workflows.

## Features

- `cpx init` creates the `.cpx/` workspace
- `cpx new <problem> [count]` creates a new problem folder with one or more sample pairs
- `cpx s <problem> [count]` appends one or more sample pairs to an existing problem
- `cpx run <problem>` compiles and tests C++17 samples

## Install

You can install `cpx` directly with Go:

```bash
go install github.com/rizqishq/cpx@latest
```

Or with curl:

```bash
curl -fsSL https://raw.githubusercontent.com/rizqishq/cpx/main/install.sh | sh
```

Then run it as:

```bash
cpx init
cpx new a
cpx new b 3
cpx s a
cpx run a
```
