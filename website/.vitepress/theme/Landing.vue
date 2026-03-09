<script setup>
import { ref, onMounted } from 'vue'

const lines = [
  { prompt: true, text: 'mcpx list', delay: 0 },
  { prompt: false, text: '  serena              uvx (daemon)', delay: 600 },
  { prompt: false, text: '  sequential-thinking npx', delay: 700 },
  { prompt: false, text: '', delay: 800 },
  { prompt: true, text: 'mcpx ping serena', delay: 1200 },
  { prompt: false, text: '  serena: \x1bok\x1b (21 tools, 47ms)', delay: 2000 },
  { prompt: false, text: '', delay: 2100 },
  { prompt: true, text: 'mcpx serena find_symbol --name "Auth" --json', delay: 2600 },
  { prompt: false, text: '  { "name": "AuthService", "file": "src/auth.go", "kind": "class" }', delay: 3400 },
]

const visibleLines = ref(0)
const mounted = ref(false)

onMounted(() => {
  mounted.value = true
  lines.forEach((line, i) => {
    setTimeout(() => {
      visibleLines.value = i + 1
    }, line.delay)
  })
})
</script>

<template>
  <div class="landing">
    <!-- Hero -->
    <section class="hero">
      <pre class="ascii">
 __  __  ____ ____  __  __
|  \/  |/ ___|  _ \ \ \/ /
| |\/| | |   | |_) | \  /
| |  | | |___|  __/  /  \
|_|  |_|\____|_|    /_/\_\</pre>
      <p class="tagline">MCP servers as CLI tools.</p>
      <p class="sub">Stop loading tool schemas into context.<br>Give the AI a terminal command instead.</p>

      <div class="actions">
        <a href="/getting-started/installation" class="btn primary">Get Started</a>
        <a href="https://github.com/codestz/mcpx" class="btn secondary" target="_blank">GitHub</a>
      </div>

      <div class="install-line">
        <code>go install github.com/codestz/mcpx/cmd/mcpx@latest</code>
      </div>
    </section>

    <!-- Before / After -->
    <section class="compare">
      <div class="compare-grid">
        <div class="compare-card bad">
          <div class="compare-label">Native MCP</div>
          <div class="compare-terminal">
            <div class="term-line dim"># .mcp.json — loaded at session start</div>
            <div class="term-line">"serena": { ... }<span class="dim">         # ~20K tokens</span></div>
            <div class="term-line">"sequential-thinking": { ... }<span class="dim"> # ~5K tokens</span></div>
            <div class="term-line">"filesystem": { ... }<span class="dim">    # ~10K tokens</span></div>
            <div class="term-line">"github": { ... }<span class="dim">        # ~30K tokens</span></div>
            <div class="term-line">"brave-search": { ... }<span class="dim">  # ~4K tokens</span></div>
            <div class="term-line dim">&nbsp;</div>
            <div class="term-line accent-bad">Total: ~69K tokens before any work.</div>
          </div>
        </div>
        <div class="compare-card good">
          <div class="compare-label">MCPX</div>
          <div class="compare-terminal">
            <div class="term-line dim"># CLAUDE.md — 3 lines</div>
            <div class="term-line">Use `mcpx list` to discover tools.</div>
            <div class="term-line">Use `mcpx &lt;server&gt; &lt;tool&gt; --help`.</div>
            <div class="term-line">Call tools via Bash as needed.</div>
            <div class="term-line dim">&nbsp;</div>
            <div class="term-line dim">&nbsp;</div>
            <div class="term-line dim">&nbsp;</div>
            <div class="term-line accent-good">Total: 0 tokens. Tools called on demand.</div>
          </div>
        </div>
      </div>
    </section>

    <!-- Stats -->
    <section class="stats">
      <div class="stat">
        <div class="stat-number">0</div>
        <div class="stat-label">tokens upfront</div>
      </div>
      <div class="stat">
        <div class="stat-number">&lt;5ms</div>
        <div class="stat-label">startup time</div>
      </div>
      <div class="stat">
        <div class="stat-number">1</div>
        <div class="stat-label">binary, zero deps</div>
      </div>
      <div class="stat">
        <div class="stat-number">&infin;</div>
        <div class="stat-label">servers, same cost</div>
      </div>
    </section>

    <!-- Terminal demo -->
    <section class="demo">
      <h2>See it work</h2>
      <div class="terminal">
        <div class="terminal-header">
          <span class="terminal-dot"></span>
          <span class="terminal-dot"></span>
          <span class="terminal-dot"></span>
          <span class="terminal-title">terminal</span>
        </div>
        <div class="terminal-body">
          <div
            v-for="(line, i) in lines"
            :key="i"
            class="term-line"
            :class="{ visible: i < visibleLines }"
          >
            <span v-if="line.prompt" class="prompt">$</span>
            <span
              v-if="line.prompt"
              class="command"
            >{{ line.text }}</span>
            <span v-else class="output" v-html="line.text.replace(/\x1bok\x1b/, '<span class=\'ok\'>ok</span>')"></span>
          </div>
          <div class="cursor" :class="{ blink: visibleLines >= lines.length }">_</div>
        </div>
      </div>
    </section>

    <!-- 3 steps -->
    <section class="steps">
      <h2>Three commands to start</h2>
      <div class="steps-grid">
        <div class="step">
          <div class="step-num">1</div>
          <div class="step-content">
            <div class="step-title">Install</div>
            <code>go install github.com/codestz/mcpx/cmd/mcpx@latest</code>
          </div>
        </div>
        <div class="step">
          <div class="step-num">2</div>
          <div class="step-content">
            <div class="step-title">Configure</div>
            <code>mcpx init</code>
          </div>
        </div>
        <div class="step">
          <div class="step-num">3</div>
          <div class="step-content">
            <div class="step-title">Call</div>
            <code>mcpx serena find_symbol --name "Auth"</code>
          </div>
        </div>
      </div>
    </section>

    <!-- Features — minimal -->
    <section class="features">
      <div class="feature-grid">
        <div class="feature">
          <h3>On-Demand Discovery</h3>
          <p>Tools discovered lazily with <code>mcpx list</code> and <code>--help</code>. The AI only pays for what it uses.</p>
        </div>
        <div class="feature">
          <h3>UNIX Composability</h3>
          <p>Every MCP tool becomes a CLI command. Pipe between servers, redirect, compose. The AI already knows how.</p>
        </div>
        <div class="feature">
          <h3>Daemon Mode</h3>
          <p>Heavy servers stay warm between calls via unix socket. Zero spawn cost after first invocation.</p>
        </div>
        <div class="feature">
          <h3>Secure by Default</h3>
          <p>No shell expansion. Secrets from OS keychain. Strict variable parsing. Zero injection surface.</p>
        </div>
        <div class="feature">
          <h3>Any MCP Server</h3>
          <p>If it speaks MCP protocol, mcpx wraps it. Zero changes to the server required.</p>
        </div>
        <div class="feature">
          <h3>Single Binary</h3>
          <p>Written in Go. No runtime dependencies. Ships as one file. Sub-millisecond startup.</p>
        </div>
      </div>
    </section>

    <!-- CTA -->
    <section class="cta">
      <h2>MCP tools belong in the terminal.</h2>
      <p>mcpx puts them there.</p>
      <div class="actions">
        <a href="/getting-started/installation" class="btn primary">Get Started</a>
        <a href="/about/why-mcpx" class="btn secondary">Read the story</a>
      </div>
    </section>
  </div>
</template>

<style scoped>
.landing {
  font-family: var(--vp-font-family-base);
  color: var(--vp-c-text-1);
}

/* ── Hero ── */
.hero {
  text-align: center;
  padding: 80px 24px 60px;
  max-width: 720px;
  margin: 0 auto;
}

.ascii {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', 'Courier New', monospace;
  font-size: clamp(0.55rem, 2.5vw, 0.95rem);
  line-height: 1.3;
  letter-spacing: 0.02em;
  color: var(--vp-c-text-1);
  margin: 0 auto 32px;
  text-align: center;
  white-space: pre;
}

.tagline {
  font-size: 1.6rem;
  font-weight: 600;
  margin: 0 0 12px;
  letter-spacing: -0.02em;
}

.sub {
  font-size: 1.1rem;
  color: var(--vp-c-text-2);
  line-height: 1.6;
  margin: 0 0 32px;
}

.actions {
  display: flex;
  gap: 12px;
  justify-content: center;
  flex-wrap: wrap;
  margin-bottom: 24px;
}

.btn {
  display: inline-block;
  padding: 10px 24px;
  border-radius: 6px;
  font-size: 0.95rem;
  font-weight: 500;
  text-decoration: none;
  transition: all 0.15s ease;
}

.btn.primary {
  background: var(--vp-c-text-1);
  color: var(--vp-c-bg);
}

.btn.primary:hover {
  opacity: 0.85;
}

.btn.secondary {
  border: 1px solid var(--vp-c-border);
  color: var(--vp-c-text-1);
  background: transparent;
}

.btn.secondary:hover {
  border-color: var(--vp-c-text-3);
}

.install-line {
  margin-top: 8px;
}

.install-line code {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 0.85rem;
  color: var(--vp-c-text-3);
  background: var(--vp-c-bg-soft);
  padding: 6px 16px;
  border-radius: 4px;
  border: 1px solid var(--vp-c-border);
  user-select: all;
}

/* ── Compare ── */
.compare {
  padding: 60px 24px;
  max-width: 900px;
  margin: 0 auto;
}

.compare-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

@media (max-width: 640px) {
  .compare-grid {
    grid-template-columns: 1fr;
  }
}

.compare-card {
  border: 1px solid var(--vp-c-border);
  border-radius: 8px;
  overflow: hidden;
}

.compare-label {
  padding: 12px 20px;
  font-weight: 600;
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  border-bottom: 1px solid var(--vp-c-border);
}

.compare-card.bad .compare-label {
  color: var(--vp-c-text-2);
}

.compare-card.good .compare-label {
  color: var(--vp-c-text-1);
}

.compare-terminal {
  padding: 16px 20px;
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 0.8rem;
  line-height: 1.7;
  background: var(--vp-c-bg-soft);
}

.compare-terminal .dim {
  color: var(--vp-c-text-3);
}

.compare-terminal .accent-bad {
  color: var(--vp-c-text-2);
  font-weight: 600;
}

.compare-terminal .accent-good {
  color: var(--vp-c-text-1);
  font-weight: 600;
}

/* ── Stats ── */
.stats {
  display: flex;
  justify-content: center;
  gap: 48px;
  padding: 60px 24px;
  flex-wrap: wrap;
}

.stat {
  text-align: center;
}

.stat-number {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 2.8rem;
  font-weight: 700;
  line-height: 1;
  color: var(--vp-c-text-1);
  margin-bottom: 8px;
}

.stat-label {
  font-size: 0.85rem;
  color: var(--vp-c-text-3);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

/* ── Terminal demo ── */
.demo {
  padding: 40px 24px 60px;
  max-width: 720px;
  margin: 0 auto;
}

.demo h2 {
  text-align: center;
  font-size: 1.3rem;
  font-weight: 600;
  margin-bottom: 24px;
}

.terminal {
  border: 1px solid var(--vp-c-border);
  border-radius: 8px;
  overflow: hidden;
}

.terminal-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--vp-c-border);
  background: var(--vp-c-bg-alt);
}

.terminal-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--vp-c-border);
}

.terminal-title {
  margin-left: 8px;
  font-size: 0.75rem;
  color: var(--vp-c-text-3);
  font-family: 'SF Mono', 'Fira Code', monospace;
}

.terminal-body {
  padding: 20px;
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 0.85rem;
  line-height: 1.7;
  background: var(--vp-c-bg-soft);
  min-height: 260px;
}

.terminal-body .term-line {
  opacity: 0;
  transform: translateY(4px);
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.terminal-body .term-line.visible {
  opacity: 1;
  transform: translateY(0);
}

.prompt {
  color: var(--vp-c-text-3);
  margin-right: 8px;
  user-select: none;
}

.command {
  color: var(--vp-c-text-1);
  font-weight: 500;
}

.output {
  color: var(--vp-c-text-2);
}

.terminal-body :deep(.ok) {
  color: var(--vp-c-text-1);
  font-weight: 600;
}

.cursor {
  display: inline-block;
  color: var(--vp-c-text-3);
  margin-top: 2px;
}

.cursor.blink {
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  50% { opacity: 0; }
}

/* ── Steps ── */
.steps {
  padding: 60px 24px;
  max-width: 720px;
  margin: 0 auto;
}

.steps h2 {
  text-align: center;
  font-size: 1.3rem;
  font-weight: 600;
  margin-bottom: 32px;
}

.steps-grid {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.step {
  display: flex;
  align-items: flex-start;
  gap: 20px;
  padding: 20px 24px;
  border: 1px solid var(--vp-c-border);
  border-radius: 8px;
  transition: border-color 0.2s;
}

.step:hover {
  border-color: var(--vp-c-text-3);
}

.step-num {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 1.4rem;
  font-weight: 700;
  color: var(--vp-c-text-3);
  min-width: 28px;
  line-height: 1.3;
}

.step-title {
  font-weight: 600;
  margin-bottom: 4px;
  font-size: 0.95rem;
}

.step-content code {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 0.82rem;
  color: var(--vp-c-text-2);
}

/* ── Features ── */
.features {
  padding: 60px 24px;
  max-width: 900px;
  margin: 0 auto;
}

.feature-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 24px;
}

@media (max-width: 640px) {
  .feature-grid {
    grid-template-columns: 1fr;
  }
}

.feature {
  padding: 24px;
  border: 1px solid var(--vp-c-border);
  border-radius: 8px;
  transition: border-color 0.2s;
}

.feature:hover {
  border-color: var(--vp-c-text-3);
}

.feature h3 {
  font-size: 0.95rem;
  font-weight: 600;
  margin: 0 0 8px;
}

.feature p {
  font-size: 0.88rem;
  color: var(--vp-c-text-2);
  line-height: 1.6;
  margin: 0;
}

.feature code {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 0.8rem;
  background: var(--vp-c-bg-soft);
  padding: 1px 5px;
  border-radius: 3px;
}

/* ── CTA ── */
.cta {
  text-align: center;
  padding: 80px 24px;
  border-top: 1px solid var(--vp-c-border);
  margin-top: 40px;
}

.cta h2 {
  font-size: 1.5rem;
  font-weight: 600;
  margin: 0 0 8px;
}

.cta p {
  color: var(--vp-c-text-2);
  font-size: 1.1rem;
  margin: 0 0 28px;
}
</style>
