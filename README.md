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

You can also download prebuilt binaries from the GitHub releases page.

## Quick Start

Initialize a workspace:

```bash
cpx init # create .cpx/config.json and .cpx/templates/main.cpp
```

Create a problem:

```bash
cpx new a # create a/main.cpp and a/samples/{1.in,1.out}
```

Create a problem with a named template:

```bash
cpx new c debug # use .cpx/templates/debug.cpp
```

Create a problem with multiple sample pairs:

```bash
cpx new b 3 # create b with 3 sample pairs
```

Create a problem with both sample count and template:

```bash
cpx new d 2 debug # create d with 2 sample pairs using debug.cpp
```

Add more samples:

```bash
cpx s a # add one more sample pair to problem a
```

Run samples:

```bash
cpx run a # compile a/main.cpp and check it against all samples in a/samples
```

Check the installed version:

```bash
cpx version # print the installed cpx version
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
- `cpx new <problem> [count] [template]`
  Create a problem with an optional named template from `.cpx/templates/`.
  Examples: `cpx new a debug`, `cpx new a 3 debug`
- `cpx s <problem> [count]`
  Add `count` new sample pairs to an existing problem. The default count is `1`.
  Example: `cpx s a 2`
- `cpx run <problem>`
  Compile the problem and compare program output against all sample outputs.
  Example: `cpx run a`
- `cpx version`
  Print the installed `cpx` version.

Problem names should be simple folder names such as `a`, `b`, or `sum`.
Paths like `../a` or `contest/a` are intentionally rejected.

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

You can also add more templates to the same directory:

```text
.cpx/templates/
├── main.cpp
├── debug.cpp
└── graph.cpp
```

Then choose one when creating a problem:

```bash
cpx new a debug
cpx new b 3 graph
```

Template names may be written with or without the `.cpp` extension.

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
