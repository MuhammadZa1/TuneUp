const { invoke } = window.__TAURI__.core;

const CHECKS = [
  { name: "audio-crackle", label: "Audio", icon: "♪" },
  { name: "gpu-vulkan", label: "GPU / Vulkan", icon: "▣" },
  { name: "kernel-dkms", label: "Kernel / dkms", icon: "⌘" },
  { name: "swap-zram", label: "Swap / zram", icon: "▤" },
  { name: "battery-wear", label: "Battery", icon: "▮" },
  { name: "disk-scheduler", label: "Disk scheduler", icon: "◍" },
  { name: "thermal-throttle", label: "Thermal", icon: "≈" },
  { name: "wifi-power-save", label: "Wi-Fi", icon: "≋" },
];

const SCAN_INTERVAL_OPTIONS = [
  { minutes: 0, label: "Off" },
  { minutes: 30, label: "Every 30 minutes" },
  { minutes: 60, label: "Every hour" },
  { minutes: 180, label: "Every 3 hours" },
  { minutes: 360, label: "Every 6 hours" },
  { minutes: 720, label: "Every 12 hours" },
  { minutes: 1440, label: "Once a day" },
];

const STATUS_SYMBOLS = { OK: "✓", WARNING: "!", PROBLEM: "✗", INFO: "i", SKIPPED: "-" };

// Lower = more severe = sorts first on the dashboard.
const STATUS_RANK = { PROBLEM: 0, WARNING: 1, INFO: 2, OK: 3, SKIPPED: 4 };

const navItemsEl = document.getElementById("nav-items");
const navSettingsBtn = document.getElementById("nav-settings-btn");
const contentEl = document.getElementById("content");

let currentSettings = null;
let currentView = "dashboard";

// ---------- Sidebar ----------

function renderNav() {
  navItemsEl.innerHTML = "";
  navItemsEl.appendChild(makeNavItem("dashboard", "◈", "Dashboard"));
  for (const check of CHECKS) {
    navItemsEl.appendChild(makeNavItem(check.name, check.icon, check.label));
  }
  updateNavActiveState();
}

function makeNavItem(view, icon, label) {
  const btn = document.createElement("button");
  btn.className = "nav-item";
  btn.dataset.view = view;
  btn.innerHTML = `<span class="nav-icon">${icon}</span> ${label}`;
  btn.addEventListener("click", () => switchView(view));
  return btn;
}

function updateNavActiveState() {
  document.querySelectorAll(".nav-item").forEach((el) => {
    el.classList.toggle("active", el.dataset.view === currentView);
  });
  navSettingsBtn.classList.toggle("active", currentView === "settings");
}

function switchView(view) {
  currentView = view;
  updateNavActiveState();
  if (view === "dashboard") renderDashboardView();
  else if (view === "settings") renderSettingsView();
  else renderCheckView(view);
}

navSettingsBtn.addEventListener("click", () => switchView("settings"));

// ---------- Shared check-card rendering ----------

function statusClass(status) {
  return status.toLowerCase();
}

function buildCheckCard(result) {
  const card = document.createElement("div");
  card.className = "check-card status-" + statusClass(result.status);
  card.dataset.check = result.check;

  const row = document.createElement("div");
  row.className = "check-row";

  const symbol = document.createElement("div");
  symbol.className = "symbol " + statusClass(result.status);
  symbol.textContent = STATUS_SYMBOLS[result.status] || "?";
  row.appendChild(symbol);

  const textWrap = document.createElement("div");
  textWrap.className = "check-text";

  const title = document.createElement("div");
  title.className = "check-title";
  title.textContent = result.title;
  textWrap.appendChild(title);

  const message = document.createElement("div");
  message.className = "check-message";
  message.textContent = result.message;
  textWrap.appendChild(message);

  row.appendChild(textWrap);

  const actionWrap = document.createElement("div");
  actionWrap.className = "check-action";

  if (result.fix_applied) {
    actionWrap.appendChild(makeFixStatus("Fixed", "applied"));
  } else if (result.fix_error) {
    actionWrap.appendChild(makeFixStatus("Fix failed", "failed", result.fix_error));
  } else if (result.fix_available) {
    const fixBtn = document.createElement("button");
    fixBtn.className = "fix-btn";
    fixBtn.textContent = "Fix";
    fixBtn.addEventListener("click", () => applyFix(result.check, fixBtn));
    actionWrap.appendChild(fixBtn);
  } else if (result.detail) {
    const isAdvisory = result.status === "WARNING" || result.status === "PROBLEM";
    const toggleBtn = document.createElement("button");
    toggleBtn.className = "detail-toggle-btn";
    toggleBtn.textContent = isAdvisory ? "What can I do?" : "Details";
    toggleBtn.addEventListener("click", () => toggleDetail(card, toggleBtn, isAdvisory));
    actionWrap.appendChild(toggleBtn);
  }

  row.appendChild(actionWrap);
  card.appendChild(row);

  if (result.detail) {
    const detail = document.createElement("div");
    detail.className = "check-detail";
    detail.textContent = result.detail;
    detail.hidden = true;
    card.appendChild(detail);
  }

  return card;
}

function toggleDetail(card, btn, isAdvisory) {
  const detail = card.querySelector(".check-detail");
  if (!detail) return;
  detail.hidden = !detail.hidden;
  if (isAdvisory) {
    btn.textContent = detail.hidden ? "What can I do?" : "Hide";
  } else {
    btn.textContent = detail.hidden ? "Details" : "Hide";
  }
}

function makeFixStatus(text, cls, tooltip) {
  const el = document.createElement("div");
  el.className = "fix-status " + cls;
  el.textContent = text;
  if (tooltip) el.title = tooltip;
  return el;
}

async function applyFix(checkName, buttonEl) {
  buttonEl.disabled = true;
  buttonEl.textContent = "Fixing…";
  try {
    const data = await invoke("apply_fix", { checkName });
    const result = data.results.find((r) => r.check === checkName);
    if (!result) return;

    const card = document.querySelector(`.check-card[data-check="${cssEscape(checkName)}"]`);
    if (!card) return;

    card.className = "check-card status-" + statusClass(result.status);
    const symbol = card.querySelector(".symbol");
    symbol.className = "symbol " + statusClass(result.status);
    symbol.textContent = STATUS_SYMBOLS[result.status] || "?";
    card.querySelector(".check-message").textContent = result.message;

    const existingDetail = card.querySelector(".check-detail");
    if (existingDetail) {
      if (result.detail) {
        existingDetail.textContent = result.detail;
      } else {
        existingDetail.remove();
      }
    }

    if (result.fix_applied) {
      buttonEl.replaceWith(makeFixStatus("Fixed", "applied"));
    } else if (result.fix_error) {
      buttonEl.replaceWith(makeFixStatus("Fix failed", "failed", result.fix_error));
    } else {
      buttonEl.disabled = false;
      buttonEl.textContent = "Fix";
    }
  } catch (err) {
    buttonEl.replaceWith(makeFixStatus("Fix failed", "failed", String(err)));
  }
}

function sortBySeverity(results) {
  return [...results].sort((a, b) => (STATUS_RANK[a.status] ?? 9) - (STATUS_RANK[b.status] ?? 9));
}

function buildSummaryBanner(data) {
  const banner = document.createElement("div");
  const ok = data.problems_found === 0;
  banner.className = "summary-banner " + (ok ? "summary-ok" : "summary-warn");
  banner.innerHTML = ok
    ? `<span class="summary-symbol">✓</span> All good — no issues found`
    : `<span class="summary-symbol">!</span> ${data.problems_found} check${data.problems_found === 1 ? "" : "s"} flagged something`;
  return banner;
}

// ---------- Dashboard view ----------

async function renderDashboardView() {
  contentEl.innerHTML = `
    <div class="view-header">
      <div>
        <h1>Dashboard</h1>
        <p class="subtitle">All checks at a glance</p>
      </div>
      <button class="action-btn" id="scan-all-btn">Scan all</button>
    </div>
    <div id="summary-slot"></div>
    <div class="check-list" id="check-list">
      <p class="loading">Running checks…</p>
    </div>
  `;

  document.getElementById("scan-all-btn").addEventListener("click", loadDashboardResults);
  await loadDashboardResults();
}

async function loadDashboardResults() {
  const btn = document.getElementById("scan-all-btn");
  const listEl = document.getElementById("check-list");
  const summarySlot = document.getElementById("summary-slot");
  if (btn) btn.disabled = true;
  listEl.innerHTML = '<p class="loading">Running checks…</p>';
  if (summarySlot) summarySlot.innerHTML = "";

  try {
    const data = await invoke("run_checks");
    listEl.innerHTML = "";
    if (summarySlot) summarySlot.appendChild(buildSummaryBanner(data));
    for (const result of sortBySeverity(data.results)) {
      listEl.appendChild(buildCheckCard(result));
    }
  } catch (err) {
    listEl.innerHTML = `<p class="error">${escapeHtml(String(err))}</p>`;
  } finally {
    if (btn) btn.disabled = false;
  }
}

// ---------- Single-check view ----------

async function renderCheckView(checkName) {
  const check = CHECKS.find((c) => c.name === checkName);
  const label = check ? check.label : checkName;

  contentEl.innerHTML = `
    <div class="view-header">
      <div>
        <h1>${escapeHtml(label)}</h1>
        <p class="subtitle">Run this check on its own</p>
      </div>
      <button class="action-btn" id="scan-one-btn">Scan</button>
    </div>
    <div class="check-list" id="check-list">
      <p class="loading">Running check…</p>
    </div>
  `;

  document.getElementById("scan-one-btn").addEventListener("click", () => loadSingleCheck(checkName));
  await loadSingleCheck(checkName);
}

async function loadSingleCheck(checkName) {
  const btn = document.getElementById("scan-one-btn");
  const listEl = document.getElementById("check-list");
  if (btn) btn.disabled = true;
  listEl.innerHTML = '<p class="loading">Running check…</p>';

  try {
    const data = await invoke("run_check", { checkName });
    listEl.innerHTML = "";
    if (!data.results.length) {
      listEl.innerHTML = '<p class="error">No result returned for this check.</p>';
      return;
    }
    listEl.appendChild(buildCheckCard(data.results[0]));
  } catch (err) {
    listEl.innerHTML = `<p class="error">${escapeHtml(String(err))}</p>`;
  } finally {
    if (btn) btn.disabled = false;
  }
}

// ---------- Settings view ----------

function renderSettingsView() {
  const s = currentSettings;

  contentEl.innerHTML = `
    <div class="view-header">
      <div>
        <h1>Settings</h1>
        <p class="subtitle">Appearance, tray, startup, and scan schedule</p>
      </div>
    </div>

    <div class="settings-group">
      <div class="settings-row">
        <div>
          <div class="settings-row-label">Theme</div>
          <div class="settings-row-desc">Switch between light and dark appearance</div>
        </div>
        <div class="theme-toggle">
          <button data-theme-choice="light" class="${s.theme === "light" ? "active" : ""}">Light</button>
          <button data-theme-choice="dark" class="${s.theme === "dark" ? "active" : ""}">Dark</button>
        </div>
      </div>
    </div>

    <div class="settings-group">
      <div class="settings-row">
        <div>
          <div class="settings-row-label">Show in system tray</div>
          <div class="settings-row-desc">Keep a tuneup icon in the panel; closing the window minimizes to tray instead of quitting</div>
        </div>
        <label class="switch">
          <input type="checkbox" id="tray-toggle" ${s.show_in_tray ? "checked" : ""} />
          <span class="switch-track"></span>
        </label>
      </div>

      <div class="settings-row">
        <div>
          <div class="settings-row-label">Launch on login</div>
          <div class="settings-row-desc">Start tuneup automatically when you log in</div>
        </div>
        <label class="switch">
          <input type="checkbox" id="autostart-toggle" ${s.launch_on_login ? "checked" : ""} />
          <span class="switch-track"></span>
        </label>
      </div>

      <div class="settings-row">
        <div>
          <div class="settings-row-label">Automatic background scans</div>
          <div class="settings-row-desc">Run all checks periodically and notify you if something's found</div>
        </div>
        <select class="settings-select" id="interval-select">
          ${SCAN_INTERVAL_OPTIONS.map(
            (opt) =>
              `<option value="${opt.minutes}" ${opt.minutes === s.scan_interval_minutes ? "selected" : ""}>${opt.label}</option>`
          ).join("")}
        </select>
      </div>
    </div>

    <p class="settings-saved-note" id="settings-saved-note"></p>
  `;

  document.querySelectorAll(".theme-toggle button").forEach((btn) => {
    btn.addEventListener("click", () => {
      document.querySelectorAll(".theme-toggle button").forEach((b) => b.classList.remove("active"));
      btn.classList.add("active");
      currentSettings.theme = btn.dataset.themeChoice;
      applyTheme(currentSettings.theme);
      persistSettings();
    });
  });

  document.getElementById("tray-toggle").addEventListener("change", (e) => {
    currentSettings.show_in_tray = e.target.checked;
    persistSettings();
  });

  document.getElementById("autostart-toggle").addEventListener("change", (e) => {
    currentSettings.launch_on_login = e.target.checked;
    persistSettings();
  });

  document.getElementById("interval-select").addEventListener("change", (e) => {
    currentSettings.scan_interval_minutes = parseInt(e.target.value, 10);
    persistSettings();
  });
}

async function persistSettings() {
  const note = document.getElementById("settings-saved-note");
  try {
    await invoke("save_settings", { newSettings: currentSettings });
    if (note) {
      note.textContent = "Saved.";
      setTimeout(() => {
        if (note.textContent === "Saved.") note.textContent = "";
      }, 1500);
    }
  } catch (err) {
    if (note) note.textContent = "Couldn't save: " + String(err);
  }
}

// ---------- Theme ----------

function applyTheme(theme) {
  document.documentElement.setAttribute("data-theme", theme);
}

// ---------- Utilities ----------

function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str;
  return div.innerHTML;
}

function cssEscape(str) {
  return window.CSS && CSS.escape ? CSS.escape(str) : str.replace(/["\\]/g, "\\$&");
}

// ---------- Init ----------

async function init() {
  try {
    currentSettings = await invoke("get_settings");
  } catch (err) {
    currentSettings = {
      theme: "light",
      show_in_tray: true,
      launch_on_login: false,
      scan_interval_minutes: 0,
    };
  }
  applyTheme(currentSettings.theme);
  renderNav();
  switchView("dashboard");
}

init();
