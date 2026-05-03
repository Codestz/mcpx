// mcpx dashboard — vanilla JS, zero npm
(function () {
  const T = window.MCPX_TOKEN;
  const auth = (path) => path + (path.includes("?") ? "&" : "?") + "t=" + T;

  const state = {
    scope: { kind: "this" }, // {kind:"this"|"all"|"project", value?}
    since: "168h",
    sinceSeconds: 7 * 24 * 3600,
    summary: null,
    selectedRowEl: null,
    selectedRecord: null,
    sseAlive: false,
    lastEventAt: Date.now(),
    filterRegex: null,
  };

  // ---------- formatters ----------
  function fmtNumber(n) {
    if (n == null) return "—";
    const a = Math.abs(n);
    if (a >= 1e9) return (n / 1e9).toFixed(1) + "B";
    if (a >= 1e6) return (n / 1e6).toFixed(a >= 1e8 ? 0 : 1) + "M";
    if (a >= 1e3) return (n / 1e3).toFixed(a >= 1e5 ? 0 : 1) + "K";
    return String(n);
  }
  function fmtPct(p) { return p == null ? "—" : Math.round(p * 100) + "%"; }
  function fmtMs(ms) {
    if (ms == null) return "—";
    if (ms < 1000) return ms + "ms";
    if (ms < 60_000) return Math.round(ms / 1000) + "s";
    return Math.floor(ms / 60_000) + "m" + String(Math.round((ms % 60_000) / 1000)).padStart(2, "0") + "s";
  }
  function fmtTime(iso) {
    const d = new Date(iso);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit", hour12: false });
  }
  function fmtAgo(ts) {
    const s = Math.floor((Date.now() - ts) / 1000);
    if (s < 5) return "just now";
    if (s < 60) return s + "s ago";
    if (s < 3600) return Math.floor(s / 60) + "m ago";
    return Math.floor(s / 3600) + "h ago";
  }
  function shortName(server, tool) { return server + "." + tool; }

  // ---------- API ----------
  async function fetchSummary() {
    const params = new URLSearchParams({ since: state.since });
    if (state.scope.kind === "project") params.set("project", state.scope.value);
    if (state.scope.kind === "all") params.set("all", "1");
    const r = await fetch(auth("/api/summary?" + params.toString()));
    if (!r.ok) throw new Error("summary fetch failed");
    return r.json();
  }
  async function fetchProjects() {
    const r = await fetch(auth("/api/projects"));
    if (!r.ok) return [];
    return r.json();
  }
  async function fetchRecord(id) {
    if (state.recentById?.[id]) return state.recentById[id];
    const r = await fetch(auth("/api/call/" + encodeURIComponent(id)));
    if (!r.ok) return null;
    return r.json();
  }
  async function fetchToolDetail(server, tool) {
    const r = await fetch(auth("/api/tool/" + encodeURIComponent(server) + "/" + encodeURIComponent(tool)));
    if (!r.ok) return null;
    return r.json();
  }
  async function fetchAudit() {
    const r = await fetch(auth("/api/audit"));
    if (!r.ok) return [];
    return r.json();
  }

  // ---------- render: hero + tiles ----------
  function render(s) {
    state.summary = s;

    // hero
    document.getElementById("hero-saved").textContent = fmtNumber(s.TokensSaved);
    document.getElementById("hero-foot").textContent = "vs native MCP loading · " + windowLabel();

    const eff = s.NativeBaseline > 0 ? s.TokensSaved / s.NativeBaseline : 0;
    const effPct = Math.max(0, Math.min(1, eff));
    document.getElementById("eff-pct").textContent = fmtPct(effPct);
    document.getElementById("eff-fill").style.width = (effPct * 100).toFixed(1) + "%";
    document.getElementById("eff-saved").textContent = fmtNumber(s.TokensSaved);
    document.getElementById("eff-native").textContent = fmtNumber(s.NativeBaseline);

    // tiles
    setTile("t-calls", fmtNumber(s.Calls));
    setTile("t-cache", fmtPct(s.CacheHitRate));
    document.getElementById("t-cache-foot").textContent =
      "schema " + fmtPct(s.SchemaHitRate) + " · result " + fmtPct(s.ResultHitRate);
    setTile("t-errors", fmtPct(s.ErrorRate), s.ErrorRate > 0.05 ? "bad" : (s.ErrorRate > 0 ? "warn" : ""));
    setTile("t-avg", fmtMs(Math.round(s.AvgLatencyMS)));
    document.getElementById("t-avg-foot").textContent = "p95 " + fmtMs(s.P95LatencyMS);

    renderTopTools(s);
    renderTopSavers(s);
    renderServerHealth(s);
    renderRecentList(s.Recent || []);
    renderServerSidebar(s);

    document.getElementById("scope-window").textContent = windowLabel();
  }
  function setTile(id, text, cls) {
    const el = document.getElementById(id);
    el.textContent = text;
    el.classList.remove("warn", "bad");
    if (cls) el.classList.add(cls);
  }
  function windowLabel() {
    if (state.since === "all") return "all time";
    return ({
      "1h": "last hour", "24h": "last 24h", "168h": "last 7 days", "720h": "last 30 days",
    }[state.since]) || state.since;
  }

  function renderTopTools(s) {
    const tbl = document.getElementById("tool-table");
    tbl.innerHTML = "";
    const items = (s.TopTools || []).slice(0, 6);
    if (!items.length) { tbl.innerHTML = '<tr><td class="dim small">no calls in window</td></tr>'; return; }
    const max = items[0].Calls;
    items.forEach((t) => {
      const tr = document.createElement("tr");
      tr.style.cursor = "pointer";
      tr.title = "click to drill into " + shortName(t.Server, t.Tool);
      tr.innerHTML =
        '<td class="name">' + escapeHtml(shortName(t.Server, t.Tool)) + "</td>" +
        '<td class="bar"><div class="fill" style="width:' + (t.Calls / max * 100) + '%"></div></td>' +
        '<td class="value">' + t.Calls + "</td>";
      tr.onclick = () => openToolDetail(t.Server, t.Tool);
      tbl.appendChild(tr);
    });
  }

  async function openToolDetail(server, tool) {
    const detail = await fetchToolDetail(server, tool);
    if (!detail) return;
    document.getElementById("drawer-title").textContent = shortName(server, tool);
    document.getElementById("drawer-sub").textContent = detail.calls + " calls in window";
    const kv = document.getElementById("drawer-kv");
    kv.innerHTML = "";
    const cell = (k, v) => { kv.innerHTML += '<div class="k">' + k + '</div><div class="v">' + v + '</div>'; };
    cell("calls", detail.calls);
    cell("errors", detail.errors);
    cell("avg latency", fmtMs(Math.round(detail.avg_latency_ms)));
    cell("tokens saved", fmtNumber(detail.tokens_saved));

    const recent = (detail.recent || []).slice(-15).reverse();
    document.getElementById("drawer-args").textContent = "(tool detail view — recent calls listed below)";
    document.getElementById("drawer-result").textContent = recent
      .map((r) => fmtTime(r.ts) + "  " + fmtMs(r.latency_ms).padStart(6) +
        (r.exit_code ? "  [error: " + r.error + "]" : ""))
      .join("\n");
    document.getElementById("drawer").classList.remove("hidden");
  }
  function renderTopSavers(s) {
    const tbl = document.getElementById("save-table");
    tbl.innerHTML = "";
    const items = (s.TopSavers || []).filter((t) => t.TokensSaved > 0).slice(0, 6);
    if (!items.length) { tbl.innerHTML = '<tr><td class="dim small">no savings yet</td></tr>'; return; }
    const max = items[0].TokensSaved;
    items.forEach((t) => {
      const tr = document.createElement("tr");
      tr.innerHTML =
        '<td class="name">' + shortName(t.Server, t.Tool) + "</td>" +
        '<td class="bar"><div class="fill" style="width:' + (t.TokensSaved / max * 100) + '%"></div></td>' +
        '<td class="value">' + fmtNumber(t.TokensSaved) + "</td>";
      tbl.appendChild(tr);
    });
  }
  function renderServerHealth(s) {
    const el = document.getElementById("server-health");
    el.innerHTML = "";
    const servers = s.TopServers || [];
    if (!servers.length) {
      el.innerHTML = '<div class="dim small">no servers active in window</div>';
      return;
    }
    servers.forEach((sv) => {
      const errRate = sv.Calls > 0 ? sv.Errors / sv.Calls : 0;
      const statusCls = errRate > 0.1 ? "bad" : (errRate > 0 ? "warn" : "");
      const statusText = errRate > 0.1 ? "degraded" : (errRate > 0 ? "warn" : "ok");
      const card = document.createElement("div");
      card.className = "health-card";
      card.innerHTML =
        '<div class="top"><span class="name">' + sv.Server + "</span>" +
        '<span class="status ' + statusCls + '">' + statusText + "</span></div>" +
        '<div class="stats">' +
          '<span><b>' + sv.Calls + "</b> calls</span>" +
          '<span>p95 <b>' + fmtMs(sv.P95LatencyMS || 0) + "</b></span>" +
          '<span><b>' + fmtPct(errRate) + "</b> err</span>" +
        "</div>";
      el.appendChild(card);
    });
  }

  function renderProjects(projects) {
    const ul = document.getElementById("project-list");
    ul.innerHTML = "";
    const liAll = document.createElement("li");
    liAll.innerHTML = "<span>all projects</span>";
    if (state.scope.kind === "all") liAll.classList.add("active");
    liAll.onclick = () => { state.scope = { kind: "all" }; bootstrap(); };
    ul.appendChild(liAll);
    projects.forEach((p) => {
      const li = document.createElement("li");
      const name = p.Name || "(no name)";
      li.innerHTML = "<span>" + escapeHtml(name) + "</span><span class='count'>" + p.Calls + "</span>";
      const isCurrent = (state.scope.kind === "this" && p.Current) ||
                        (state.scope.kind === "project" && state.scope.value === p.Path);
      if (isCurrent) li.classList.add("active");
      li.onclick = () => { state.scope = { kind: "project", value: p.Path }; bootstrap(); };
      ul.appendChild(li);
    });
  }
  function renderServerSidebar(s) {
    const ul = document.getElementById("server-list");
    ul.innerHTML = "";
    (s.Servers || []).forEach((name) => {
      const li = document.createElement("li");
      const calls = (s.PerServer && s.PerServer[name]) || 0;
      li.innerHTML = "<span>" + escapeHtml(name) + "</span><span class='count'>" + calls + "</span>";
      ul.appendChild(li);
    });
  }

  // ---------- live tail ----------
  function renderRecentList(rows) {
    state.recentById = {};
    const el = document.getElementById("live-tail");
    el.innerHTML = "";
    if (!rows.length) {
      el.innerHTML = '<div class="tail-empty">no calls yet — run any mcpx command to start filling this view</div>';
      return;
    }
    // newest first
    [...rows].reverse().forEach((r, i) => prependTail(el, r, false, i === 0));
  }
  function prependTail(el, r, animate, isNewest) {
    if (!r) return;
    if (state.filterRegex && !state.filterRegex.test(shortName(r.server, r.tool))) {
      return;
    }
    const id = r.ts + "|" + r.server + "|" + r.tool + "|" + (r.session || "");
    state.recentById[id] = r;

    const row = document.createElement("div");
    row.className = "tail-row";
    row.dataset.id = id;
    let badge = "";
    if (r.result_cache_hit) badge = '<span class="badge cached">cached</span>';
    else if (r.schema_cache_hit) badge = '<span class="badge warm">warm</span>';
    row.innerHTML =
      '<span class="ts">' + fmtTime(r.ts) + "</span>" +
      '<span class="tool">' + escapeHtml(shortName(r.server, r.tool)) + "</span>" +
      '<span class="lat">' + fmtMs(r.latency_ms) + "</span>" +
      "<span>" + badge + "</span>" +
      '<span class="' + (r.exit_code ? "err" : "ok") + '">' + (r.exit_code ? "✗" : "✓") + "</span>";
    row.onclick = () => openDrawer(row, r);
    el.prepend(row);
    while (el.children.length > 80) el.removeChild(el.lastChild);

    if (animate) {
      row.style.opacity = 0;
      requestAnimationFrame(() => {
        row.style.transition = "opacity 0.25s, background 0.25s";
        row.style.opacity = 1;
      });
    }
  }

  // ---------- drawer ----------
  async function openDrawer(rowEl, recordOrId) {
    if (state.selectedRowEl) state.selectedRowEl.classList.remove("selected");
    if (rowEl) {
      rowEl.classList.add("selected");
      state.selectedRowEl = rowEl;
    }

    const r = typeof recordOrId === "string" ? await fetchRecord(recordOrId) : recordOrId;
    if (!r) { closeDrawer(); return; }
    state.selectedRecord = r;

    document.getElementById("drawer-title").textContent = shortName(r.server, r.tool);
    document.getElementById("drawer-sub").textContent =
      new Date(r.ts).toLocaleString() + (r.id ? "  ·  " + r.id : "");

    const kv = document.getElementById("drawer-kv");
    kv.innerHTML = "";
    const cell = (k, v) => { kv.innerHTML += '<div class="k">' + k + '</div><div class="v">' + v + '</div>'; };
    cell("latency", fmtMs(r.latency_ms));
    cell("exit", String(r.exit_code || 0));
    cell("project", r.project ? '<code style="font-size:11px">' + escapeHtml(r.project) + '</code>' : '—');
    cell("session", r.session || "—");
    cell("agent", r.agent || "—");
    cell("transport", (r.transport || "—") + (r.daemon ? " (daemon)" : ""));
    cell("schema cache", r.schema_cache_hit ? "hit" : "miss");
    cell("result cache", r.result_cache_hit ? "hit" : "miss");
    cell("baseline tokens", fmtNumber(r.native_baseline_tokens || 0));
    cell("tokens saved", fmtNumber(r.tokens_saved || 0));
    if (r.policy_action) cell("policy", r.policy_action + (r.policy_name ? " (" + r.policy_name + ")" : ""));
    if (r.error) cell("error", '<span style="color:var(--red)">' + escapeHtml(r.error) + '</span>');

    document.getElementById("drawer-args").textContent =
      r.args ? JSON.stringify(r.args, null, 2) : "(no args recorded)";

    let resultDisplay = "(no result preview captured)";
    if (r.result_preview) {
      resultDisplay = r.result_preview;
      if (r.result_truncated) resultDisplay += "\n\n… (truncated)";
    }
    document.getElementById("drawer-result").textContent = resultDisplay;

    document.getElementById("drawer").classList.remove("hidden");
  }
  function closeDrawer() {
    document.getElementById("drawer").classList.add("hidden");
    if (state.selectedRowEl) state.selectedRowEl.classList.remove("selected");
    state.selectedRowEl = null;
  }

  // ---------- SSE ----------
  function startSSE() {
    const ev = new EventSource(auth("/api/events"));
    ev.onopen = () => { setConn(true); };
    ev.addEventListener("call", (e) => {
      try {
        const r = JSON.parse(e.data);
        state.lastEventAt = Date.now();
        prependTail(document.getElementById("live-tail"), r, true, true);
        // refresh aggregates lazily
        fetchSummary().then(render).catch(() => {});
      } catch {}
    });
    ev.onerror = () => {
      setConn(false);
      ev.close();
      setTimeout(startSSE, 3000);
    };
  }
  function setConn(alive) {
    state.sseAlive = alive;
    const dot = document.getElementById("conn-dot");
    dot.classList.remove("live", "dead");
    dot.classList.add(alive ? "live" : "dead");
  }

  // ---------- bootstrap ----------
  async function bootstrap() {
    try {
      const [s, projects] = await Promise.all([fetchSummary(), fetchProjects()]);
      render(s);
      renderProjects(projects);
      const label =
        state.scope.kind === "all" ? "All projects" :
        state.scope.kind === "project" ? short(state.scope.value) :
        "This project";
      document.getElementById("scope-label").textContent = label;
      tickFreshness();
    } catch (e) { console.error(e); }
  }
  function short(p) {
    if (!p) return "—";
    if (p.length <= 36) return p;
    return "…" + p.slice(-32);
  }
  function escapeHtml(s) {
    return String(s).replace(/[&<>"']/g, (c) => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]));
  }
  function tickFreshness() {
    const el = document.getElementById("freshness");
    el.textContent = fmtAgo(state.lastEventAt);
    setTimeout(tickFreshness, 1000);
  }

  // ---------- wire interactions ----------
  document.getElementById("time-tabs").addEventListener("click", (e) => {
    const btn = e.target.closest("button[data-since]");
    if (!btn) return;
    document.querySelectorAll("#time-tabs button").forEach((b) => b.classList.remove("active"));
    btn.classList.add("active");
    state.since = btn.dataset.since;
    bootstrap();
  });
  document.getElementById("drawer-close").onclick = closeDrawer;
  document.getElementById("drawer-copy").onclick = () => {
    if (!state.selectedRecord) return;
    navigator.clipboard.writeText(JSON.stringify(state.selectedRecord, null, 2));
  };
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") closeDrawer();
    if (e.key === "/" && document.activeElement.tagName !== "INPUT") {
      e.preventDefault();
      document.getElementById("tail-filter").focus();
    }
  });
  document.getElementById("tail-filter").addEventListener("input", (e) => {
    const v = e.target.value.trim();
    try {
      state.filterRegex = v ? new RegExp(v, "i") : null;
    } catch { state.filterRegex = null; }
    if (state.summary) renderRecentList(state.summary.Recent || []);
  });

  bootstrap().then(startSSE).catch(console.error);
})();
