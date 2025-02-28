# Brio: Code Snippet Extraction CLI

**Brio** is a simple but super handy command-line tool for pulling specific code snippets out of your codebase. It works by scanning your files for sections marked with special start/end tags that include JSON metadata. Instead of copying and pasting from entire files, you can just grab exactly what you need.

Where Brio really shines is when you’re working with Large Language Models (LLMs) or trying to focus on specific parts of your code. Whether you’re building prompts, fine-tuning models, debugging, or just sharing snippets with your team, Brio makes it quick and painless. It’s like having a custom filter for your code that pulls out only what matters.

---

## Table of Contents

- [Installation](#installation)
- [Getting Started](#getting-started)
- [Usage](#usage)
    - [Extract Command](#extract-command)
    - [Annotation Format](#annotation-format)
- [Examples](#examples)
- [Advanced Tips](#advanced-tips)
- [Contributing](#contributing)
- [License](#license)

---

## Installation

1. **CURL**

```bash
curl -fsSL https://raw.githubusercontent.com/rechati/brio/main/install/install.sh | bash
```

After this, you can run `brio` from any directory.

---

## Getting Started

1. **Annotate your code** with `# start: {...}` and `# end: {...}` comments.
2. **Tag these snippets** with JSON containing categories and domains (e.g., `"foundation": ["messages"]`).
3. **Run the `brio` CLI** to extract only the code that matches your desired categories.

---

## Usage

Run `brio --help` to see the top-level usage and available commands:

```bash
brio --help
```

### Extract Command

The core subcommand is `extract`, which searches files for tagged code snippets and prints them in Markdown by default.

Basic command:

```bash
brio extract [flags]
```

#### Flags

- **-d, --dir** (default: `"."`)  
  The root directory to scan.

- **-f, --files** (default: `"*.py"`)  
  A file pattern (glob) for matching relevant files (e.g., `*.py`, `*.go`, etc.).

- **-c, --categories**  
  A comma-separated list (optionally containing colons) to filter which tags to extract.

Examples of `--categories` usage:
- `foundation`
- `foundation,tests`
- `messages:foundation,tests`

---

## Annotation Format

Brio looks for ">" and "<" comments formatted as follows:

```python
# >: { "foundation": ["messages"], "model": ["messages"] }
< ... lines of code ... >
# <: { "foundation": ["messages"] }
```

#### Rules

1. The `# >:` or `# <:` must be followed by a **JSON object** with the categories you want to associate with the snippet.
2. The snippet content is every line **between** the start and end tags.
3. Categories are stored as key-value pairs (`key = category`, `value = array of domains`), for example `"foundation": ["messages"]`.
4. Brio uses these categories to decide whether a snippet matches your CLI filter.

---

## Examples

### Extract All Snippets (No Category Specified)

```bash
brio extract
```

- Scans the current directory for supported files by default.
- Prints all tagged snippets found.

### Extract Specific Categories

```bash
brio extract --categories "foundation"
```

- Extracts snippets containing the `foundation` category in their JSON metadata.

### Extract Multiple Categories

```bash
brio extract --categories "foundation,tests"
```

- Extracts snippets with either `foundation` or `tests`.

### Extract Category with Domain

```bash
brio extract --categories "messages:foundation,tests"
```

- Looks for tagged snippets that have `"foundation": ["messages"]` or `"tests": ["messages"]`.

### Scan a Specific Directory

```bash
brio extract --dir /path/to/my/project --categories "foundation"
```

- Recursively scans all subdirectories, matching supported plugins files extension.

### Specify File Pattern

```bash
brio extract --files "*.go" --categories "foundation"
```

- Only scans `.go` files for relevant snippets.

---

## Advanced Tips

1. **Nested Snippets**  
   If you embed a `# >:` inside another snippet, Brio will treat them **separately**. Overlapping snippets can be tricky to parse, so consider carefully how you tag nested sections.

2. **Merging Categories**  
   By default, Brio uses only the **start tag’s** categories, unless you modify the code to merge with the end tag’s JSON. If that is desirable, you can adjust the snippet creation logic.

3. **Extending Output Formats**  
   By default, snippets print in **Markdown**. You could add flags (`--format=json`, `--format=plain`, etc.) to integrate Brio with other tools or pipelines.

---

## Contributing

1. **Fork** the repo & create your feature branch (`git checkout -b feature/amazing-feature`)
2. **Commit** your changes (`git commit -m 'Add some amazing feature'`)
3. **Push** to the branch (`git push origin feature/amazing-feature`)
4. **Open a Pull Request**

All contributions, big or small, are welcome!

---

## License

This project is licensed under the MIT License.
