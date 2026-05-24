# cpx

A minimal Go-based CLI for local competitive programming workflows.

## Features

- Initialize a local workspace with templates and config
- Create a problem folder with sample input/output files
- Append more sample pairs to an existing problem
- Compile and run C++ samples using the configured language standard

## Install

You can install `cpx` directly with Go:

```bash
go install github.com/rizqishq/cpx@latest
```

Or with curl:

```bash
curl -fsSL https://raw.githubusercontent.com/rizqishq/cpx/master/install.sh | sh
```

## Quick Start

Initialize a workspace:

```bash
cpx init # create .cpx/config.json and .cpx/templates/main.cpp
```

Create a problem:

```bash
cpx new a # create a/main.cpp and a/samples/{1.in,1.out}
```

Create a problem with multiple sample pairs:

```bash
cpx new b 3 # create b with 3 sample pairs
```

Add more samples:

```bash
cpx s a # add one more sample pair to problem a
```

Run samples:

```bash
cpx run a # compile a/main.cpp and check it against all samples in a/samples
```

After `cpx new a`, the workspace looks like this:

```text
.
├── .cpx
│   ├── config.json
│   └── templates
│       └── main.cpp
└── a
    ├── main.cpp
    └── samples
        ├── 1.in
        └── 1.out
```

Edit `a/main.cpp`, fill the sample files, then run `cpx run a`.

Example:

`a/main.cpp`

```cpp
#include <iostream>
using namespace std;

int main() {
    int a, b;
    cin >> a >> b;
    cout << a + b;
    return 0;
}
```

`a/samples/1.in`

```text
2 3
```

`a/samples/1.out`

```text
5
```

Then run:

```bash
cpx run a # this should print PASS for sample 1
```

## Commands

- `cpx init`
  Initialize `.cpx/` with the default config and template.
  Run this once before using `new`, `s`, or `run` in a workspace.
- `cpx new <problem> [count]`
  Create a problem folder and `count` sample pairs. The default count is `1`.
  Example: `cpx new a`
- `cpx s <problem> [count]`
  Add `count` new sample pairs to an existing problem. The default count is `1`.
  Example: `cpx s a 2`
- `cpx run <problem>`
  Compile the problem and compare program output against all sample outputs.
  Example: `cpx run a`

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

Example: compile with C++20

```json
{
  "language": "cpp",
  "standard": "c++20"
}
```

If the config file is invalid, `cpx run` will fail with a config error instead of silently ignoring it.

## Template

`cpx init` also creates `.cpx/templates/main.cpp`.

New problems copy that template into `<problem>/main.cpp`, so you can customize the template once and reuse it for future problems.

Typical workflow:

```bash
cpx init                    # create the workspace files
$EDITOR .cpx/templates/main.cpp
cpx new a                   # a/main.cpp now uses your edited template
```

## Windows

- Recommended toolchain: `MSYS2 UCRT64`
- Make sure `g++` is available in `PATH`
- If you need to override the compiler, set `CXX`

Example:

```powershell
$env:CXX = "g++" # force cpx to use g++
cpx run a
```

You can also point `CXX` at a specific compiler path:

```powershell
$env:CXX = "C:\msys64\ucrt64\bin\g++.exe" # use this exact compiler
cpx run a
```
