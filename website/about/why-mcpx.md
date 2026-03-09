# Why MCPX

## The Problem

The Model Context Protocol gave AI systems a standard way to call tools — semantic code search, structured reasoning, databases, web browsing. Hundreds of MCP servers exist today.

But every one comes with a hidden cost.

**Loading MCP servers into an AI session is ruinously expensive.**

Each server dumps its full tool schema into the conversation context. Five MCP servers? That's 50,000-100,000 tokens consumed before the AI does anything. The AI carries that weight for the entire session — whether it uses those tools or not.

| Servers Loaded | Context Cost | Impact |
|---------------|-------------|--------|
| 3 servers | ~30-60K tokens | Noticeable context reduction |
| 5 servers | ~50-100K tokens | Significant. Less room for code. |
| 10 servers | ~100K+ tokens | Context nearly full before work begins |

Servers also initialize slowly. Some take seconds to spin up. Every new conversation pays that startup cost again.

**The AI is drowning in tool definitions instead of solving your problem.**

## The Insight

What if the AI didn't need to load MCP servers at all?

AI agents already know how to use the terminal. They run `grep`. They run `git`. They run `curl`. They compose commands with pipes. They read stdout.

Give MCP servers the same interface and the problem disappears.

## The Solution

mcpx converts MCP servers into CLI commands. The AI discovers and calls tools through Bash — the same way it runs any other command.

```bash
# No schema loaded. No initialization. Just a shell command.
mcpx serena search_symbol --name "UserAuth"
```

**On-demand instead of upfront:**

- Native MCP: 100K tokens loaded immediately, carried forever
- mcpx: 0 tokens upfront, small per-call cost only when used

The AI explores tools lazily. It calls `mcpx list serena` only when it needs Serena. It calls `--help` only when it needs a specific tool's flags. Context stays clean. The AI stays fast.

## What This Unlocks

### Context efficiency

The AI's context window is for reasoning, not for carrying tool schemas. mcpx moves tool definitions out of context and into the filesystem where they belong.

### Speed

Daemon mode keeps heavy servers warm between calls. The AI doesn't wait for LSP initialization every time it needs code search.

### Composability

AI agents are excellent at chaining shell commands. Two MCP servers that know nothing about each other can be piped together:

```bash
mcpx serena find_symbol --name "PaymentService" --json \
  | mcpx sequential-thinking think --problem - --total_thoughts 5
```

### Scalability

Add 20 MCP servers to a project. The AI pays zero tokens for the ones it doesn't use. With native MCP, 20 servers would consume the entire context window.

## What MCPX Is Not

**Not a replacement for MCP.** MCP is the protocol. mcpx is a better delivery mechanism.

**Not an AI client.** It speaks MCP protocol but never calls an LLM.

**Not just for humans.** It's designed for AI agents first. Humans benefit too, but the primary user is the AI in your terminal.

## Why Go

Single binary. Zero runtime dependency. Sub-millisecond startup.

When an AI agent runs `mcpx` in a Bash call, startup cost matters. Go starts in under 5ms. The binary ships as a single file. No Python. No Node. No runtime tax.
