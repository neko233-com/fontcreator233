(function () {
  "use strict";

  var data = window.FontCreatorData || { family: "", cps: [], coverage: {} };
  var all = (data.cps || []).map(function (cp) {
    return { cp: cp, script: scriptOf(cp) };
  });
  var filtered = all.slice();
  var scroll = document.getElementById("glyph-scroll");
  var spacer = document.getElementById("glyph-spacer");
  var layer = document.getElementById("glyph-layer");
  var search = document.getElementById("glyph-search");
  var scriptSelect = document.getElementById("glyph-script");
  var countLabel = document.getElementById("glyph-count");
  var toastTimer = null;
  var virtualizer = null;
  var cols = 8;
  var rowHeight = 84;

  function scriptOf(cp) {
    if ((cp >= 0x3040 && cp <= 0x30ff) || (cp >= 0x31f0 && cp <= 0x31ff) || (cp >= 0xff66 && cp <= 0xff9f)) return "kana";
    if ((cp >= 0x1100 && cp <= 0x11ff) || (cp >= 0x3130 && cp <= 0x318f) || (cp >= 0xa960 && cp <= 0xa97f) || (cp >= 0xac00 && cp <= 0xd7af) || (cp >= 0xd7b0 && cp <= 0xd7ff) || (cp >= 0xffa0 && cp <= 0xffdc)) return "hangul";
    if ((cp >= 0x2e80 && cp <= 0x303f) || (cp >= 0x3400 && cp <= 0x4dbf) || (cp >= 0x4e00 && cp <= 0x9fff) || (cp >= 0xf900 && cp <= 0xfaff) || (cp >= 0x20000 && cp <= 0x2fa1f) || (cp >= 0x30000 && cp <= 0x323af)) return "cjk";
    if ((cp >= 0x0370 && cp <= 0x03ff) || (cp >= 0x1f00 && cp <= 0x1fff)) return "greek";
    if (cp >= 0x0400 && cp <= 0x052f) return "cyrillic";
    if ((cp >= 0x0020 && cp <= 0x024f) || (cp >= 0x1e00 && cp <= 0x1eff) || cp === 0x0131) return "latin";
    if ((cp >= 0x2000 && cp <= 0x27bf) || (cp >= 0xfe30 && cp <= 0xfe4f) || (cp >= 0xff00 && cp <= 0xffef)) return "symbols";
    return "other";
  }

  function hex(cp) {
    return "U+" + cp.toString(16).toUpperCase().padStart(4, "0");
  }

  function toast(message) {
    var el = document.getElementById("toast");
    el.textContent = message;
    el.classList.add("show");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { el.classList.remove("show"); }, 1600);
  }

  function copy(text) {
    if (navigator.clipboard && window.isSecureContext) {
      navigator.clipboard.writeText(text).then(function () { toast("Copied " + text); }, function () { fallbackCopy(text); });
    } else {
      fallbackCopy(text);
    }
  }

  function fallbackCopy(text) {
    var ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand("copy"); toast("Copied " + text); } catch (e) { toast("Copy failed"); }
    document.body.removeChild(ta);
  }

  function computeCols() {
    var width = scroll.clientWidth || 800;
    cols = Math.max(4, Math.min(16, Math.floor(width / 84)));
    layer.style.setProperty("--cols", String(cols));
  }

  function makeRow(index) {
    var row = document.createElement("div");
    row.className = "glyph-row";
    row.style.position = "absolute";
    row.style.left = "0";
    row.style.right = "0";
    row.style.top = (index * rowHeight) + "px";
    row.style.height = rowHeight + "px";
    var start = index * cols;
    for (var i = 0; i < cols; i++) {
      var glyph = filtered[start + i];
      var cell = document.createElement("div");
      cell.className = "glyph-cell";
      if (!glyph) {
        row.appendChild(cell);
        continue;
      }
      var ch = document.createElement("span");
      ch.className = "ch";
      ch.textContent = String.fromCodePoint(glyph.cp);
      var cp = document.createElement("span");
      cp.className = "cp";
      cp.textContent = hex(glyph.cp);
      cell.appendChild(ch);
      cell.appendChild(cp);
      cell.title = "Copy " + ch.textContent + " " + hex(glyph.cp);
      cell.addEventListener("click", function (value) {
        return function () { copy(value); };
      }(ch.textContent));
      row.appendChild(cell);
    }
    return row;
  }

  function fallbackRender() {
    if (!spacer || !layer) return;
    var totalRows = Math.ceil(filtered.length / cols);
    spacer.style.height = (totalRows * rowHeight) + "px";
    var top = scroll.scrollTop;
    var first = Math.max(0, Math.floor(top / rowHeight) - 2);
    var last = Math.min(totalRows - 1, Math.ceil((top + scroll.clientHeight) / rowHeight) + 2);
    layer.textContent = "";
    for (var i = first; i <= last; i++) layer.appendChild(makeRow(i));
  }

  function rebuildVirtualizer() {
    if (virtualizer && virtualizer.cleanup) virtualizer.cleanup();
    virtualizer = null;
    computeCols();
    var rows = Math.ceil(filtered.length / cols);
    countLabel.textContent = filtered.length.toLocaleString() + " glyphs · " + cols + " columns";
    if (!window.TanStackVirtual || !window.TanStackVirtual.Virtualizer) {
      fallbackRender();
      return;
    }
    virtualizer = new window.TanStackVirtual.Virtualizer({
      count: rows,
      getScrollElement: function () { return scroll; },
      estimateSize: function () { return rowHeight; },
      overscan: 2,
      observeElementRect: window.TanStackVirtual.observeElementRect,
      observeElementOffset: window.TanStackVirtual.observeElementOffset,
      scrollToFn: window.TanStackVirtual.elementScroll
    });
    virtualizer._didMount();
    renderVirtual();
  }

  function renderVirtual() {
    if (!virtualizer) { fallbackRender(); return; }
    virtualizer._willUpdate();
    var items = virtualizer.getVirtualItems();
    spacer.style.height = virtualizer.getTotalSize() + "px";
    layer.textContent = "";
    for (var i = 0; i < items.length; i++) layer.appendChild(makeRow(items[i].index));
  }

  function applyFilter() {
    var q = (search.value || "").trim().toLowerCase();
    var script = scriptSelect.value;
    var target = null;
    if (/^u\+?[0-9a-f]{2,6}$/.test(q)) {
      target = parseInt(q.replace(/^u\+?/, ""), 16);
    }
    filtered = all.filter(function (g) {
      if (script !== "all" && g.script !== script) return false;
      if (!q) return true;
      var ch = String.fromCodePoint(g.cp).toLowerCase();
      return ch === q || (target !== null && g.cp === target) || hex(g.cp).toLowerCase().indexOf(q) >= 0;
    });
    scroll.scrollTop = 0;
    rebuildVirtualizer();
  }

  function setFontStatus() {
    var status = document.getElementById("font-status");
    var banner = document.getElementById("font-error");
    if (!document.fonts || !document.fonts.check) {
      status.textContent = "font API unavailable";
      return;
    }
    document.fonts.ready.then(function () {
      var ok = document.fonts.check('16px "Specimen"');
      status.textContent = ok ? "font loaded" : "font missing";
      status.className = "status-pill " + (ok ? "ok" : "bad");
      banner.hidden = ok;
    });
  }

  function wire() {
    if (document.body.dataset.accent) {
      document.documentElement.style.setProperty("--accent", document.body.dataset.accent);
    }
    setFontStatus();
    var hero = document.getElementById("hero");
    var size = document.getElementById("hero-size");
    var sizeValue = document.getElementById("hero-size-value");
    size.addEventListener("input", function () {
      hero.style.fontSize = size.value + "px";
      sizeValue.textContent = size.value + " px";
    });
    document.getElementById("copy-hero").addEventListener("click", function () { copy(hero.innerText); });
    document.getElementById("copy-css").addEventListener("click", function () { copy(document.getElementById("css-snippet").innerText); });
    document.getElementById("theme-toggle").addEventListener("click", function () {
      var root = document.documentElement;
      var dark = root.dataset.theme ? root.dataset.theme === "dark" : matchMedia("(prefers-color-scheme: dark)").matches;
      root.dataset.theme = dark ? "light" : "dark";
    });
    var resizeTimer = null;
    window.addEventListener("resize", function () {
      clearTimeout(resizeTimer);
      resizeTimer = setTimeout(rebuildVirtualizer, 120);
    });
    scroll.addEventListener("scroll", function () {
      window.requestAnimationFrame(function () {
        if (virtualizer) renderVirtual(); else fallbackRender();
      });
    }, { passive: true });
    search.addEventListener("input", applyFilter);
    scriptSelect.addEventListener("change", applyFilter);
    rebuildVirtualizer();
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", wire);
  } else {
    wire();
  }
}());
