---
hide:
  - toc
---

# Template preview

Write a [notification template](notifications.md#templates) and see what it renders for generated containers and
log entries. The preview runs Watchtower's own template engine in your browser, so nothing is sent anywhere.

<style>
  .tplprev { display: grid; gap: 1rem; margin-top: 1.2rem; }
  .tplprev__panes { display: grid; gap: 1rem; grid-template-columns: minmax(0, 1fr); }
  @media screen and (min-width: 70em) {
    .tplprev__panes { grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); }
  }
  .tplprev__pane { display: flex; flex-direction: column; min-width: 0; }
  .tplprev__pane-header {
    align-items: center; color: var(--wt-muted); display: flex; font-size: 0.64rem; font-weight: 650;
    justify-content: space-between; letter-spacing: 0.06em; margin-bottom: 0.4rem; min-height: 1.5rem;
    text-transform: uppercase;
  }
  .tplprev textarea, .md-typeset .tplprev__output {
    background: var(--wt-surface); border: 1px solid var(--wt-border); border-radius: var(--wt-radius);
    box-sizing: border-box; color: var(--md-code-fg-color); font-family: var(--md-code-font-family);
    flex: 1; font-size: 0.68rem; line-height: 1.6; margin: 0; min-height: 26rem; padding: 0.9rem 1rem; width: 100%;
  }
  .tplprev textarea { overflow: auto; resize: vertical; tab-size: 2; white-space: pre; }
  .tplprev textarea:focus-visible { border-color: var(--wt-accent); outline: 2px solid var(--wt-accent-soft); }
  .tplprev__output { overflow: auto; white-space: pre-wrap; word-break: break-word; }
  .tplprev__output[data-state="empty"], .tplprev__output[data-state="loading"] { color: var(--wt-muted); font-style: italic; }
  .tplprev__output[data-state="error"] { color: #d14343; }
  .tplprev__controls {
    background: var(--wt-surface); border: 1px solid var(--wt-border); border-radius: var(--wt-radius);
    display: grid; gap: 1rem 2rem; grid-template-columns: repeat(auto-fit, minmax(16rem, 1fr)); padding: 0.9rem 1rem;
  }
  .tplprev fieldset { border: 0; margin: 0; min-width: 0; padding: 0; }
  .tplprev legend { align-items: center; display: flex; font-size: 0.7rem; font-weight: 650; gap: 0.4rem; margin-bottom: 0.5rem; padding: 0; }
  .tplprev legend input { accent-color: var(--wt-accent); margin: 0; }
  .tplprev__fields { display: grid; gap: 0.5rem; grid-template-columns: repeat(auto-fill, minmax(4.6rem, 1fr)); }
  .tplprev fieldset:disabled .tplprev__fields { opacity: 0.45; }
  .tplprev__field { color: var(--wt-muted); display: flex; flex-direction: column; font-size: 0.6rem; gap: 0.2rem; }
  .tplprev__field input {
    background: var(--wt-bg); border: 1px solid var(--wt-border); border-radius: var(--wt-radius-sm);
    color: var(--wt-text); font: inherit; font-size: 0.72rem; padding: 0.3rem 0.45rem; width: 100%; box-sizing: border-box;
  }
  .tplprev__field input:focus-visible { border-color: var(--wt-accent); outline: 2px solid var(--wt-accent-soft); }
  .tplprev__actions { display: flex; gap: 0.4rem; }
  .tplprev__actions button {
    background: transparent; border: 1px solid var(--wt-border); border-radius: var(--wt-radius-sm); color: var(--wt-muted);
    cursor: pointer; font: inherit; font-size: 0.6rem; letter-spacing: 0; padding: 0.15rem 0.55rem; text-transform: none;
  }
  .tplprev__actions button:hover { border-color: var(--wt-muted); color: var(--wt-text); }
</style>

<form class="tplprev" id="tplprev" autocomplete="off">
  <div class="tplprev__panes">
    <div class="tplprev__pane">
      <div class="tplprev__pane-header">
        <label for="tplprev-template">Template</label>
        <span class="tplprev__actions">
          <button type="button" id="tplprev-share" title="Copy a link to this template">Copy link</button>
          <button type="button" id="tplprev-reset" title="Restore the default template">Reset</button>
        </span>
      </div>
      <textarea id="tplprev-template" name="template" spellcheck="false"></textarea>
    </div>
    <div class="tplprev__pane">
      <div class="tplprev__pane-header"><span id="tplprev-result-label">Rendered notification</span></div>
      <pre class="tplprev__output" id="tplprev-result" aria-labelledby="tplprev-result-label" aria-live="polite" data-state="loading">Loading the template engine…</pre>
    </div>
  </div>

  <div class="tplprev__controls">
    <fieldset id="tplprev-report">
      <legend><input type="checkbox" name="report" id="tplprev-report-toggle" checked><label for="tplprev-report-toggle">Container report</label></legend>
      <div class="tplprev__fields">
        <label class="tplprev__field">Scanned<input type="number" min="0" max="50" name="scanned" value="3"></label>
        <label class="tplprev__field">Updated<input type="number" min="0" max="50" name="updated" value="3"></label>
        <label class="tplprev__field">Failed<input type="number" min="0" max="50" name="failed" value="1"></label>
        <label class="tplprev__field">Skipped<input type="number" min="0" max="50" name="skipped" value="1"></label>
        <label class="tplprev__field">Fresh<input type="number" min="0" max="50" name="fresh" value="3"></label>
        <label class="tplprev__field">Stale<input type="number" min="0" max="50" name="stale" value="0"></label>
      </div>
    </fieldset>
    <fieldset id="tplprev-log">
      <legend><input type="checkbox" name="log" id="tplprev-log-toggle" checked><label for="tplprev-log-toggle">Log entries</label></legend>
      <div class="tplprev__fields">
        <label class="tplprev__field">Error<input type="number" min="0" max="50" name="error" value="1"></label>
        <label class="tplprev__field">Warning<input type="number" min="0" max="50" name="warning" value="1"></label>
        <label class="tplprev__field">Info<input type="number" min="0" max="50" name="info" value="2"></label>
        <label class="tplprev__field">Debug<input type="number" min="0" max="50" name="debug" value="0"></label>
      </div>
    </fieldset>
  </div>
</form>

<script src="../assets/wasm_exec.js"></script>
<script>
(() => {
  const defaultTemplate = `{{- with .Report -}}
  {{- if ( or .Updated .Failed ) -}}
{{len .Scanned}} Scanned, {{len .Updated}} Updated, {{len .Failed}} Failed
    {{- range .Updated}}
- {{.Name}} ({{.ImageName}}): {{.CurrentImageID.ShortID}} updated to {{.LatestImageID.ShortID}}
    {{- end -}}
    {{- range .Fresh}}
- {{.Name}} ({{.ImageName}}): {{.State}}
    {{- end -}}
    {{- range .Skipped}}
- {{.Name}} ({{.ImageName}}): {{.State}}: {{.Error}}
    {{- end -}}
    {{- range .Failed}}
- {{.Name}} ({{.ImageName}}): {{.State}}: {{.Error}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- if (and .Entries .Report) }}

Logs:
{{ end -}}
{{range .Entries -}}{{.Time.Format "2006-01-02T15:04:05Z07:00"}} [{{.Level}}] {{.Message}}{{"\\n"}}{{- end -}}`;

  const form = document.getElementById("tplprev");
  const result = document.getElementById("tplprev-result");
  const reportStates = ["skipped", "scanned", "updated", "failed", "fresh", "stale"];
  const logLevels = ["error", "warning", "info", "debug"];
  let engineReady = false;
  let debounce;

  const repeat = (name) => Array.from({ length: Math.min(50, Math.max(0, form.elements[name].valueAsNumber || 0)) }, () => name);

  const show = (text, state) => {
    result.textContent = text;
    result.dataset.state = state;
  };

  const render = () => {
    if (!engineReady) return;
    document.getElementById("tplprev-report").disabled = !form.elements.report.checked;
    document.getElementById("tplprev-log").disabled = !form.elements.log.checked;

    const states = form.elements.report.checked ? reportStates.flatMap(repeat) : [];
    const levels = form.elements.log.checked ? logLevels.flatMap(repeat) : [];
    const output = WATCHTOWER.tplprev(form.elements.template.value, states, levels);

    if (output.startsWith("Error: ")) {
      show(output.substring(7), "error");
    } else if (output.length) {
      show(output, "ok");
    } else {
      show("The template rendered an empty message, so no notification would be sent.", "empty");
    }
  };

  const scheduleRender = () => {
    clearTimeout(debounce);
    debounce = setTimeout(render, 250);
  };

  const loadFromQuery = () => {
    const params = new URLSearchParams(location.search);
    form.elements.template.value = params.get("template") ?? defaultTemplate;
    for (const [key, value] of params) {
      const field = form.elements[key];
      if (!field || key === "template") continue;
      if (field.type === "checkbox") {
        field.checked = value === "yes";
      } else {
        field.value = value;
      }
    }
  };

  const shareLink = async (event) => {
    const params = new URLSearchParams();
    for (const field of form.elements) {
      if (!field.name) continue;
      params.set(field.name, field.type === "checkbox" ? (field.checked ? "yes" : "no") : field.value);
    }
    const url = `${location.origin}${location.pathname}?${params}`;
    history.replaceState(null, "", url);
    try {
      await navigator.clipboard.writeText(url);
      event.target.textContent = "Copied";
    } catch {
      event.target.textContent = "Link in address bar";
    }
    setTimeout(() => (event.target.textContent = "Copy link"), 2000);
  };

  loadFromQuery();
  form.addEventListener("input", scheduleRender);
  form.addEventListener("change", render);
  form.addEventListener("submit", (event) => event.preventDefault());
  document.getElementById("tplprev-share").addEventListener("click", shareLink);
  document.getElementById("tplprev-reset").addEventListener("click", () => {
    form.elements.template.value = defaultTemplate;
    render();
  });

  const go = new Go();
  WebAssembly.instantiateStreaming(fetch("../assets/tplprev.wasm"), go.importObject)
    .then(({ instance }) => {
      go.run(instance);
      engineReady = true;
      render();
    })
    .catch((err) => show(`The template engine could not be loaded: ${err}`, "error"));
})();
</script>
