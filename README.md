# cpx

A minimal Go-based CLI for local competitive programming workflows.

## Features

- `cpx init` creates the `.cpx/` workspace
- `cpx new <problem> [count]` creates a new problem folder with one or more sample pairs
- `cpx s <problem> [count]` appends one or more sample pairs to an existing problem
- `cpx run <problem>` compiles and tests C++ samples using the configured standard

## Install

You can install `cpx` directly with Go:

```bash
go install github.com/rizqishq/cpx@latest
```

Or with curl:

```bash
curl -fsSL https://raw.githubusercontent.com/rizqishq/cpx/master/install.sh | sh
```

Then run it as:

```bash
cpx init
cpx new a
cpx new b 3
cpx s a
cpx run a
```

## Config

`cpx init` creates `.cpx/config.json`:

```json
{
  "language": "cpp",
  "standard": "c++17"
}
```

- `language` currently supports `cpp`
- `standard` controls the `-std=...` flag used by `cpx run`

Example:

```json
{
  "language": "cpp",
  "standard": "c++20"
}
```

## Windows

- Recommended toolchain: `MSYS2 UCRT64`
- Make sure `g++` is available in `PATH`
- If you need to override the compiler, set `CXX`

Example:

```powershell
$env:CXX = "g++"
cpx run a
```
