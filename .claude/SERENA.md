# serena (21 tools)

Usage: `mcpx serena <tool> --flags`

**list_dir** — Lists files and directories in the given directory (optionally with recursion).
  --max_answer_chars <int>, --recursive <bool> *, --relative_path <string> *, --skip_ignored_files <bool>

**find_file** — Finds non-gitignored files matching the given file mask within the given relative path.
  --file_mask <string> *, --relative_path <string> *

**search_for_pattern** — Offers a flexible search for arbitrary patterns in the codebase, including the
  --context_lines_after <int>, --context_lines_before <int>, --max_answer_chars <int>, --paths_exclude_glob <string>, --paths_include_glob <string>, --relative_path <string>, --restrict_search_to_code_files <bool>, --substring_pattern <string> *

**get_symbols_overview** — Use this tool to get a high-level understanding of the code symbols in a file.
  --depth <int>, --max_answer_chars <int>, --relative_path <string> *

**find_symbol** — Retrieves information on all symbols/code entities (classes, methods, etc.) based on the given name path pattern.
  --depth <int>, --exclude_kinds <json[]>, --include_body <bool>, --include_info <bool>, --include_kinds <json[]>, --max_answer_chars <int>, --name_path_pattern <string> *, --relative_path <string>, --substring_matching <bool>

**find_referencing_symbols** — Finds references to the symbol at the given `name_path`.
  --exclude_kinds <json[]>, --include_kinds <json[]>, --max_answer_chars <int>, --name_path <string> *, --relative_path <string> *

**replace_symbol_body** — Replaces the body of the symbol with the given `name_path`.
  --body <string> *, --name_path <string> *, --relative_path <string> *

**insert_after_symbol** — Inserts the given body/content after the end of the definition of the given symbol (via the symbol's location).
  --body <string> *, --name_path <string> *, --relative_path <string> *

**insert_before_symbol** — Inserts the given content before the beginning of the definition of the given symbol (via the symbol's location).
  --body <string> *, --name_path <string> *, --relative_path <string> *

**rename_symbol** — Renames the symbol with the given `name_path` to `new_name` throughout the entire codebase.
  --name_path <string> *, --new_name <string> *, --relative_path <string> *

**write_memory** — Write information (utf-8-encoded) about this project that can be useful for future tasks to a memory in md format.
  --content <string> *, --max_chars <int>, --memory_name <string> *

**read_memory** — Reads the contents of a memory.
  --memory_name <string> *

**list_memories** — Lists available memories, optionally filtered by topic.
  --topic <string>

**delete_memory** — Delete a memory, only call if instructed explicitly or permission was granted by the user.
  --memory_name <string> *

**rename_memory** — Rename or move a memory, use "/" in the name to organize into topics.
  --new_name <string> *, --old_name <string> *

**edit_memory** — Replaces content matching a regular expression in a memory.
  --allow_multiple_occurrences <bool>, --memory_name <string> *, --mode <string> *, --needle <string> *, --repl <string> *

**activate_project** — Activates the project with the given name or path.
  --project <string> *

**get_current_config** — Print the current configuration of the agent, including the active and available projects, tools, contexts, and modes.

**check_onboarding_performed** — Checks whether project onboarding was already performed.

**onboarding** — Call this tool if onboarding was not performed yet.

**initial_instructions** — Provides the 'Serena Instructions Manual', which contains essential information on how to use the Serena toolbox.

`*` = required
