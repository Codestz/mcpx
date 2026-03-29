<script setup>
import { ref, onMounted } from 'vue'
import { withBase } from 'vitepress'

// Terminal demo lines
const lines = [
  { prompt: true, text: 'mcpx list', delay: 0 },
  { prompt: false, text: '  serena     serena (daemon)', delay: 500 },
  { prompt: false, text: '  postgres   postgres-mcp', delay: 600 },
  { prompt: false, text: '', delay: 700 },
  { prompt: true, text: 'mcpx serena find_symbol --name "Auth"', delay: 1100 },
  { prompt: false, text: '  [{"name": "AuthService", "kind": "class", "file": "src/auth.go"}]', delay: 1900 },
  { prompt: false, text: '', delay: 2000 },
  { prompt: true, text: 'mcpx postgres query --sql "DROP TABLE users"', delay: 2500 },
  { prompt: false, text: '  <span class="denied">error:</span> policy "no-mutations" denied tool "query"', delay: 3200 },
  { prompt: false, text: '  <span class="dim">Reason: Mutation queries blocked</span>', delay: 3400 },
]

// Security demo lines
const secLines = [
  { prompt: true, text: 'mcpx serena replace_symbol_body \\', delay: 0 },
  { prompt: false, text: '    --relative_path "../../../etc/passwd"', delay: 200 },
  { prompt: false, text: '', delay: 400 },
  { prompt: false, text: '  <span class="denied">error:</span> policy "no-path-traversal" denied', delay: 900 },
  { prompt: false, text: '  <span class="dim">Reason: Path traversal blocked</span>', delay: 1100 },
  { prompt: false, text: '  <span class="dim">relative_path = "../../../etc/passwd"</span>', delay: 1200 },
  { prompt: false, text: '', delay: 1400 },
  { prompt: true, text: 'cat .mcpx/audit.jsonl | jq .action', delay: 1800 },
  { prompt: false, text: '  "allowed"', delay: 2400 },
  { prompt: false, text: '  "allowed"', delay: 2500 },
  { prompt: false, text: '  <span class="denied">"denied"</span>', delay: 2600 },
]

const visibleLines = ref(0)
const secVisibleLines = ref(0)
const secAnimStarted = ref(false)

onMounted(() => {
  lines.forEach((line, i) => {
    setTimeout(() => { visibleLines.value = i + 1 }, line.delay)
  })

  // Intersection observer for security terminal
  const observer = new IntersectionObserver((entries) => {
    entries.forEach(e => {
      if (e.isIntersecting && !secAnimStarted.value) {
        secAnimStarted.value = true
        secLines.forEach((line, i) => {
          setTimeout(() => { secVisibleLines.value = i + 1 }, line.delay)
        })
      }
    })
  }, { threshold: 0.3 })

  setTimeout(() => {
    const el = document.querySelector('.sec-right')
    if (el) observer.observe(el)
  }, 100)
})
</script>

<template>
  <div class="landing">

    <!-- ══════════════ HERO — full width, two-column ══════════════ -->
    <section class="hero">
      <div class="hero-inner">
        <div class="hero-text">
          <div class="hero-badge">Secure MCP Gateway</div>
          <h1>
            The control plane<br>
            between <span class="ul">AI agents</span><br>
            and <span class="ul">MCP servers</span>.
          </h1>
          <p class="hero-sub">
            Security policies. Audit logging. Lifecycle hooks.<br>
            Workspace routing. One binary. Zero tokens upfront.
          </p>
          <div class="hero-actions">
            <a :href="withBase('/getting-started/installation')" class="btn primary">Get Started</a>
            <a :href="withBase('/security/overview')" class="btn secondary">Security Docs</a>
            <a href="https://github.com/codestz/mcpx" class="btn ghost" target="_blank">GitHub</a>
          </div>
          <div class="hero-install">
            <code><span class="dim">$</span> brew install codestz/tap/mcpx</code>
          </div>
        </div>
        <div class="hero-terminal">
          <div class="terminal">
            <div class="term-chrome">
              <span class="dot"></span><span class="dot"></span><span class="dot"></span>
              <span class="term-title">terminal</span>
            </div>
            <div class="term-body">
              <div
                v-for="(line, i) in lines" :key="i"
                class="tl" :class="{ vis: i < visibleLines }"
              >
                <span v-if="line.prompt" class="pr">$</span>
                <span v-if="line.prompt" class="cmd">{{ line.text }}</span>
                <span v-else class="out" v-html="line.text"></span>
              </div>
              <div class="cursor" :class="{ blink: visibleLines >= lines.length }">_</div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ══════════════ STATS — full-width bar ══════════════ -->
    <section class="stats-bar">
      <div class="stats-inner">
        <div class="stat">
          <div class="stat-n">0</div>
          <div class="stat-l">tokens upfront</div>
        </div>
        <div class="stat-divider"></div>
        <div class="stat">
          <div class="stat-n">&lt;5ms</div>
          <div class="stat-l">startup time</div>
        </div>
        <div class="stat-divider"></div>
        <div class="stat">
          <div class="stat-n">3</div>
          <div class="stat-l">transports</div>
        </div>
        <div class="stat-divider"></div>
        <div class="stat">
          <div class="stat-n">&infin;</div>
          <div class="stat-l">servers, same cost</div>
        </div>
      </div>
    </section>

    <!-- ══════════════ PROBLEM — comparison ══════════════ -->
    <section class="problem">
      <div class="wide">
        <div class="sec-label">The Problem</div>
        <h2>MCP servers are expensive and unrestricted.</h2>
        <p class="sec-sub">Every server dumps its schema into context. Every tool call has zero access control. mcpx fixes both.</p>

        <div class="cmp-grid">
          <div class="cmp-card bad">
            <div class="cmp-head">Without mcpx</div>
            <div class="cmp-body">
              <div class="tl dim"># 5 servers loaded at session start</div>
              <div class="tl">"serena": { ... }<span class="dim">         # ~20K tokens</span></div>
              <div class="tl">"postgres": { ... }<span class="dim">       # ~15K tokens</span></div>
              <div class="tl">"jira": { ... }<span class="dim">           # ~12K tokens</span></div>
              <div class="tl">"slack": { ... }<span class="dim">          # ~8K tokens</span></div>
              <div class="tl">"github": { ... }<span class="dim">         # ~30K tokens</span></div>
              <div class="tl">&nbsp;</div>
              <div class="tl accent-bad">~85K tokens gone. No security. No audit.</div>
            </div>
          </div>
          <div class="cmp-card good">
            <div class="cmp-head">With mcpx</div>
            <div class="cmp-body">
              <div class="tl dim"># .mcpx/config.yml — one file</div>
              <div class="tl">5 servers configured</div>
              <div class="tl">Security policies per server</div>
              <div class="tl">Audit log for every call</div>
              <div class="tl">Lifecycle hooks on connect</div>
              <div class="tl">Workspace auto-detection</div>
              <div class="tl">&nbsp;</div>
              <div class="tl accent-good">0 tokens. Full security. Full audit.</div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ══════════════ SECURITY — side by side, full width ══════════════ -->
    <section class="security">
      <div class="wide">
        <div class="sec-label">Security</div>
        <h2>Every call. Every server. Enforced.</h2>
        <p class="sec-sub">Policies evaluate tool names, arguments, and content before the call reaches the server.</p>

        <div class="sec-grid">
          <div class="sec-left">
            <div class="cfg-chrome">
              <span class="cfg-title">.mcpx/config.yml</span>
            </div>
            <div class="cfg-body">
              <pre><code><span class="y">security</span>:
  <span class="y">enabled</span>: <span class="g">true</span>
  <span class="y">global</span>:
    <span class="y">audit</span>:
      <span class="y">enabled</span>: <span class="g">true</span>
      <span class="y">log</span>: .mcpx/audit.jsonl
    <span class="y">policies</span>:
      - <span class="y">name</span>: no-path-traversal
        <span class="y">match</span>:
          <span class="y">args</span>:
            <span class="y">"*path*"</span>:
              <span class="y">deny_pattern</span>: "\\.\\.\\/|\\.\\.\\\\\\/"
        <span class="y">action</span>: <span class="r">deny</span>
        <span class="y">message</span>: Path traversal blocked

<span class="y">servers</span>:
  <span class="y">postgres</span>:
    <span class="y">security</span>:
      <span class="y">mode</span>: <span class="r">read-only</span></code></pre>
            </div>
          </div>
          <div class="sec-right">
            <div class="terminal">
              <div class="term-chrome">
                <span class="dot"></span><span class="dot"></span><span class="dot"></span>
                <span class="term-title">result</span>
              </div>
              <div class="term-body">
                <div
                  v-for="(line, i) in secLines" :key="'s'+i"
                  class="tl" :class="{ vis: i < secVisibleLines }"
                >
                  <span v-if="line.prompt" class="pr">$</span>
                  <span v-if="line.prompt" class="cmd">{{ line.text }}</span>
                  <span v-else class="out" v-html="line.text"></span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- ══════════════ STEPS — horizontal 3-col ══════════════ -->
    <section class="steps">
      <div class="wide">
        <h2>Three commands to start</h2>
        <div class="steps-grid">
          <div class="step">
            <div class="step-n">1</div>
            <div class="step-t">Install</div>
            <code>brew install codestz/tap/mcpx</code>
          </div>
          <div class="step">
            <div class="step-n">2</div>
            <div class="step-t">Configure</div>
            <code>mcpx init</code>
          </div>
          <div class="step">
            <div class="step-n">3</div>
            <div class="step-t">Call</div>
            <code>mcpx serena find_symbol --name "Auth"</code>
          </div>
        </div>
      </div>
    </section>

    <!-- ══════════════ FEATURES — 4-column grid ══════════════ -->
    <section class="features">
      <div class="wide">
        <div class="feat-grid">
          <a :href="withBase('/security/policies')" class="feat link">
            <h3>Security Policies</h3>
            <p>Tool allow/deny, argument inspection, content regex. Global + per-server cascading rules.</p>
          </a>
          <a :href="withBase('/security/audit-logging')" class="feat link">
            <h3>Audit Logging</h3>
            <p>Every tool call recorded in JSONL. Timestamps, args, policy decisions. Secret redaction built in.</p>
          </a>
          <a :href="withBase('/workspaces/overview')" class="feat link">
            <h3>Workspaces</h3>
            <p>Monorepo auto-detection. Per-workspace lifecycle hooks and security profiles. One config file.</p>
          </a>
          <a :href="withBase('/integrations/serena')" class="feat link">
            <h3>Serena Integration</h3>
            <p>Lifecycle hooks for project activation. Workspace routing for monorepos. Path-restricted editing.</p>
          </a>
          <div class="feat">
            <h3>Daemon Mode</h3>
            <p>Heavy servers stay warm via unix socket. Zero spawn cost after first invocation.</p>
          </div>
          <div class="feat">
            <h3>On-Demand Discovery</h3>
            <p>Tools discovered lazily with <code>mcpx list</code> and <code>--help</code>. Zero context overhead.</p>
          </div>
          <div class="feat">
            <h3>Three Transports</h3>
            <p>stdio for local, HTTP (streamable) and SSE for remote. Auth headers and bearer tokens built in.</p>
          </div>
          <div class="feat">
            <h3>Single Binary</h3>
            <p>Go. No runtime deps. Sub-millisecond startup. Homebrew, <code>go install</code>, or build from source.</p>
          </div>
        </div>
      </div>
    </section>

    <!-- ══════════════ CTA — full-width banner ══════════════ -->
    <section class="cta">
      <div class="cta-inner">
        <h2>The missing control plane for MCP.</h2>
        <p>Security, audit, lifecycle — from CLI to production.</p>
        <div class="cta-actions">
          <a :href="withBase('/getting-started/installation')" class="btn primary">Get Started</a>
          <a :href="withBase('/about/why-mcpx')" class="btn secondary">Why mcpx</a>
          <a :href="withBase('/workspaces/serena-monorepo')" class="btn ghost">Monorepo Walkthrough</a>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
/* ── Foundation ── */
.landing { font-family: var(--vp-font-family-base); color: var(--vp-c-text-1); }
.wide { max-width: 1200px; margin: 0 auto; padding: 0 32px; }
.mono { font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace; }

.sec-label {
  text-align: center; font-size: 0.72rem; font-weight: 600;
  text-transform: uppercase; letter-spacing: 0.12em; color: var(--vp-c-text-3); margin-bottom: 10px;
}
h2 { text-align: center; font-size: 1.6rem; font-weight: 700; margin: 0 0 10px; letter-spacing: -0.02em; }
.sec-sub { text-align: center; font-size: 0.95rem; color: var(--vp-c-text-2); max-width: 580px; margin: 0 auto 40px; line-height: 1.6; }

/* ── Buttons ── */
.btn { display: inline-block; padding: 10px 22px; border-radius: 6px; font-size: 0.9rem; font-weight: 500; text-decoration: none; transition: all 0.15s ease; }
.btn.primary { background: var(--vp-c-text-1); color: var(--vp-c-bg); }
.btn.primary:hover { opacity: 0.85; }
.btn.secondary { border: 1px solid var(--vp-c-border); color: var(--vp-c-text-1); }
.btn.secondary:hover { border-color: var(--vp-c-text-3); }
.btn.ghost { color: var(--vp-c-text-2); }
.btn.ghost:hover { color: var(--vp-c-text-1); }

/* ── Terminal shared ── */
.terminal { border: 1px solid var(--vp-c-border); border-radius: 8px; overflow: hidden; }
.term-chrome { display: flex; align-items: center; gap: 6px; padding: 10px 16px; border-bottom: 1px solid var(--vp-c-border); background: var(--vp-c-bg-alt); }
.dot { width: 10px; height: 10px; border-radius: 50%; background: var(--vp-c-border); }
.term-title { margin-left: 8px; font-size: 0.72rem; color: var(--vp-c-text-3); font-family: 'SF Mono', 'Fira Code', monospace; }
.term-body { padding: 20px; font-family: 'SF Mono', 'Fira Code', monospace; font-size: 0.8rem; line-height: 1.75; background: var(--vp-c-bg-soft); }
.tl { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.term-body .tl { opacity: 0; transform: translateY(4px); transition: opacity 0.3s, transform 0.3s; }
.term-body .tl.vis { opacity: 1; transform: translateY(0); }
.pr { color: var(--vp-c-text-3); margin-right: 8px; user-select: none; }
.cmd { color: var(--vp-c-text-1); font-weight: 500; }
.out { color: var(--vp-c-text-2); }
.term-body :deep(.denied) { color: #e06c75; font-weight: 600; }
.term-body :deep(.dim) { color: var(--vp-c-text-3); }
.cursor { display: inline-block; color: var(--vp-c-text-3); margin-top: 2px; }
.cursor.blink { animation: blink 1s step-end infinite; }
@keyframes blink { 50% { opacity: 0; } }

/* ════════════════════════════════════════
   HERO — two-column, full width
   ════════════════════════════════════════ */
.hero { padding: 80px 32px 40px; }
.hero-inner {
  max-width: 1200px; margin: 0 auto;
  display: grid; grid-template-columns: 1fr 1fr; gap: 48px; align-items: center;
}
.hero-badge {
  display: inline-block; font-size: 0.7rem; font-weight: 600; text-transform: uppercase;
  letter-spacing: 0.1em; padding: 4px 12px; border: 1px solid var(--vp-c-border);
  border-radius: 20px; color: var(--vp-c-text-2); margin-bottom: 20px;
}
.hero-text h1 {
  font-size: clamp(1.7rem, 3.2vw, 2.4rem); font-weight: 700; line-height: 1.2;
  letter-spacing: -0.03em; margin: 0 0 16px;
}
.ul { text-decoration: underline; text-decoration-color: var(--vp-c-border); text-underline-offset: 4px; text-decoration-thickness: 2px; }
.hero-sub { font-size: 1rem; color: var(--vp-c-text-2); line-height: 1.7; margin: 0 0 28px; }
.hero-actions { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 20px; }
.hero-install code {
  font-family: 'SF Mono', 'Fira Code', monospace; font-size: 0.82rem; color: var(--vp-c-text-3);
  background: var(--vp-c-bg-soft); padding: 6px 14px; border-radius: 4px; border: 1px solid var(--vp-c-border); user-select: all;
}
.hero-install code .dim { color: var(--vp-c-text-3); }
.hero-terminal .term-body { min-height: 280px; }

@media (max-width: 860px) {
  .hero-inner { grid-template-columns: 1fr; }
  .hero-text { text-align: center; }
  .hero-actions { justify-content: center; }
  .hero-install { text-align: center; }
}

/* ════════════════════════════════════════
   STATS BAR — full-width divider
   ════════════════════════════════════════ */
.stats-bar {
  border-top: 1px solid var(--vp-c-border); border-bottom: 1px solid var(--vp-c-border);
  background: var(--vp-c-bg-alt); padding: 40px 32px;
}
.stats-inner { max-width: 1200px; margin: 0 auto; display: flex; justify-content: center; align-items: center; gap: 0; flex-wrap: wrap; }
.stat { text-align: center; padding: 0 40px; }
.stat-divider { width: 1px; height: 48px; background: var(--vp-c-border); }
.stat-n { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 2.4rem; font-weight: 700; color: var(--vp-c-text-1); line-height: 1; margin-bottom: 6px; }
.stat-l { font-size: 0.78rem; color: var(--vp-c-text-3); text-transform: uppercase; letter-spacing: 0.06em; }

@media (max-width: 640px) {
  .stat { padding: 12px 24px; }
  .stat-divider { display: none; }
  .stats-inner { gap: 8px; }
}

/* ════════════════════════════════════════
   PROBLEM — comparison cards
   ════════════════════════════════════════ */
.problem { padding: 80px 0; }
.cmp-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
.cmp-card { border: 1px solid var(--vp-c-border); border-radius: 8px; overflow: hidden; }
.cmp-head { padding: 12px 20px; font-weight: 600; font-size: 0.8rem; text-transform: uppercase; letter-spacing: 0.08em; border-bottom: 1px solid var(--vp-c-border); color: var(--vp-c-text-2); }
.cmp-body { padding: 16px 20px; font-family: 'SF Mono', 'Fira Code', monospace; font-size: 0.78rem; line-height: 1.75; background: var(--vp-c-bg-soft); }
.cmp-body .dim { color: var(--vp-c-text-3); }
.cmp-body .accent-bad { color: var(--vp-c-text-2); font-weight: 600; }
.cmp-body .accent-good { color: var(--vp-c-text-1); font-weight: 600; }

@media (max-width: 640px) { .cmp-grid { grid-template-columns: 1fr; } }

/* ════════════════════════════════════════
   SECURITY — config left, terminal right
   ════════════════════════════════════════ */
.security { padding: 80px 0; }
.sec-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; align-items: start; }
.sec-left { border: 1px solid var(--vp-c-border); border-radius: 8px; overflow: hidden; }
.cfg-chrome { padding: 10px 16px; border-bottom: 1px solid var(--vp-c-border); background: var(--vp-c-bg-alt); }
.cfg-title { font-size: 0.72rem; color: var(--vp-c-text-3); font-family: 'SF Mono', 'Fira Code', monospace; }
.cfg-body { padding: 16px 20px; background: var(--vp-c-bg-soft); }
.cfg-body pre { margin: 0; font-family: 'SF Mono', 'Fira Code', monospace; font-size: 0.75rem; line-height: 1.7; }
.cfg-body code { color: var(--vp-c-text-2); }
.cfg-body .y { color: var(--vp-c-text-1); }
.cfg-body .g { color: #98c379; }
.cfg-body .r { color: #e06c75; }

@media (max-width: 860px) { .sec-grid { grid-template-columns: 1fr; } }

/* ════════════════════════════════════════
   STEPS — horizontal 3-col
   ════════════════════════════════════════ */
.steps { padding: 80px 0; }
.steps h2 { margin-bottom: 32px; }
.steps-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; }
.step {
  padding: 24px; border: 1px solid var(--vp-c-border); border-radius: 8px;
  text-align: center; transition: border-color 0.2s;
}
.step:hover { border-color: var(--vp-c-text-3); }
.step-n { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 1.6rem; font-weight: 700; color: var(--vp-c-text-3); margin-bottom: 8px; }
.step-t { font-weight: 600; font-size: 0.95rem; margin-bottom: 8px; }
.step code { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 0.78rem; color: var(--vp-c-text-2); word-break: break-all; }

@media (max-width: 640px) { .steps-grid { grid-template-columns: 1fr; } }

/* ════════════════════════════════════════
   FEATURES — 4-col grid (2 rows)
   ════════════════════════════════════════ */
.features { padding: 60px 0 80px; }
.feat-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; }
.feat {
  padding: 24px 20px; border: 1px solid var(--vp-c-border); border-radius: 8px;
  transition: border-color 0.2s; text-decoration: none; color: inherit; display: block;
}
.feat:hover { border-color: var(--vp-c-text-3); }
.feat.link { cursor: pointer; }
.feat h3 { font-size: 0.88rem; font-weight: 600; margin: 0 0 6px; }
.feat p { font-size: 0.82rem; color: var(--vp-c-text-2); line-height: 1.55; margin: 0; }
.feat code { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 0.75rem; background: var(--vp-c-bg-soft); padding: 1px 4px; border-radius: 3px; }

@media (max-width: 1024px) { .feat-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 640px) { .feat-grid { grid-template-columns: 1fr; } }

/* ════════════════════════════════════════
   CTA — full-width banner
   ════════════════════════════════════════ */
.cta {
  border-top: 1px solid var(--vp-c-border); background: var(--vp-c-bg-alt);
  padding: 80px 32px; text-align: center;
}
.cta-inner { max-width: 600px; margin: 0 auto; }
.cta h2 { font-size: 1.5rem; font-weight: 700; margin: 0 0 8px; letter-spacing: -0.02em; }
.cta p { color: var(--vp-c-text-2); font-size: 1.05rem; margin: 0 0 28px; }
.cta-actions { display: flex; gap: 10px; justify-content: center; flex-wrap: wrap; }
</style>
