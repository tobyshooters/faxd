const $ = s => document.querySelector(s);
const $$ = s => document.querySelectorAll(s);

// tabs
let activeTab = location.hash.slice(1) || "home";

function switchTab(tab) {
  $$(".tab").forEach(b => b.classList.remove("active"));
  $$(".page").forEach(p => p.classList.remove("active"));
  const page = $("#" + tab);
  if (!page) return;
  const btn = $(`.tab[data-tab="${tab}"]`);
  if (btn) btn.classList.add("active");
  page.classList.add("active");
  activeTab = tab;
  location.hash = tab;
  load(tab);
}

$$(".tab").forEach(btn => {
  btn.addEventListener("click", () => switchTab(btn.dataset.tab));
});

$("#logo").addEventListener("click", () => switchTab("home"));

switchTab(activeTab);

function load(tab) {
  if (tab === "home") { loadStatus(); loadLog(); }
  if (tab === "settings") loadSettings();
}

async function api(path, opts) {
  const res = await fetch("/api/" + path, opts);
  return res.json();
}

// auto-refresh every 3s
setInterval(() => {
  if (activeTab === "home") { loadStatus(); loadLog(); }
}, 3000);

// home
async function loadStatus() {
  const s = await api("status");
  $("#d-running").textContent = s.running ? "running" : "stopped";
  $("#d-last").textContent = s.last_fax;
  const lines = await api("debug");
  const pre = $("#debug-log");
  pre.textContent = (lines || []).join("\n");
  pre.scrollTop = pre.scrollHeight;
}

// log
async function loadLog() {
  const entries = await api("log");
  const body = $("#log-body");
  body.innerHTML = "";
  if (!entries || entries.length === 0) {
    body.innerHTML = "<tr><td colspan=4>No faxes yet.</td></tr>";
    return;
  }
  entries.slice().reverse().forEach(e => {
    const tr = document.createElement("tr");
    const t = new Date(e.time).toLocaleString();
    tr.innerHTML = `<td>${esc(e.sender)}</td><td>${t}</td><td>${esc(e.filename)}</td><td>${esc(e.status)}</td>`;
    body.appendChild(tr);
  });
}

// settings + whitelist
let cachedConfig = null;

async function loadSettings() {
  cachedConfig = await api("config");
  const f = $("#settings-form");
  f.email.value = cachedConfig.email || "";
  f.password.value = "";
  f.poll.value = cachedConfig.poll_interval_seconds || 30;
  f.maxmb.value = cachedConfig.max_attachment_mb || 20;
  f.exts.value = (cachedConfig.allowed_extensions || []).join(",");
  f.monochrome.checked = cachedConfig.monochrome;
  f.scaling.value = cachedConfig.scaling || 0;
  renderWhitelist();
}

function renderWhitelist() {
  const ul = $("#wl-list");
  ul.innerHTML = "";
  (cachedConfig.allowed_senders || []).forEach((s, i) => {
    const li = document.createElement("li");
    li.innerHTML = `<span>${esc(s)}</span><button data-i="${i}">remove</button>`;
    li.querySelector("button").addEventListener("click", () => removeSender(i));
    ul.appendChild(li);
  });
}

async function removeSender(i) {
  cachedConfig = await api("config");
  cachedConfig.allowed_senders.splice(i, 1);
  await saveSettings(cachedConfig);
  renderWhitelist();
}

$("#wl-add").addEventListener("click", async () => {
  const inp = $("#wl-input");
  const addr = inp.value.trim();
  if (!addr) return;
  cachedConfig = await api("config");
  if (!cachedConfig.allowed_senders) cachedConfig.allowed_senders = [];
  cachedConfig.allowed_senders.push(addr);
  await saveSettings(cachedConfig);
  inp.value = "";
  renderWhitelist();
});

$("#settings-form").addEventListener("submit", async e => {
  e.preventDefault();
  const f = e.target;
  const cfg = {
    email: f.email.value,
    password: f.password.value,
    poll_interval_seconds: parseInt(f.poll.value),
    max_attachment_mb: parseInt(f.maxmb.value),
    allowed_extensions: f.exts.value.split(",").map(s => s.trim()).filter(Boolean),
    allowed_senders: (cachedConfig || {}).allowed_senders || [],
    monochrome: f.monochrome.checked,
    scaling: parseInt(f.scaling.value),
  };
  await saveSettings(cfg);
  cachedConfig = cfg;
  alert("Settings saved.");
});

async function saveSettings(cfg) {
  await api("config", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(cfg),
  });
}

// send
const fileInput = $("#send-form").file;
const preview = $("#send-preview");

fileInput.addEventListener("change", () => {
  const file = fileInput.files[0];
  if (!file) {
    preview.hidden = true;
    return;
  }
  if (file.type.startsWith("image/")) {
    preview.src = URL.createObjectURL(file);
    preview.hidden = false;
  } else {
    preview.hidden = true;
  }
});

$("#send-form").addEventListener("submit", async e => {
  e.preventDefault();
  const f = e.target;
  const fd = new FormData();
  fd.append("to", f.to.value);
  fd.append("file", f.file.files[0]);
  $("#send-status").textContent = "Sending...";
  try {
    const res = await fetch("/api/send", { method: "POST", body: fd });
    if (res.ok) {
      $("#send-status").textContent = "Sent!";
      f.reset();
      preview.hidden = true;
    } else {
      const text = await res.text();
      $("#send-status").textContent = "Error: " + text;
    }
  } catch (err) {
    $("#send-status").textContent = "Error: " + err.message;
  }
});

function esc(s) {
  const d = document.createElement("div");
  d.textContent = s;
  return d.innerHTML;
}

// initial load
loadStatus();
