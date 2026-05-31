# Markdown Formatting Guide

This document serves as a standard reference and styling guide for formatting Markdown files in this repository. Consistent formatting ensures that documentation is clean, readable, and highly professional when viewed in code editors, on GitHub, or in generated documentation sites.

---

## Headers

Headers are defined using the hash (`#`) character followed by a space. The number of hashes determines the header level (1 through 6).

### Guidelines
- **Use a single `<h1>`** per document, typically at the very top as the document title.
- **Maintain a logical hierarchy**. Do not skip header levels (e.g., do not follow an `## H2` directly with an `#### H4`).
- **Include a single space** after the `#` characters.
- **Capitalize headers properly** (Title Case or Sentence Case, but be consistent within the document).
- **Avoid verbose branding in headers**: Do not repeat the full system name (e.g., `KK Version Control System`) in document titles or headers. 
  - For generic topics, omit prefixes entirely (e.g., `# Installation & Setup` or `# Markdown Formatting Guide`).
  - If a prefix is helpful to clarify context, keep it short (e.g., `# KK Commit Message Guide` instead of `# KK Version Control System Commit Message Guide`).

### Syntax
```markdown
# Header 1 (Document Title)
## Header 2 (Major Sections)
### Header 3 (Sub-sections)
#### Header 4 (Detailed Sub-sections)
##### Header 5
###### Header 6
```

---

## Body and Text Styling

Standard text is written as normal paragraphs. Separate paragraphs with a blank line.

### Emphasis and Styling
Use asterisks (`*`) or underscores (`_`) to style text. 

| Style | Syntax | Example | Result |
| :--- | :--- | :--- | :--- |
| **Bold** | `**text**` or `__text__` | `**Important**` | **Important** |
| *Italics* | `*text*` or `_text_` | `*Emphasis*` | *Emphasis* |
| ***Bold & Italic*** | `***text***` or `___text___` | `***Critical***` | ***Critical*** |
| ~~Strikethrough~~ | `~~text~~` | `~~Deprecated~~` | ~~Deprecated~~ |
| `Code` (Inline) | `` `code` `` | `` `git commit` `` | `git commit` |

> [!TIP]
> Prefer using asterisks (`*` and `**`) over underscores (`_` and `__`) for text styling, as they are easier to distinguish in plain text code editors.

---

## Lists

Markdown supports ordered (numbered), unordered (bulleted), and task (checklist) lists.

### Unordered Lists
Use asterisks (`*`), hyphens (`-`), or plus signs (`+`) for bullets. Hyphens are preferred for consistency.
To nest lists, indent the sub-list by 2 or 4 spaces.

#### Syntax
```markdown
- Main item 1
- Main item 2
  - Sub-item 2a
  - Sub-item 2b
- Main item 3
```

### Ordered Lists
Use numbers followed by a period and a space. You can use sequential numbers or write `1.` for every item (Markdown renders them sequentially automatically).

#### Syntax
```markdown
1. First step
2. Second step
   1. Sub-step A
   2. Sub-step B
3. Third step
```

### Task Lists / Checklists
Task lists allow you to render interactive or visual checklists.

#### Syntax
```markdown
- [x] Completed task
- [/] In-progress task (supported in custom dashboards/tools)
- [ ] Uncompleted task
```

---

## Links

Links are written using brackets for the label text and parentheses for the URL/destination.

### Types of Links

1. **External Link**: Points to an external website.
   - Syntax: `[Google](https://google.com)`
   - Result: [Google](https://google.com)

2. **Internal Link**: Points to another file in the repository (relative path).
   - Syntax: `[Commit Message Guide](commit-message-guide.md)`
   - Result: [Commit Message Guide](commit-message-guide.md)

3. **Anchor Link**: Points to a specific section/header in the current document. Headers are auto-slugified (all lowercase, spaces replaced by hyphens).
   - Syntax: `[Go to Lists Section](#lists)`
   - Result: [Go to Lists Section](#lists)

4. **Reference-Style Link**: Useful when you reference the same URL multiple times or want to keep paragraphs uncluttered.
   - Syntax:
     ```markdown
     Visit [GitHub][1] or read the [Docs][2].

     [1]: https://github.com
     [2]: https://docs.github.com
     ```

---

## Images

Images use a syntax similar to links, prefixed with an exclamation mark (`!`).

### Guidelines
- Always provide a descriptive alternative text (inside the square brackets) for accessibility.
- Prefer absolute repository paths or relative paths that resolve correctly in your target renderer.
- To display images on GitHub, use relative paths to image files stored within the repository.

### Syntax
```markdown
![Logo Description](/icon/logo.png)
```

---

## Code Samples

Markdown supports inline code and blocks of code.

### Inline Code
Use a single backtick (`` ` ``) to wrap short code snippets, commands, variables, or file paths within a sentence.

- Syntax: ``Use the `make build` command to compile.``
- Result: Use the `make build` command to compile.

### Code Blocks
Use triple backticks (`` ``` ``) before and after your code block. Always specify the language identifier immediately after the first set of backticks for syntax highlighting.

#### Go Sample
````markdown
```go
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
```
````

#### Shell / Terminal Commands
Use `bash`, `sh`, or `powershell` for terminal sequences. Do not prefix single commands with `$` unless showing both input and output.
````markdown
```bash
go build -v -o kk ./cmd/kk
```
````

#### Diffs
Use the `diff` language identifier to highlight additions and deletions in code or configuration.
````markdown
```diff
- oldValue := 10
+ newValue := 20
  unchangedValue := 100
```
````

---

## Tables

Tables are created using pipes (`|`) to separate columns and hyphens (`-`) to define headers. 

### Alignment
Use colons (`:`) within the header separator row to align columns:
- `:---` Left-aligned (default)
- `:---:` Centered
- `---:` Right-aligned

### Syntax
```markdown
| Parameter | Type | Default | Description |
| :--- | :---: | :---: | :--- |
| `name` | string | `""` | Name of the workspace target |
| `timeout` | int | `30` | Execution timeout in seconds |
| `verbose` | bool | `false` | Enable verbose debug logs |
```

### Result
| Parameter | Type | Default | Description |
| :--- | :---: | :---: | :--- |
| `name` | string | `""` | Name of the workspace target |
| `timeout` | int | `30` | Execution timeout in seconds |
| `verbose` | bool | `false` | Enable verbose debug logs |

---

## Quotes and Alerts (Admonitions)

### Blockquotes
Use the `>` character before a paragraph to create a blockquote. Blockquotes are great for quotes, background context, or secondary explanations.

#### Syntax
```markdown
> "Perfection is achieved, not when there is nothing more to add, but when there is nothing left to take away."
> — Antoine de Saint-Exupéry
```

### Alerts / Callouts
Many modern Markdown processors (including GitHub and VS Code extensions) support formatted alert boxes. These are defined by putting an alert tag in the first line of a blockquote.

#### Syntax
```markdown
> [!NOTE]
> Useful information that users should know when reading the documentation.

> [!TIP]
> Helpful advice or shortcuts to make things easier or more efficient.

> [!IMPORTANT]
> Crucial information necessary for users to succeed or prevent mistakes.

> [!WARNING]
> Urgent info about potential pitfalls, deprecated features, or compatibility.

> [!CAUTION]
> Negative consequences or risks associated with an action (e.g., data loss).
```

---

## Horizontal Rules

Horizontal rules (dividers) are created using three or more asterisks (`***`), hyphens (`---`), or underscores (`___`) on a line by themselves.

- Syntax: `---`
- Result:
---

## Advanced Formatting

### Escaping Characters
If you want to display a literal Markdown formatting character (such as a backtick, asterisk, or square bracket), escape it with a backslash (`\`).

- Syntax: `\*This is literal text, not italics\*`
- Result: \*This is literal text, not italics\*

### HTML Elements
Standard Markdown supports inline HTML. This can be useful for advanced layouts, such as coloring text, styling specific elements, or centering text.

- Syntax: `<p align="center">This paragraph is centered using HTML.</p>`
- Result: <p align="center">This paragraph is centered using HTML.</p>
