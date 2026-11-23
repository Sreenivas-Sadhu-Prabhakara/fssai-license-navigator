/* FSSAI licence navigator — local user interface.
   Talks only to this binary on the same origin. No third-party code, no
   outbound requests, no eval. */
(function () {
  "use strict";

  var FLAGS = ["importer", "exporter", "eou", "ecommerce", "port"];

  /* The thresholds and fees the user may correct, in display order. Keys match
     the fields the binary sends and accepts. */
  var PARAM_FIELDS = [
    { key: "pettyFBOTurnoverCeilingINR", label: "Petty food business operator turnover ceiling (rupees a year)", step: "1000" },
    { key: "stateLicenceTurnoverCeilingINR", label: "Turnover above which a Central Licence is needed (rupees a year)", step: "100000" },
    { key: "pettyFBOCapacityKgPerDay", label: "Petty food business operator food capacity (kg a day)", step: "1" },
    { key: "stateCeilingFoodMTPerDay", label: "Food capacity a State Licence covers (tonnes a day)", step: "0.1" },
    { key: "dairyBasicCeilingLitresPerDay", label: "Petty food business operator milk ceiling (litres a day)", step: "10" },
    { key: "stateCeilingDairyLitresPerDay", label: "Milk a State Licence covers (litres a day)", step: "1000" },
    { key: "stateCeilingMilkSolidsMTPerYear", label: "Milk solids a State Licence covers (tonnes a year)", step: "100" },
    { key: "centralMinStatesOfOperation", label: "States of operation that trigger a Central Licence", step: "1", integer: true },
    { key: "feeBasicRegistrationPerYearINR", label: "Basic Registration fee (rupees a year)", step: "50", integer: true },
    { key: "feeStateFoodServicePerYearINR", label: "State Licence fee, food service (rupees a year)", step: "100", integer: true },
    { key: "feeStateOtherFBOPerYearINR", label: "State Licence fee, any other food business (rupees a year)", step: "100", integer: true },
    { key: "feeStateManufacturerSmallPerYearINR", label: "State Licence fee, small manufacturing unit (rupees a year)", step: "100", integer: true },
    { key: "feeStateManufacturerLargePerYearINR", label: "State Licence fee, large manufacturing unit (rupees a year)", step: "100", integer: true },
    { key: "feeCentralLicencePerYearINR", label: "Central Licence fee (rupees a year)", step: "100", integer: true },
    { key: "stateFeeManufacturerBandMTPerDay", label: "Capacity band separating the two manufacturing fees (tonnes a day)", step: "0.1" },
    { key: "stateFeeDairyBandLitresPerDay", label: "Milk band separating the two manufacturing fees (litres a day)", step: "500" },
    { key: "validityMinYears", label: "Shortest validity you may apply for (years)", step: "1", integer: true },
    { key: "validityMaxYears", label: "Longest validity you may apply for (years)", step: "1", integer: true },
    { key: "alertNoticeDays", label: "First renewal alert (days before expiry)", step: "1", integer: true },
    { key: "alertWarningDays", label: "Second renewal alert (days before expiry)", step: "1", integer: true },
    { key: "alertCriticalDays", label: "Final renewal alert (days before expiry)", step: "1", integer: true },
    { key: "lateRenewalPenaltyPerDayINR", label: "Indicative late renewal penalty (rupees a day)", step: "10", integer: true }
  ];

  var DEFAULT_ANSWERS = {
    turnover: "800000",
    activity: "restaurant",
    capacity: "0",
    unit: "kg/day",
    dairy: "0",
    solids: "0",
    states: "1"
  };

  var flags = {};
  var money = null;
  try {
    money = new Intl.NumberFormat("en-IN", {
      style: "currency", currency: "INR", maximumFractionDigits: 0
    });
  } catch (e) {
    money = null;
  }

  function el(id) { return document.getElementById(id); }

  function rupees(n) {
    if (typeof n !== "number" || !isFinite(n)) { return "₹0"; }
    if (money) { return money.format(Math.round(n)); }
    return "₹" + Math.round(n);
  }

  function clear(node) {
    while (node && node.firstChild) { node.removeChild(node.firstChild); }
  }

  function make(tag, text, cls) {
    var n = document.createElement(tag);
    if (text !== undefined && text !== null) { n.textContent = String(text); }
    if (cls) { n.className = cls; }
    return n;
  }

  function number(id) {
    var raw = el(id) ? el(id).value : "";
    var v = parseFloat(raw);
    if (!isFinite(v) || v < 0) { return 0; }
    return v;
  }

  function whole(id) {
    var v = Math.round(number(id));
    return v < 1 ? 1 : v;
  }

  /* ----------------------------------------------------------- theme toggle */

  function themeNow() {
    var set = document.documentElement.getAttribute("data-theme");
    if (set) { return set; }
    if (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches) {
      return "dark";
    }
    return "light";
  }

  function paintThemer(button) {
    var dark = themeNow() === "dark";
    button.textContent = dark ? "☀ Light" : "☾ Dark";
    button.setAttribute("aria-label", dark ? "Switch to light theme" : "Switch to dark theme");
  }

  function wireTheme() {
    var button = document.querySelector("[data-themer]");
    if (!button) { return; }
    paintThemer(button);
    button.addEventListener("click", function () {
      var next = themeNow() === "dark" ? "light" : "dark";
      document.documentElement.setAttribute("data-theme", next);
      try { localStorage.setItem("omsme-theme", next); } catch (e) { /* ignore */ }
      paintThemer(button);
    });
  }

  /* ------------------------------------------------------------ the request */

  function answers() {
    var q = new URLSearchParams();
    q.set("turnover", String(number("turnover")));
    q.set("activity", el("activity").value || "manufacturer");
    q.set("capacity", String(number("capacity")));
    q.set("unit", el("unit").value || "kg/day");
    q.set("dairy", String(number("dairy")));
    q.set("solids", String(number("solids")));
    q.set("states", String(whole("states")));
    FLAGS.forEach(function (f) { q.set(f, flags[f] ? "1" : "0"); });
    return q.toString();
  }

  function readJSON(response) {
    return response.json().then(function (body) {
      if (!response.ok) {
        throw new Error(body && body.error ? body.error : "request failed");
      }
      return body;
    });
  }

  /* ----------------------------------------------------------- the decision */

  function renderDecision(d) {
    el("tier-name").textContent = d.tierName;
    el("tier-sub").textContent = d.form + " · issued by the " + d.authority;

    var rows = el("fee-rows");
    clear(rows);
    (d.feeByYears || []).forEach(function (opt) {
      var tr = document.createElement("tr");
      tr.appendChild(make("td", opt.years === 1 ? "1 year" : opt.years + " years"));
      tr.appendChild(make("td", rupees(opt.totalINR), "num"));
      rows.appendChild(tr);
    });
    el("fee-basis").textContent = d.feeBasis +
      " Validity is your choice at application, from " + d.validityMinYears +
      " to " + d.validityMaxYears + " years. Indicative only.";

    var reasons = el("reasons");
    clear(reasons);
    (d.reasons || []).forEach(function (r) { reasons.appendChild(make("li", r)); });

    var notes = el("notes");
    clear(notes);
    (d.notes || []).forEach(function (n) { notes.appendChild(make("p", n)); });

    el("checklist-intro").textContent = (d.checklist || []).length +
      " documents for a " + d.tierName + ", applied for in " + d.form + " on FoSCoS. " +
      "The list varies by state, food category and entity type, so treat it as a starting point.";

    var list = el("checklist-rows");
    clear(list);
    (d.checklist || []).forEach(function (item) {
      var tr = document.createElement("tr");
      var first = document.createElement("td");
      first.appendChild(make("b", item.document));
      first.appendChild(document.createElement("br"));
      first.appendChild(make("span", item.note, "assumptions"));
      tr.appendChild(first);
      tr.appendChild(make("td", item.appliesTo));
      list.appendChild(tr);
    });
  }

  var pending = null;

  function compute() {
    if (pending) { window.clearTimeout(pending); }
    pending = window.setTimeout(function () {
      pending = null;
      fetch("/api/tier?" + answers(), { headers: { Accept: "application/json" } })
        .then(readJSON)
        .then(renderDecision)
        .catch(function (err) {
          el("tier-name").textContent = "—";
          el("tier-sub").textContent = "Could not work that out: " + err.message;
        });
    }, 120);
  }

  /* ------------------------------------------------------------- watch list */

  function renderWatch(state) {
    if (state.asOf) {
      el("watch-asof").textContent = "Judged as of " + state.asOf +
        ". Saved on this machine only.";
    }
    var rows = el("watch-rows");
    clear(rows);
    var items = state.watchlist || [];
    if (items.length === 0) {
      var empty = document.createElement("tr");
      var cell = make("td", "No licences saved yet. Add one above and it will be watched here.");
      cell.colSpan = 5;
      empty.appendChild(cell);
      rows.appendChild(empty);
      return;
    }
    items.forEach(function (item) {
      var l = item.licence;
      var r = item.renewal;
      var tr = document.createElement("tr");

      var name = document.createElement("td");
      name.appendChild(make("b", l.businessName));
      name.appendChild(document.createElement("br"));
      var detail = l.tier === "basic" ? "Basic Registration"
        : l.tier === "state" ? "State Licence" : "Central Licence";
      if (l.licenceNumber) { detail = l.licenceNumber + " · " + detail; }
      if (l.notes) { detail = detail + " · " + l.notes; }
      name.appendChild(make("span", detail, "assumptions"));
      tr.appendChild(name);

      tr.appendChild(make("td", r.expiryDate));

      var urgent = r.level === "overdue" || r.level === "critical";
      tr.appendChild(make("td", r.level === "unknown" ? "—" : r.daysRemaining,
        urgent ? "num neg" : "num"));

      var status = document.createElement("td");
      status.appendChild(make("span", r.levelName, "pill"));
      tr.appendChild(status);

      var actions = document.createElement("td");
      var remove = make("button", "Remove", "btn ghost");
      remove.type = "button";
      remove.setAttribute("data-remove", l.id);
      actions.appendChild(remove);
      tr.appendChild(actions);

      rows.appendChild(tr);

      var note = document.createElement("tr");
      note.className = "sub";
      var noteCell = make("td", r.message);
      noteCell.colSpan = 5;
      note.appendChild(noteCell);
      rows.appendChild(note);
    });
  }

  function addLicence() {
    var msg = el("add-msg");
    var body = {
      businessName: el("l-name").value,
      licenceNumber: el("l-number").value,
      tier: el("l-tier").value || "state",
      expiryDate: el("l-expiry").value,
      notes: el("l-notes").value
    };
    msg.textContent = "Saving…";
    fetch("/api/licences", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify(body)
    })
      .then(readJSON)
      .then(function (state) {
        renderWatch(state);
        el("l-name").value = "";
        el("l-number").value = "";
        el("l-expiry").value = "";
        el("l-notes").value = "";
        msg.textContent = "Saved to the local data file.";
      })
      .catch(function (err) { msg.textContent = "Not saved: " + err.message; });
  }

  function removeLicence(id) {
    fetch("/api/licences/" + encodeURIComponent(id), {
      method: "DELETE",
      headers: { Accept: "application/json" }
    })
      .then(readJSON)
      .then(renderWatch)
      .catch(function (err) { el("add-msg").textContent = "Not removed: " + err.message; });
  }

  /* -------------------------------------------------- thresholds and fees */

  function renderParams(params) {
    var host = el("params-fields");
    clear(host);
    PARAM_FIELDS.forEach(function (field) {
      var wrap = make("div", null, "field");
      var label = make("label", field.label);
      label.setAttribute("for", "p-" + field.key);
      var input = document.createElement("input");
      input.type = "number";
      input.id = "p-" + field.key;
      input.min = "0";
      input.step = field.step;
      input.value = String(params[field.key]);
      wrap.appendChild(label);
      wrap.appendChild(input);
      host.appendChild(wrap);
    });
  }

  function collectParams() {
    var out = {};
    PARAM_FIELDS.forEach(function (field) {
      var input = el("p-" + field.key);
      var v = parseFloat(input ? input.value : "");
      if (!isFinite(v)) { v = 0; }
      out[field.key] = field.integer ? Math.round(v) : v;
    });
    return out;
  }

  function saveParams() {
    var msg = el("params-msg");
    msg.textContent = "Saving…";
    fetch("/api/params", {
      method: "PUT",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify(collectParams())
    })
      .then(readJSON)
      .then(function (body) {
        renderParams(body.params);
        msg.textContent = "Your figures are in use. The decision above has been recalculated.";
        compute();
        refreshWatch();
      })
      .catch(function (err) { msg.textContent = "Not saved: " + err.message; });
  }

  function resetParams() {
    var msg = el("params-msg");
    fetch("/api/params", { method: "DELETE", headers: { Accept: "application/json" } })
      .then(readJSON)
      .then(function (body) {
        renderParams(body.params);
        msg.textContent = "Back to the published figures.";
        compute();
        refreshWatch();
      })
      .catch(function (err) { msg.textContent = "Not restored: " + err.message; });
  }

  function refreshWatch() {
    fetch("/api/licences", { headers: { Accept: "application/json" } })
      .then(readJSON)
      .then(renderWatch)
      .catch(function () { /* the next action will report it */ });
  }

  /* ------------------------------------------------------------------ setup */

  function fillSelect(select, options, chosen) {
    clear(select);
    options.forEach(function (opt) {
      var node = make("option", opt.label);
      node.value = opt.key;
      select.appendChild(node);
    });
    if (chosen) { select.value = chosen; }
  }

  function resetAnswers() {
    Object.keys(DEFAULT_ANSWERS).forEach(function (id) {
      var node = el(id);
      if (node) { node.value = DEFAULT_ANSWERS[id]; }
    });
    FLAGS.forEach(function (f) { flags[f] = false; });
    document.querySelectorAll("[data-flag]").forEach(function (button) {
      button.setAttribute("aria-pressed", "false");
    });
    compute();
  }

  function wire() {
    ["turnover", "capacity", "dairy", "solids", "states"].forEach(function (id) {
      var node = el(id);
      if (node) { node.addEventListener("input", compute); }
    });
    ["activity", "unit"].forEach(function (id) {
      var node = el(id);
      if (node) { node.addEventListener("change", compute); }
    });
    document.querySelectorAll("[data-flag]").forEach(function (button) {
      button.addEventListener("click", function () {
        var key = button.getAttribute("data-flag");
        flags[key] = !flags[key];
        button.setAttribute("aria-pressed", flags[key] ? "true" : "false");
        compute();
      });
    });

    el("reset").addEventListener("click", resetAnswers);
    el("print").addEventListener("click", function () { window.print(); });
    el("export").addEventListener("click", function () {
      window.location.assign("/api/checklist.csv?" + answers());
    });
    el("add").addEventListener("click", addLicence);
    el("save-params").addEventListener("click", saveParams);
    el("reset-params").addEventListener("click", resetParams);

    el("watch-rows").addEventListener("click", function (event) {
      var id = event.target && event.target.getAttribute
        ? event.target.getAttribute("data-remove") : null;
      if (id) { removeLicence(id); }
    });
  }

  function start() {
    wireTheme();
    FLAGS.forEach(function (f) { flags[f] = false; });

    fetch("/api/state", { headers: { Accept: "application/json" } })
      .then(readJSON)
      .then(function (state) {
        fillSelect(el("activity"), state.activities, DEFAULT_ANSWERS.activity);
        fillSelect(el("unit"), state.units, DEFAULT_ANSWERS.unit);
        fillSelect(el("l-tier"), state.tiers, "state");
        renderParams(state.params);
        renderWatch(state);
        if (state.dataFile) {
          el("watch-asof").textContent = "Judged as of " + state.asOf +
            ". Saved on this machine only, in " + state.dataFile;
        }
        wire();
        compute();
      })
      .catch(function (err) {
        el("tier-sub").textContent = "Could not reach the local server: " + err.message;
      });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", start);
  } else {
    start();
  }
})();
