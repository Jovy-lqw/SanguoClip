const app = {
  state: {
    assets: [],
    captures: [],
    edits: [],
    styles: [],
    model_config: {}
  },
  devices: [],
  environment: null,
  modelInitialized: false,
  segmentSeededFor: "",
  refreshTimer: null
};

const viewLabels = {
  workspace: ["今日工作区", "视频工作台"],
  library: ["本地媒体库", "素材与成片"],
  styles: ["复用你的剪辑偏好", "剪辑风格"],
  settings: ["运行配置", "模型与环境"]
};

document.addEventListener("DOMContentLoaded", () => {
  bindNavigation();
  bindActions();
  addSegment({ start: 0, end: 30, label: "开场" });
  refreshAll();
});

function bindNavigation() {
  document.querySelectorAll("[data-view-target]").forEach((button) => {
    button.addEventListener("click", () => showView(button.dataset.viewTarget));
  });
}

function showView(name) {
  document.querySelectorAll("[data-view-target]").forEach((item) => {
    item.classList.toggle("active", item.dataset.viewTarget === name);
  });
  document.querySelectorAll("[data-view]").forEach((view) => {
    view.classList.toggle("active", view.dataset.view === name);
  });
  const [eyebrow, title] = viewLabels[name];
  document.querySelector("#page-eyebrow").textContent = eyebrow;
  document.querySelector("#page-title").textContent = title;
  window.scrollTo({ top: 0, behavior: "smooth" });
}

function bindActions() {
  document.querySelector("#refresh-button").addEventListener("click", refreshAll);
  document.querySelector("#diagnose-button").addEventListener("click", refreshEnvironment);
  document.querySelector("#scan-devices-button").addEventListener("click", scanDevices);
  document.querySelector("#device-select").addEventListener("change", renderClientOptions);
  document.querySelector("#capture-form").addEventListener("submit", startCapture);
  document.querySelector("#active-capture").addEventListener("click", stopCapture);
  document.querySelector("#add-segment-button").addEventListener("click", () => addSegmentAtPlayhead());
  document.querySelector("#segments").addEventListener("click", removeSegment);
  document.querySelector("#editor-asset").addEventListener("change", selectEditorAsset);
  document.querySelector("#edit-form").addEventListener("submit", saveEdit);
  document.querySelector("#suggest-button").addEventListener("click", suggestEdit);
  document.querySelector("#recent-edits").addEventListener("click", editListAction);
  document.querySelector("#edit-list").addEventListener("click", editListAction);
  document.querySelector("#asset-list").addEventListener("click", assetListAction);
  document.querySelector("#new-style-button").addEventListener("click", resetStyleForm);
  document.querySelector("#style-list").addEventListener("click", selectStyle);
  document.querySelector("#style-form").addEventListener("submit", saveStyle);
  document.querySelector("#model-form").addEventListener("submit", saveModel);
  document.querySelector("[name=temperature]").addEventListener("input", updateTemperature);

  const dialog = document.querySelector("#upload-dialog");
  document.querySelectorAll("[data-action=open-upload]").forEach((button) => {
    button.addEventListener("click", () => dialog.showModal());
  });
  document.querySelector("#upload-input").addEventListener("change", (event) => {
    document.querySelector("#upload-file-label").textContent =
      event.target.files[0]?.name || "选择视频文件";
  });
  document.querySelector("#upload-form").addEventListener("submit", uploadMedia);

  const exportDialog = document.querySelector("#export-dialog");
  exportDialog.addEventListener("close", () => {
    const video = document.querySelector("#export-preview");
    video.pause();
    video.removeAttribute("src");
    video.load();
  });
}

async function refreshAll() {
  try {
    const [state, environment] = await Promise.all([
      api("/api/state"),
      api("/api/environment")
    ]);
    app.state = normalizeState(state);
    app.environment = environment;
    renderAll();
    setServiceStatus(true);
    scheduleRefresh();
  } catch (error) {
    setServiceStatus(false);
    toast(error.message, true);
  }
}

async function refreshState() {
  try {
    app.state = normalizeState(await api("/api/state"));
    renderAll();
    scheduleRefresh();
  } catch (error) {
    toast(error.message, true);
  }
}

async function refreshEnvironment() {
  try {
    app.environment = await api("/api/environment");
    renderEnvironment();
    toast("环境诊断已更新");
  } catch (error) {
    toast(error.message, true);
  }
}

function renderAll() {
  renderMetrics();
  renderAssetOptions();
  renderActiveCapture();
  renderEdits("#recent-edits", app.state.edits.slice(0, 5));
  renderEdits("#edit-list", app.state.edits);
  renderAssets();
  renderStyles();
  renderStyleExamples();
  renderModel();
  renderEnvironment();
  refreshIcons();
}

function renderMetrics() {
  const completed = app.state.edits.filter((edit) => edit.status === "completed").length;
  const recording = app.state.captures.filter((capture) => capture.status === "recording").length;
  const duration = app.state.assets.reduce((sum, asset) => sum + (asset.duration || 0), 0);
  document.querySelector("#metrics").innerHTML = [
    ["本地素材", app.state.assets.length],
    ["剪辑方案", app.state.edits.length],
    ["可下载成片", completed],
    [recording ? "正在录制" : "素材总时长", recording || formatDuration(duration)]
  ].map(([label, value]) => `
    <div class="metric"><span>${escapeHTML(label)}</span><strong>${escapeHTML(String(value))}</strong></div>
  `).join("");
}

function renderAssetOptions() {
  const select = document.querySelector("#editor-asset");
  const previous = select.value;
  select.innerHTML = app.state.assets.length
    ? app.state.assets.map((asset) => `<option value="${asset.id}">${escapeHTML(asset.name)}</option>`).join("")
    : `<option value="">暂无素材</option>`;
  if (app.state.assets.some((asset) => asset.id === previous)) {
    select.value = previous;
  }

  const styleSelect = document.querySelector("#editor-style");
  const previousStyle = styleSelect.value;
  styleSelect.innerHTML = app.state.styles
    .map((style) => `<option value="${style.id}">${escapeHTML(style.name)}</option>`)
    .join("");
  if (app.state.styles.some((style) => style.id === previousStyle)) {
    styleSelect.value = previousStyle;
  }
  selectEditorAsset();
}

function selectEditorAsset() {
  const assetID = document.querySelector("#editor-asset").value;
  const asset = app.state.assets.find((item) => item.id === assetID);
  const stage = document.querySelector(".preview-stage");
  const video = document.querySelector("#video-preview");
  if (!asset) {
    video.removeAttribute("src");
    stage.classList.remove("has-video");
    document.querySelector("#editor-duration").textContent = "未选择素材";
    return;
  }

  if (video.dataset.assetId !== asset.id) {
    video.src = `/api/media/${encodeURIComponent(asset.id)}`;
    video.dataset.assetId = asset.id;
    stage.classList.add("has-video");
  }
  document.querySelector("#editor-duration").textContent =
    asset.duration > 0 ? `素材 ${formatDuration(asset.duration)}` : "时长待探测";

  if (app.segmentSeededFor !== asset.id && asset.duration > 0) {
    document.querySelector("#segments").innerHTML = "";
    addSegment({ start: 0, end: Math.min(asset.duration, 30), label: "开场" });
    app.segmentSeededFor = asset.id;
  }
}

function renderActiveCapture() {
  const container = document.querySelector("#active-capture");
  const current = app.state.captures.find((capture) => capture.status === "recording");
  if (!current) {
    container.innerHTML = "";
    return;
  }
  container.innerHTML = `
    <div class="capture-running">
      <div>
        <strong><span class="record-indicator"></span>${escapeHTML(current.label)}</strong>
        <small>${escapeHTML(current.device_serial)} · ${formatClockSince(current.started_at)}</small>
      </div>
      <button class="primary-button danger-button compact" data-stop-capture="${current.id}">
        <i data-lucide="square"></i><span>停止并导入</span>
      </button>
    </div>`;
}

function renderEdits(selector, edits) {
  const container = document.querySelector(selector);
  if (!edits.length) {
    container.innerHTML = `<div class="empty-state">还没有剪辑方案</div>`;
    return;
  }
  container.innerHTML = edits.map((edit) => {
    const asset = app.state.assets.find((item) => item.id === edit.asset_id);
    const total = edit.segments.reduce((sum, segment) => sum + segment.end - segment.start, 0);
    const action = edit.status === "completed"
      ? `<button class="primary-button compact" data-watch-edit="${edit.id}">
          <i data-lucide="play"></i><span>观看</span>
        </button>
        <a class="secondary-button compact" href="/api/exports/${edit.id}?download=1">
          <i data-lucide="download"></i><span>下载</span>
        </a>`
      : `<button class="primary-button compact" data-render-edit="${edit.id}" ${edit.status === "rendering" ? "disabled" : ""}>
          <i data-lucide="${edit.status === "rendering" ? "loader-circle" : "play"}"></i>
          <span>${edit.status === "rendering" ? "渲染中" : "开始渲染"}</span>
        </button>`;
    return `
      <div class="data-item">
        <div>
          <h3>${escapeHTML(edit.title)} <span class="pill ${edit.status}">${statusLabel(edit.status)}</span></h3>
          <p>${escapeHTML(asset?.name || "素材已丢失")} · ${edit.segments.length} 段 · ${formatDuration(total)}</p>
          ${edit.error ? `<p class="error-text">${escapeHTML(edit.error)}</p>` : ""}
        </div>
        <div class="row-actions">${action}</div>
      </div>`;
  }).join("");
}

function renderAssets() {
  const container = document.querySelector("#asset-list");
  if (!app.state.assets.length) {
    container.innerHTML = `<div class="empty-state">导入一段录像后，素材会显示在这里</div>`;
    return;
  }
  container.innerHTML = app.state.assets.map((asset) => `
    <article class="media-item">
      <button class="media-thumb" data-use-asset="${asset.id}" title="在剪辑器中打开" aria-label="在剪辑器中打开 ${escapeHTML(asset.name)}">
        <i data-lucide="play"></i>
      </button>
      <div class="media-body">
        <h3 title="${escapeHTML(asset.name)}">${escapeHTML(asset.name)}</h3>
        <div class="media-meta">
          <span>${asset.source === "capture" ? "设备采集" : "本地导入"}</span>
          <span>${asset.duration > 0 ? formatDuration(asset.duration) : formatBytes(asset.size)}</span>
        </div>
      </div>
    </article>
  `).join("");
}

function renderStyles() {
  const container = document.querySelector("#style-list");
  if (!app.state.styles.length) {
    container.innerHTML = `<div class="empty-state">还没有剪辑风格</div>`;
    return;
  }
  const activeID = document.querySelector("#style-form [name=id]").value;
  container.innerHTML = app.state.styles.map((style) => `
    <div class="style-item ${activeID === style.id ? "active" : ""}" data-style-id="${style.id}">
      <h3>${escapeHTML(style.name)}</h3>
      <p>${escapeHTML(style.description || style.prompt)}</p>
      <p>${style.example_asset_ids.length} 个示例素材</p>
    </div>
  `).join("");
}

function renderStyleExamples(selected = null) {
  const container = document.querySelector("#style-example-assets");
  const checked = new Set(selected || checkedStyleAssets());
  if (!app.state.assets.length) {
    container.innerHTML = `<small>导入素材后可将其关联为风格示例。</small>`;
    return;
  }
  container.innerHTML = app.state.assets.map((asset) => `
    <label>
      <input type="checkbox" name="example_asset_ids" value="${asset.id}" ${checked.has(asset.id) ? "checked" : ""}>
      <span>${escapeHTML(asset.name)}</span>
    </label>
  `).join("");
}

function renderModel() {
  if (app.modelInitialized) return;
  const form = document.querySelector("#model-form");
  const config = app.state.model_config || {};
  form.elements.base_url.value = config.base_url || "https://api.openai.com/v1";
  form.elements.model.value = config.model || "";
  form.elements.api_key_env.value = config.api_key_env || "SANGUOCLIP_MODEL_API_KEY";
  form.elements.temperature.value = config.temperature ?? 0.2;
  updateTemperature();
  app.modelInitialized = true;
}

function renderEnvironment() {
  if (!app.environment) return;
  const items = [
    ["ADB", app.environment.adb],
    ["FFmpeg", app.environment.ffmpeg],
    ["FFprobe", app.environment.ffprobe]
  ];
  document.querySelector("#environment-list").innerHTML = items.map(([name, value]) => `
    <div class="environment-row">
      <strong>${name}</strong>
      <code title="${escapeHTML(value.path || "")}">${escapeHTML(value.version || value.path || "未检测到")}</code>
      <strong class="${value.available ? "available" : "missing"}">${value.available ? "可用" : "缺失"}</strong>
    </div>
  `).join("") + `
    <div class="environment-row">
      <strong>数据目录</strong>
      <code title="${escapeHTML(app.environment.data_dir)}">${escapeHTML(app.environment.data_dir)}</code>
      <strong class="available">本地</strong>
    </div>`;
  document.querySelector("#environment-warnings").innerHTML = app.environment.warnings
    .map((warning) => `<div class="notice neutral"><i data-lucide="triangle-alert"></i><span>${escapeHTML(warning)}</span></div>`)
    .join("");
  refreshIcons();
}

async function scanDevices() {
  const button = document.querySelector("#scan-devices-button");
  setBusy(button, true);
  try {
    const result = await api("/api/devices");
    app.devices = result.devices || [];
    const select = document.querySelector("#device-select");
    select.innerHTML = app.devices.length
      ? app.devices.map((device) => `<option value="${escapeHTML(device.serial)}">${escapeHTML(device.model || device.serial)} · ${escapeHTML(device.state)}</option>`).join("")
      : `<option value="">没有发现设备</option>`;
    renderClientOptions();
    toast(app.devices.length ? `发现 ${app.devices.length} 台设备` : "没有发现已连接设备");
  } catch (error) {
    toast(error.message, true);
  } finally {
    setBusy(button, false);
  }
}

function renderClientOptions() {
  const serial = document.querySelector("#device-select").value;
  const device = app.devices.find((item) => item.serial === serial);
  const clients = device?.clients || [];
  const select = document.querySelector("#client-select");
  select.innerHTML = clients.length
    ? clients.map((client) => `
        <option value="${escapeHTML(client.package_name)}">
          ${escapeHTML(client.channel)} · ${escapeHTML(client.package_name)}
        </option>`).join("")
    : `<option value="">未发现支持分享链接的客户端</option>`;
  select.disabled = !clients.length;
}

async function startCapture(event) {
  event.preventDefault();
  const button = event.currentTarget.querySelector("[type=submit]");
  const form = new FormData(event.currentTarget);
  setBusy(button, true);
  try {
    await api("/api/captures/start", {
      method: "POST",
      body: JSON.stringify({
        share_url: form.get("share_url"),
        device_serial: form.get("device_serial"),
        package_name: form.get("package_name"),
        label: form.get("label")
      })
    });
    toast("录制已开始，请在游戏中播放对局");
    await refreshState();
  } catch (error) {
    toast(error.message, true);
  } finally {
    setBusy(button, false);
  }
}

async function stopCapture(event) {
  const button = event.target.closest("[data-stop-capture]");
  if (!button) return;
  setBusy(button, true);
  try {
    const result = await api("/api/captures/stop", {
      method: "POST",
      body: JSON.stringify({ id: button.dataset.stopCapture })
    });
    toast(result.warning ? `录制已导入；${result.warning}` : "录制已停止并导入素材");
    await refreshState();
  } catch (error) {
    toast(error.message, true);
    await refreshState();
  }
}

async function uploadMedia(event) {
  event.preventDefault();
  const button = event.currentTarget.querySelector("[type=submit]");
  const input = document.querySelector("#upload-input");
  if (!input.files[0]) {
    toast("请选择视频文件", true);
    return;
  }
  const form = new FormData();
  form.append("file", input.files[0]);
  setBusy(button, true);
  try {
    const result = await api("/api/media/upload", { method: "POST", body: form }, false);
    document.querySelector("#upload-dialog").close();
    input.value = "";
    document.querySelector("#upload-file-label").textContent = "选择视频文件";
    toast(result.warning ? `素材已导入；${result.warning}` : "素材已导入");
    await refreshState();
    document.querySelector("#editor-asset").value = result.asset.id;
    selectEditorAsset();
  } catch (error) {
    toast(error.message, true);
  } finally {
    setBusy(button, false);
  }
}

function addSegmentAtPlayhead() {
  const video = document.querySelector("#video-preview");
  const asset = currentAsset();
  const start = Number.isFinite(video.currentTime) ? video.currentTime : 0;
  const limit = asset?.duration || start + 30;
  addSegment({ start, end: Math.min(start + 15, limit), label: "" });
}

function addSegment(segment) {
  const row = document.createElement("div");
  row.className = "segment-row";
  row.innerHTML = `
    <input name="segment_start" inputmode="decimal" value="${formatTimestamp(segment.start)}" aria-label="开始时间" title="开始时间">
    <input name="segment_end" inputmode="decimal" value="${formatTimestamp(segment.end)}" aria-label="结束时间" title="结束时间">
    <input name="segment_label" value="${escapeHTML(segment.label || "")}" placeholder="片段说明" aria-label="片段说明">
    <button class="icon-button" type="button" data-remove-segment title="删除片段" aria-label="删除片段"><i data-lucide="trash-2"></i></button>`;
  document.querySelector("#segments").appendChild(row);
  refreshIcons();
}

function removeSegment(event) {
  const button = event.target.closest("[data-remove-segment]");
  if (!button) return;
  if (document.querySelectorAll(".segment-row").length === 1) {
    toast("至少保留一个片段", true);
    return;
  }
  button.closest(".segment-row").remove();
}

function readSegments() {
  return [...document.querySelectorAll(".segment-row")].map((row) => ({
    start: parseTimestamp(row.querySelector("[name=segment_start]").value),
    end: parseTimestamp(row.querySelector("[name=segment_end]").value),
    label: row.querySelector("[name=segment_label]").value.trim()
  }));
}

async function suggestEdit() {
  const form = document.querySelector("#edit-form");
  const button = document.querySelector("#suggest-button");
  const assetID = form.elements.asset_id.value;
  const styleID = form.elements.style_id.value;
  if (!assetID || !styleID) {
    toast("请先选择素材和剪辑风格", true);
    return;
  }
  setBusy(button, true);
  try {
    const result = await api("/api/edits/suggest", {
      method: "POST",
      body: JSON.stringify({
        asset_id: assetID,
        style_id: styleID,
        notes: form.elements.notes.value
      })
    });
    document.querySelector("#segments").innerHTML = "";
    result.segments.forEach(addSegment);
    form.dataset.modelSummary = result.summary || "";
    toast(result.summary || "模型已生成时间段");
  } catch (error) {
    toast(error.message, true);
  } finally {
    setBusy(button, false);
  }
}

async function saveEdit(event) {
  event.preventDefault();
  const button = event.currentTarget.querySelector("[type=submit]");
  setBusy(button, true);
  try {
    const form = event.currentTarget.elements;
    const edit = await api("/api/edits", {
      method: "POST",
      body: JSON.stringify({
        title: form.title.value,
        asset_id: form.asset_id.value,
        style_id: form.style_id.value,
        segments: readSegments(),
        model_summary: event.currentTarget.dataset.modelSummary || ""
      })
    });
    toast(`已保存“${edit.title}”`);
    await refreshState();
  } catch (error) {
    toast(error.message, true);
  } finally {
    setBusy(button, false);
  }
}

async function editListAction(event) {
  const watchButton = event.target.closest("[data-watch-edit]");
  if (watchButton) {
    const edit = app.state.edits.find((item) => item.id === watchButton.dataset.watchEdit);
    if (edit) showExportPreview(edit);
    return;
  }

  const button = event.target.closest("[data-render-edit]");
  if (!button) return;
  setBusy(button, true);
  try {
    await api("/api/edits/render", {
      method: "POST",
      body: JSON.stringify({ id: button.dataset.renderEdit })
    });
    toast("已加入渲染队列");
    await refreshState();
  } catch (error) {
    toast(error.message, true);
    setBusy(button, false);
  }
}

function showExportPreview(edit) {
  const dialog = document.querySelector("#export-dialog");
  const video = document.querySelector("#export-preview");
  document.querySelector("#export-preview-title").textContent = edit.title;
  document.querySelector("#export-download").href =
    `/api/exports/${encodeURIComponent(edit.id)}?download=1`;
  video.src = `/api/exports/${encodeURIComponent(edit.id)}`;
  dialog.showModal();
  video.play().catch(() => {});
}

function assetListAction(event) {
  const button = event.target.closest("[data-use-asset]");
  if (!button) return;
  showView("workspace");
  document.querySelector("#editor-asset").value = button.dataset.useAsset;
  selectEditorAsset();
}

function resetStyleForm() {
  const form = document.querySelector("#style-form");
  form.reset();
  form.elements.id.value = "";
  document.querySelector("#style-form-title").textContent = "新建风格";
  renderStyleExamples([]);
  renderStyles();
}

function selectStyle(event) {
  const item = event.target.closest("[data-style-id]");
  if (!item) return;
  const style = app.state.styles.find((candidate) => candidate.id === item.dataset.styleId);
  if (!style) return;
  const form = document.querySelector("#style-form");
  form.elements.id.value = style.id;
  form.elements.name.value = style.name;
  form.elements.description.value = style.description;
  form.elements.prompt.value = style.prompt;
  document.querySelector("#style-form-title").textContent = "编辑风格";
  renderStyleExamples(style.example_asset_ids);
  renderStyles();
}

async function saveStyle(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const button = form.querySelector("[type=submit]");
  const id = form.elements.id.value;
  setBusy(button, true);
  try {
    const payload = {
      name: form.elements.name.value,
      description: form.elements.description.value,
      prompt: form.elements.prompt.value,
      example_asset_ids: checkedStyleAssets()
    };
    const saved = await api(id ? `/api/styles/${encodeURIComponent(id)}` : "/api/styles", {
      method: id ? "PUT" : "POST",
      body: JSON.stringify(payload)
    });
    toast(`已保存风格“${saved.name}”`);
    await refreshState();
    resetStyleForm();
  } catch (error) {
    toast(error.message, true);
  } finally {
    setBusy(button, false);
  }
}

function checkedStyleAssets() {
  return [...document.querySelectorAll("#style-example-assets input:checked")].map((input) => input.value);
}

async function saveModel(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const button = form.querySelector("[type=submit]");
  setBusy(button, true);
  try {
    const saved = await api("/api/model", {
      method: "PUT",
      body: JSON.stringify({
        base_url: form.elements.base_url.value,
        model: form.elements.model.value,
        api_key_env: form.elements.api_key_env.value,
        temperature: Number(form.elements.temperature.value)
      })
    });
    app.state.model_config = saved;
    toast("模型配置已保存");
  } catch (error) {
    toast(error.message, true);
  } finally {
    setBusy(button, false);
  }
}

function updateTemperature() {
  document.querySelector("#temperature-value").textContent =
    Number(document.querySelector("[name=temperature]").value).toFixed(1);
}

function currentAsset() {
  const id = document.querySelector("#editor-asset").value;
  return app.state.assets.find((asset) => asset.id === id);
}

function scheduleRefresh() {
  clearTimeout(app.refreshTimer);
  const active = app.state.edits.some((edit) => edit.status === "rendering") ||
    app.state.captures.some((capture) => capture.status === "recording");
  if (active) {
    app.refreshTimer = setTimeout(refreshState, 2000);
  }
}

function normalizeState(state) {
  return {
    ...state,
    assets: Array.isArray(state?.assets) ? state.assets : [],
    captures: Array.isArray(state?.captures) ? state.captures : [],
    edits: Array.isArray(state?.edits) ? state.edits : [],
    styles: Array.isArray(state?.styles)
      ? state.styles.map((style) => ({
          ...style,
          example_asset_ids: Array.isArray(style.example_asset_ids) ? style.example_asset_ids : []
        }))
      : [],
    model_config: state?.model_config || {}
  };
}

async function api(path, options = {}, jsonBody = true) {
  const headers = new Headers(options.headers || {});
  if (jsonBody && options.body) {
    headers.set("Content-Type", "application/json");
  }
  const response = await fetch(path, { ...options, headers });
  const contentType = response.headers.get("content-type") || "";
  const result = contentType.includes("application/json") ? await response.json() : null;
  if (!response.ok) {
    throw new Error(result?.error || `请求失败 (${response.status})`);
  }
  return result;
}

function setBusy(button, busy) {
  if (!button) return;
  button.disabled = busy;
  button.classList.toggle("busy", busy);
}

function setServiceStatus(online) {
  document.querySelector("#service-dot").classList.toggle("online", online);
  document.querySelector("#service-label").textContent = online ? "服务运行中" : "服务未连接";
}

function toast(message, isError = false) {
  const item = document.createElement("div");
  item.className = `toast${isError ? " error" : ""}`;
  item.textContent = message;
  document.querySelector("#toast-region").appendChild(item);
  setTimeout(() => item.remove(), 5000);
}

function statusLabel(status) {
  return {
    draft: "待渲染",
    rendering: "渲染中",
    completed: "已完成",
    failed: "失败",
    recording: "录制中"
  }[status] || status;
}

function parseTimestamp(value) {
  const parts = String(value).trim().split(":").map(Number);
  if (parts.some((part) => !Number.isFinite(part) || part < 0)) return -1;
  if (parts.length === 1) return parts[0];
  if (parts.length === 2) return parts[0] * 60 + parts[1];
  if (parts.length === 3) return parts[0] * 3600 + parts[1] * 60 + parts[2];
  return -1;
}

function formatTimestamp(seconds) {
  if (!Number.isFinite(Number(seconds))) return "00:00.000";
  const value = Math.max(0, Number(seconds));
  const minutes = Math.floor(value / 60);
  const remaining = (value % 60).toFixed(3).padStart(6, "0");
  return `${String(minutes).padStart(2, "0")}:${remaining}`;
}

function formatDuration(seconds) {
  if (!seconds || seconds < 0) return "0:00";
  const rounded = Math.round(seconds);
  const hours = Math.floor(rounded / 3600);
  const minutes = Math.floor((rounded % 3600) / 60);
  const remaining = rounded % 60;
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(remaining).padStart(2, "0")}`
    : `${minutes}:${String(remaining).padStart(2, "0")}`;
}

function formatBytes(bytes) {
  if (!bytes) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  return `${(bytes / 1024 ** index).toFixed(index ? 1 : 0)} ${units[index]}`;
}

function formatClockSince(value) {
  const seconds = Math.max(0, (Date.now() - new Date(value).getTime()) / 1000);
  return `已录制 ${formatDuration(seconds)}`;
}

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>"']/g, (character) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#039;"
  })[character]);
}

function refreshIcons() {
  if (window.lucide) {
    window.lucide.createIcons();
  }
}
