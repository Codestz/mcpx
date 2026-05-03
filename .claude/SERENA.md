# serena

`mcpx serena <tool> --flags` — 21 tools. Use `--help` / `--example` / `--validate-args` per tool.

## Tool selector

| Tool | What it does | Required |
|---|---|---|
| `list_dir` | Lists files and directories in the given directory (optionally with recursion). | `--relative_path` `--recursive` |
| `find_file` | Finds non-gitignored files matching the given file mask within the given relative path. | `--file_mask` `--relative_path` |
| `search_for_pattern` | Offers a flexible search for arbitrary patterns in the codebase, including the | `--substring_pattern` |
| `get_symbols_overview` | Use this tool to get a high-level understanding of the code symbols in a file. | `--relative_path` |
| `find_symbol` | Retrieves information on all symbols/code entities (classes, methods, etc.) based on the given name path pattern. | `--name_path_pattern` |
| `find_referencing_symbols` | Finds references to the symbol at the given `name_path`. | `--name_path` `--relative_path` |
| `replace_symbol_body` | Replaces the body of the symbol with the given `name_path`. | `--name_path` `--relative_path` `--body` |
| `insert_after_symbol` | Inserts the given body/content after the end of the definition of the given symbol (via the symbol's location). | `--name_path` `--relative_path` `--body` |
| `insert_before_symbol` | Inserts the given content before the beginning of the definition of the given symbol (via the symbol's location). | `--name_path` `--relative_path` `--body` |
| `rename_symbol` | Renames the symbol with the given `name_path` to `new_name` throughout the entire codebase. | `--name_path` `--relative_path` `--new_name` |
| `write_memory` | Write information (utf-8-encoded) about this project that can be useful for future tasks to a memory in md format. | `--memory_name` `--content` |
| `read_memory` | Reads the contents of a memory. | `--memory_name` |
| `list_memories` | Lists available memories, optionally filtered by topic. | — |
| `delete_memory` | Delete a memory, only call if instructed explicitly or permission was granted by the user. | `--memory_name` |
| `rename_memory` | Rename or move a memory, use "/" in the name to organize into topics. | `--old_name` `--new_name` |
| `edit_memory` | Replaces content matching a regular expression in a memory. | `--memory_name` `--needle` `--repl` `--mode` |
| `activate_project` | Activates the project with the given name or path. | `--project` |
| `get_current_config` | Print the current configuration of the agent, including the active and available projects, tools, contexts, and modes. | — |
| `check_onboarding_performed` | Checks whether project onboarding was already performed. | — |
| `onboarding` | Call this tool if onboarding was not performed yet. | — |
| `initial_instructions` | Provides the 'Serena Instructions Manual', which contains essential information on how to use the Serena toolbox. | — |

## Edit-tool safety

For `replace_symbol_body`, body must start AFTER the declaration keyword — the keyword is preserved by the server. mcpx warns on duplicates and (for `.go`) re-parses the file post-edit.

| Language | ✓ correct | ✗ wrong |
|---|---|---|
| Go | `Foo struct{...}` / `Foo() error {...}` | `type Foo...` / `func Foo...` |
| Python | `foo():\n    ...` | `def foo():` |
| TS / JS | `bar() { ... }` | `function bar()` / `class Bar` |
| Rust | `foo() { ... }` | `fn foo()` / `struct Foo` |

## Notes

- Symbol-aware lookups (`find_symbol`) are faster and more accurate than text search (`search_for_pattern`) for code symbols — prefer them.

## Compact reference

- `list_dir` --max_answer_chars <int> --recursive <bool>* --relative_path <string>* --skip_ignored_files <bool>
- `find_file` --file_mask <string>* --relative_path <string>*
- `search_for_pattern` --context_lines_after <int> --context_lines_before <int> --max_answer_chars <int> --paths_exclude_glob <string> --paths_include_glob <string> --relative_path <string> --restrict_search_to_code_files <bool> --substring_pattern <string>*
- `get_symbols_overview` --depth <int> --max_answer_chars <int> --relative_path <string>*
- `find_symbol` --depth <int> --exclude_kinds <json[]> --include_body <bool> --include_info <bool> --include_kinds <json[]> --max_answer_chars <int> --name_path_pattern <string>* --relative_path <string> --substring_matching <bool>
- `find_referencing_symbols` --exclude_kinds <json[]> --include_kinds <json[]> --max_answer_chars <int> --name_path <string>* --relative_path <string>*
- `replace_symbol_body` --body <string>* --name_path <string>* --relative_path <string>*
- `insert_after_symbol` --body <string>* --name_path <string>* --relative_path <string>*
- `insert_before_symbol` --body <string>* --name_path <string>* --relative_path <string>*
- `rename_symbol` --name_path <string>* --new_name <string>* --relative_path <string>*
- `write_memory` --content <string>* --max_chars <int> --memory_name <string>*
- `read_memory` --memory_name <string>*
- `list_memories` --topic <string>
- `delete_memory` --memory_name <string>*
- `rename_memory` --new_name <string>* --old_name <string>*
- `edit_memory` --allow_multiple_occurrences <bool> --memory_name <string>* --mode <string>* --needle <string>* --repl <string>*
- `activate_project` --project <string>*
- `get_current_config`
- `check_onboarding_performed`
- `onboarding`
- `initial_instructions`

`*` = required
