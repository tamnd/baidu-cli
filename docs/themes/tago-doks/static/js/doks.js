/* tago-doks runtime: theme toggle, mobile nav, TOC scroll-spy, copy buttons, search. */
(function () {
  "use strict";

  var DOKS = window.DOKS || {};

  /* ---- Theme toggle ------------------------------------------------- */
  function initTheme() {
    var btn = document.getElementById("doks-theme-btn");
    if (!btn) return;
    btn.addEventListener("click", function () {
      var root = document.documentElement;
      var next = root.getAttribute("data-theme") === "dark" ? "light" : "dark";
      root.setAttribute("data-theme", next);
      try { localStorage.setItem("doks-theme", next); } catch (e) {}
    });
  }

  /* ---- Mobile sidebar ----------------------------------------------- */
  function initSidebar() {
    var burger = document.getElementById("doks-burger");
    var sidebar = document.getElementById("doks-sidebar");
    var backdrop = document.getElementById("doks-backdrop");
    if (!burger || !sidebar) return;

    function close() {
      sidebar.classList.remove("is-open");
      if (backdrop) backdrop.classList.remove("is-open");
      burger.setAttribute("aria-expanded", "false");
    }
    function toggle() {
      var open = sidebar.classList.toggle("is-open");
      if (backdrop) backdrop.classList.toggle("is-open", open);
      burger.setAttribute("aria-expanded", open ? "true" : "false");
    }
    burger.addEventListener("click", toggle);
    if (backdrop) backdrop.addEventListener("click", close);
    sidebar.addEventListener("click", function (e) {
      if (e.target.closest("a")) close();
    });
    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape") close();
    });
  }

  /* ---- Table of contents + scroll-spy ------------------------------- */
  function slugify(text) {
    return text.toLowerCase().trim()
      .replace(/[^\w\s-]/g, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-");
  }

  function initToc() {
    var prose = document.getElementById("doks-prose");
    var toc = document.getElementById("doks-toc");
    var nav = document.getElementById("doks-toc-nav");
    if (!prose || !toc || !nav) return;

    var heads = prose.querySelectorAll("h2, h3");
    if (!heads.length) return;

    var seen = {};
    var links = [];
    heads.forEach(function (h) {
      if (!h.id) {
        var base = slugify(h.textContent) || "section";
        var id = base, n = 1;
        while (seen[id] || document.getElementById(id)) { id = base + "-" + n++; }
        h.id = id;
      }
      seen[h.id] = true;

      var a = document.createElement("a");
      a.href = "#" + h.id;
      a.textContent = h.textContent;
      a.className = h.tagName === "H3" ? "lvl-3" : "lvl-2";
      nav.appendChild(a);
      links.push(a);

      var anchor = document.createElement("a");
      anchor.href = "#" + h.id;
      anchor.className = "doks-anchor";
      anchor.setAttribute("aria-label", "Link to this section");
      anchor.textContent = "#";
      h.appendChild(anchor);
    });

    toc.hidden = false;

    var byId = {};
    links.forEach(function (a) { byId[a.getAttribute("href").slice(1)] = a; });
    var current = null;
    function setActive(id) {
      if (current === id) return;
      current = id;
      links.forEach(function (a) { a.classList.remove("is-active"); });
      if (byId[id]) byId[id].classList.add("is-active");
    }

    if ("IntersectionObserver" in window) {
      var visible = {};
      var obs = new IntersectionObserver(function (entries) {
        entries.forEach(function (en) {
          visible[en.target.id] = en.isIntersecting;
        });
        var ids = Array.prototype.map.call(heads, function (h) { return h.id; });
        for (var i = 0; i < ids.length; i++) {
          if (visible[ids[i]]) { setActive(ids[i]); return; }
        }
      }, { rootMargin: "-80px 0px -70% 0px", threshold: 0 });
      heads.forEach(function (h) { obs.observe(h); });
    }
  }

  /* ---- Copy buttons on code blocks ---------------------------------- */
  function initCopy() {
    var prose = document.getElementById("doks-prose");
    if (!prose) return;
    var blocks = prose.querySelectorAll("pre");
    blocks.forEach(function (pre) {
      var wrap = document.createElement("div");
      wrap.className = "doks-codeblock";
      pre.parentNode.insertBefore(wrap, pre);
      wrap.appendChild(pre);

      var btn = document.createElement("button");
      btn.type = "button";
      btn.className = "doks-copy";
      btn.textContent = "Copy";
      wrap.appendChild(btn);

      btn.addEventListener("click", function () {
        var code = pre.querySelector("code");
        var text = code ? code.innerText : pre.innerText;
        navigator.clipboard.writeText(text).then(function () {
          btn.textContent = "Copied";
          btn.classList.add("is-copied");
          setTimeout(function () {
            btn.textContent = "Copy";
            btn.classList.remove("is-copied");
          }, 1600);
        });
      });
    });
  }

  /* ---- Search (FlexSearch over the tago search-data index) ---------- */
  function initSearch() {
    var hasInputs = document.getElementById("doks-search-input") ||
      document.getElementById("doks-searchpage-input");
    if (!hasInputs) return;

    // The search index keys are root-relative permalinks. On a sub-path
    // deploy (e.g. /dbrest/) prepend the baseURL path so links resolve.
    var prefix = "";
    try {
      var p = new URL(DOKS.baseURL || "/", location.href).pathname.replace(/\/$/, "");
      if (p && p !== "") prefix = p;
    } catch (e) {}
    function resolveURL(u) {
      if (/^https?:/.test(u)) return u;
      if (prefix && u.indexOf(prefix + "/") !== 0) return prefix + u;
      return u;
    }

    var index = null, docs = [], loading = null;

    function load() {
      if (loading) return loading;
      loading = fetch(DOKS.searchURL)
        .then(function (r) { return r.json(); })
        .then(function (data) {
          index = new FlexSearch.Document({
            tokenize: "forward",
            document: { id: "id", index: ["title", "body"], store: ["title", "body", "url", "crumb"] }
          });
          var id = 0;
          Object.keys(data).forEach(function (url) {
            var entry = data[url];
            var body = "";
            if (entry.data && typeof entry.data === "object") {
              body = Object.keys(entry.data).map(function (k) { return entry.data[k]; }).join(" ");
            }
            var doc = { id: id++, title: entry.title || url, body: body, url: url, crumb: entry.crumb || "" };
            docs.push(doc);
            index.add(doc);
          });
        })
        .catch(function () { index = null; loading = null; });
      return loading;
    }

    function query(q) {
      if (!index) return [];
      var res = index.search(q, { limit: 12, enrich: true });
      var seen = {}, out = [];
      res.forEach(function (field) {
        field.result.forEach(function (hit) {
          var d = hit.doc;
          if (!d || seen[d.url]) return;
          seen[d.url] = true;
          out.push(d);
        });
      });
      return out.slice(0, 10);
    }

    function snippet(body, q) {
      if (!body) return "";
      var lc = body.toLowerCase();
      var pos = lc.indexOf(q.toLowerCase());
      var start = pos > 40 ? pos - 40 : 0;
      var text = body.slice(start, start + 140);
      if (start > 0) text = "..." + text;
      return text;
    }

    function escapeHtml(s) {
      return s.replace(/[&<>"]/g, function (c) {
        return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c];
      });
    }

    function renderInto(container, results, q, asDropdown) {
      container.innerHTML = "";
      if (!q) { if (asDropdown) container.hidden = true; return; }
      if (!results.length) {
        container.innerHTML = '<div class="doks-result__empty">No results for &ldquo;' + escapeHtml(q) + '&rdquo;</div>';
        if (asDropdown) container.hidden = false;
        return;
      }
      results.forEach(function (d) {
        var a = document.createElement("a");
        a.className = "doks-result";
        a.href = resolveURL(d.url);
        var html = '<span class="doks-result__title">' + escapeHtml(d.title) + "</span>";
        if (d.crumb) html += '<span class="doks-result__crumb">' + escapeHtml(d.crumb) + "</span>";
        var sn = snippet(d.body, q);
        if (sn) html += '<span class="doks-result__snippet">' + escapeHtml(sn) + "</span>";
        a.innerHTML = html;
        container.appendChild(a);
      });
      if (asDropdown) container.hidden = false;
    }

    function wire(input, results, asDropdown) {
      if (!input || !results) return;
      var run = function () {
        var q = input.value.trim();
        if (!q) { renderInto(results, [], "", asDropdown); return; }
        load().then(function () { renderInto(results, query(q), q, asDropdown); });
      };
      input.addEventListener("input", run);
      input.addEventListener("focus", function () { if (input.value.trim()) load(); });
      if (asDropdown) {
        document.addEventListener("click", function (e) {
          if (!e.target.closest("#doks-search")) results.hidden = true;
        });
        input.addEventListener("keydown", function (e) {
          if (e.key === "Escape") { results.hidden = true; input.blur(); }
        });
      }
    }

    wire(document.getElementById("doks-search-input"), document.getElementById("doks-search-results"), true);
    wire(document.getElementById("doks-searchpage-input"), document.getElementById("doks-searchpage-results"), false);
  }

  /* ---- Boot --------------------------------------------------------- */
  function boot() {
    initTheme();
    initSidebar();
    initToc();
    initCopy();
    initSearch();
  }
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", boot);
  } else {
    boot();
  }
})();
