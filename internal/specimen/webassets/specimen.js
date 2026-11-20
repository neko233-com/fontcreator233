(function () {
  "use strict";

  var data = window.FontCreatorData;
  var hasGlyphData = !!(data && data.cps);
  if (!hasGlyphData) data = { family: "", cps: [], coverage: {} };
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
  var rowHeight = 88;

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
      navigator.clipboard.writeText(text).then(function () { toast("Copied"); }, function () { fallbackCopy(text); });
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
    try { document.execCommand("copy"); toast("Copied"); } catch (e) { toast("Copy failed"); }
    document.body.removeChild(ta);
  }

  function computeCols() {
    var width = scroll ? scroll.clientWidth : 800;
    cols = Math.max(4, Math.min(16, Math.floor(width / 88)));
    if (layer) layer.style.setProperty("--cols", String(cols));
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
    if (!scroll || !spacer || !layer) return;
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
    if (!scroll) return;
    computeCols();
    var rows = Math.ceil(filtered.length / cols);
    if (countLabel) countLabel.textContent = filtered.length.toLocaleString() + " glyphs · " + cols + " columns";
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
    if (spacer) spacer.style.height = virtualizer.getTotalSize() + "px";
    if (!layer) return;
    layer.textContent = "";
    for (var i = 0; i < items.length; i++) layer.appendChild(makeRow(items[i].index));
  }

  function applyFilter() {
    if (!search || !scriptSelect) return;
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
    if (scroll) scroll.scrollTop = 0;
    rebuildVirtualizer();
  }

  function showAssetError() {
    var el = document.getElementById("asset-error");
    if (el) el.hidden = false;
    var status = document.getElementById("font-status");
    if (status) {
      status.className = "status bad";
      status.textContent = "assets missing";
    }
  }

  function setFontStatus() {
    var status = document.getElementById("font-status");
    var banner = document.getElementById("font-error");
    if (!status) return;
    if (!document.fonts || !document.fonts.check) {
      status.textContent = "font API unavailable";
      return;
    }
    document.fonts.ready.then(function () {
      var ok = document.fonts.check('16px "Specimen"');
      status.className = "status " + (ok ? "ok" : "bad");
      status.textContent = ok ? (hasGlyphData ? "font loaded" : "assets missing") : "font missing";
      if (banner) banner.hidden = ok;
      if (!ok) {
        var asset = document.getElementById("asset-error");
        if (asset) asset.hidden = true;
      }
    }, function () {
      status.className = "status bad";
      status.textContent = "font error";
      if (banner) banner.hidden = false;
    });
  }

  function controlValue(id, format) {
    var el = document.getElementById(id);
    var out = document.getElementById(id + "-value");
    if (el && out) out.textContent = format(el.value);
  }

  function applyTester() {
    var text = document.getElementById("tester-text");
    var size = document.getElementById("tester-size");
    var leading = document.getElementById("tester-leading");
    var tracking = document.getElementById("tester-tracking");
    if (!text || !size || !leading || !tracking) return;
    text.style.fontSize = size.value + "px";
    text.style.lineHeight = (Number(leading.value) / 100).toFixed(2);
    text.style.letterSpacing = (Number(tracking.value) / 100) + "em";
    controlValue("tester-size", function (v) { return v; });
    controlValue("tester-leading", function (v) { return (Number(v) / 100).toFixed(2); });
    controlValue("tester-tracking", function (v) { return (Number(v) / 100).toFixed(2) + "em"; });
  }

  function applyFeatures() {
    var demo = document.getElementById("feature-demo");
    var code = document.getElementById("feature-code");
    var ids = ["liga", "kern", "calt", "curs", "frac", "case", "tnum", "ss01"];
    var settings = ids.map(function (id) {
      var el = document.getElementById("feature-" + id);
      return '"' + id + '" ' + (el && el.checked ? 1 : 0);
    }).join(", ");
    demo.style.fontFeatureSettings = settings;
    if (code) code.textContent = "font-feature-settings: " + settings + ";";
  }

  function applyVariable() {
    var axis = document.getElementById("var-axis");
    var sample = document.getElementById("var-sample");
    var value = document.getElementById("var-value");
    if (!axis || !sample) return;
    sample.style.fontVariationSettings = '"wght" ' + axis.value;
    if (value) value.textContent = axis.value;
  }

  function wire() {
    if (document.body.dataset.accent) {
      document.documentElement.style.setProperty("--accent", document.body.dataset.accent);
    }
    var styleLink = document.querySelector('link[rel="stylesheet"]');
    if (!hasGlyphData || (styleLink && !styleLink.sheet)) {
      showAssetError();
    } else {
      var asset = document.getElementById("asset-error");
      if (asset) asset.hidden = true;
    }
    setFontStatus();
    var text = document.getElementById("tester-text");
    var size = document.getElementById("tester-size");
    var leading = document.getElementById("tester-leading");
    var tracking = document.getElementById("tester-tracking");
    var reset = document.getElementById("tester-reset");
    if (size) size.addEventListener("input", applyTester);
    if (leading) leading.addEventListener("input", applyTester);
    if (tracking) tracking.addEventListener("input", applyTester);
    if (reset) reset.addEventListener("click", function () {
      if (size) size.value = "112";
      if (leading) leading.value = "118";
      if (tracking) tracking.value = "0";
      applyTester();
    });
    applyTester();
    ["copy-sample"].forEach(function (id) {
      var btn = document.getElementById(id);
      if (btn && text) btn.addEventListener("click", function () { copy(text.innerText); });
    });
    ["copy-css", "copy-css-2", "copy-css-3"].forEach(function (id) {
      var btn = document.getElementById(id);
      var snippet = document.getElementById("css-snippet");
      if (btn && snippet) btn.addEventListener("click", function () { copy(snippet.innerText); });
    });
    var theme = document.getElementById("theme-toggle");
    if (theme) theme.addEventListener("click", function () {
      var root = document.documentElement;
      var dark = root.dataset.theme ? root.dataset.theme === "dark" : matchMedia("(prefers-color-scheme: dark)").matches;
      root.dataset.theme = dark ? "light" : "dark";
    });
    if (scroll) {
      var resizeTimer = null;
      window.addEventListener("resize", function () {
        clearTimeout(resizeTimer);
        resizeTimer = setTimeout(rebuildVirtualizer, 140);
      });
      scroll.addEventListener("scroll", function () {
        window.requestAnimationFrame(function () {
          if (virtualizer) renderVirtual(); else fallbackRender();
        });
      }, { passive: true });
    }
    if (search) search.addEventListener("input", applyFilter);
    if (scriptSelect) scriptSelect.addEventListener("change", applyFilter);
    ["feature-liga", "feature-kern", "feature-calt", "feature-curs", "feature-frac", "feature-case", "feature-tnum", "feature-ss01"].forEach(function (id) {
      var el = document.getElementById(id);
      if (el) el.addEventListener("change", applyFeatures);
    });
    applyFeatures();
    var paletteButtons = document.querySelectorAll(".palette-btn");
    if (paletteButtons.length) {
      var activePalette = document.querySelector(".palette-btn[data-palette='0']");
      if (activePalette) activePalette.classList.add("active");
      paletteButtons.forEach(function (btn) {
        btn.addEventListener("click", function () {
          var name = "--fcp" + btn.getAttribute("data-palette");
          document.querySelectorAll(".color-sample").forEach(function (el) {
            el.style.fontPalette = name;
          });
          paletteButtons.forEach(function (b) { b.classList.remove("active"); });
          btn.classList.add("active");
        });
      });
    }
    var axis = document.getElementById("var-axis");
    if (axis) {
      var match = (window.location.hash || "").match(/wght=(\d+)/);
      if (match) axis.value = match[1];
      axis.addEventListener("input", applyVariable);
    }
    applyVariable();
    rebuildVirtualizer();
    window.__fontcreatorBoot = true;
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", wire);
  } else {
    wire();
  }
}());
