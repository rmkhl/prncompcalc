"use strict";

const storageKey = "prncompcalc.v1";
const initialRows = 6;
const features = [
  ["", "featureOuter"],
  ["x", "featureX"],
  ["y", "featureY"],
  ["hole", "featureHole"],
];
const exampleRows = [
  [20, "19.80", ""], [20, "19.83", ""],
  [30, "29.74", ""], [30, "29.76", ""],
  [40, "39.69", ""], [40, "39.71", ""],
].map(([expected, actual, feature]) => ({ expected: String(expected), actual, feature }));

const state = { language: "fi", rows: [] };
let ready = false;
let loadFailed = false;

const byId = (id) => document.getElementById(id);
const blankRow = () => ({ expected: "", actual: "", feature: "" });
const text = (key) => translations[state.language][key];

function format(template, args = []) {
  return template.replace(/\{(\d+)\}/g, (_, index) => String(args[index]));
}

// Everything in the page follows the language, except what goes into OrcaSlicer:
// those always use a point so they can be copied as they are.
function number(value, decimals) {
  const formatted = value.toFixed(decimals);
  return state.language === "fi" ? formatted.replace(".", ",") : formatted;
}

function signed(value, decimals) {
  return (value > 0 ? "+" : "") + number(value, decimals);
}

function percent(value) {
  return state.language === "fi" ? `${number(value, 3)} %` : `${number(value, 3)}%`;
}

function messageArgs(issue) {
  return (issue.args || []).map((arg) => (typeof arg === "number" && !Number.isInteger(arg) ? number(arg, 2) : arg));
}

function messageText(issue) {
  return format(text(issue.code) || issue.code, messageArgs(issue));
}

function save() {
  try {
    localStorage.setItem(storageKey, JSON.stringify(state));
  } catch {
    // Private browsing can refuse storage. The page still works without it.
  }
}

function load() {
  try {
    const saved = JSON.parse(localStorage.getItem(storageKey));
    if (saved && translations[saved.language]) {
      state.language = saved.language;
    }
    if (saved && Array.isArray(saved.rows) && saved.rows.length > 0) {
      state.rows = saved.rows.map((row) => ({ ...blankRow(), ...row }));
    }
  } catch {
    // Unreadable storage is the same as nothing saved.
  }
  if (state.rows.length === 0) {
    state.rows = Array.from({ length: initialRows }, blankRow);
  }
}

function createInput(row, field, label, index) {
  const input = document.createElement("input");
  input.type = "text";
  // A type="number" field would refuse a decimal comma in some locales.
  input.inputMode = "decimal";
  input.autocomplete = "off";
  input.value = row[field];
  input.setAttribute("aria-label", `${label} ${index + 1}`);
  input.dataset.field = field;
  input.addEventListener("input", () => {
    row[field] = input.value;
    update();
  });
  input.addEventListener("paste", (event) => pasteRows(event, index, field));
  input.addEventListener("keydown", (event) => moveOnEnter(event, index, field));
  return input;
}

function createFeatureSelect(row, index) {
  const select = document.createElement("select");
  select.setAttribute("aria-label", `${text("columnFeature")} ${index + 1}`);
  for (const [value, key] of features) {
    select.add(new Option(text(key), value, false, row.feature === value));
  }
  select.addEventListener("change", () => {
    row.feature = select.value;
    update();
  });
  return select;
}

function createRemoveButton(index) {
  const button = document.createElement("button");
  button.type = "button";
  button.textContent = "×";
  button.className = "remove";
  button.setAttribute("aria-label", format(text("removeRow"), [index + 1]));
  button.addEventListener("click", () => {
    state.rows.splice(index, 1);
    if (state.rows.length === 0) {
      state.rows.push(blankRow());
    }
    renderTable();
    update();
  });
  return button;
}

function renderTable() {
  const body = document.querySelector("#measurements tbody");
  body.replaceChildren(...state.rows.map((row, index) => {
    const tr = document.createElement("tr");
    const cells = [
      document.createTextNode(String(index + 1)),
      createInput(row, "expected", text("columnExpected"), index),
      createInput(row, "actual", text("columnActual"), index),
      createFeatureSelect(row, index),
      createRemoveButton(index),
    ];
    for (const cell of cells) {
      const td = document.createElement("td");
      td.append(cell);
      tr.append(td);
    }
    tr.firstChild.className = "row-number";
    return tr;
  }));
}

function focusCell(index, field) {
  const row = document.querySelectorAll("#measurements tbody tr")[index];
  row.querySelector(`input[data-field="${field}"]`).focus();
}

// Entering readings at the bench goes size, measured, size, measured, so Enter
// walks that path and makes a new row at the end.
function moveOnEnter(event, index, field) {
  if (event.key !== "Enter") {
    return;
  }
  event.preventDefault();
  if (field === "expected") {
    focusCell(index, "actual");
    return;
  }
  if (index === state.rows.length - 1) {
    state.rows.push(blankRow());
    renderTable();
  }
  focusCell(index + 1, "expected");
}

function featureFromLabel(label) {
  const lowered = (label || "").trim().toLowerCase();
  if (lowered === "reikä" || lowered === "reika") {
    return "hole";
  }
  return features.some(([value]) => value === lowered) ? lowered : "";
}

// Only a paste holding more than one value is taken over. A single number goes
// into the field the normal way.
function pasteRows(event, startIndex, field) {
  const pasted = event.clipboardData.getData("text").trim();
  if (!/[\t\n]/.test(pasted) && !(field === "expected" && /\s/.test(pasted))) {
    return;
  }
  event.preventDefault();
  pasted.split(/\r?\n/).forEach((line, offset) => {
    const [expected = "", actual = "", label] = line.trim().split(/\s+/);
    const index = startIndex + offset;
    if (index >= state.rows.length) {
      state.rows.push(blankRow());
    }
    state.rows[index] = { expected, actual, feature: featureFromLabel(label) };
  });
  renderTable();
  update();
}

function fillList(list, messages) {
  list.replaceChildren(...messages.map((message) => {
    const item = document.createElement("li");
    item.textContent = message;
    return item;
  }));
}

function detail(term, description) {
  const dt = document.createElement("dt");
  dt.textContent = term;
  const dd = document.createElement("dd");
  dd.textContent = description;
  return [dt, dd];
}

function renderDetails(result) {
  const entries = [
    detail(text("detailPoints"), format(text("detailPointsValue"), [
      result.pointsUsed, result.sizes, number(result.minSize, 0), number(result.maxSize, 0),
    ])),
    detail(text("detailRange"), `${percent(result.shrinkageRange.low)} – ${percent(result.shrinkageRange.high)}`),
    detail(text("detailOffset"), `${signed(result.offset, 4)} mm`),
    detail(text("detailOffsetRange"), `${signed(result.offsetRange.low, 4)} – ${signed(result.offsetRange.high, 4)} mm`),
    detail(text("detailScatter"), `${number(result.scatter, 4)} mm`),
    detail(text("detailScale"), percent(result.modelScale)),
  ];
  for (const axis of result.axes) {
    entries.push(detail(
      format(text("detailAxis"), [axis.feature.toUpperCase()]),
      format(text("detailAxisValue"), [number(axis.shrinkage, 3), signed(axis.offset, 4)]),
    ));
  }
  if (result.axes.length > 0) {
    entries.push(detail("", text("detailAxisNote")));
  }
  byId("details").replaceChildren(...entries.flat());
}

function renderResult(result) {
  byId("shrinkage").textContent = `${result.shrinkage.toFixed(3)} %`;
  byId("contour").textContent = `${result.contourComp.toFixed(3)} mm`;
  byId("hole").textContent = result.holeComp === null ? text("holeNotMeasured") : `${result.holeComp.toFixed(3)} mm`;
  byId("hole").classList.toggle("unavailable", result.holeComp === null);

  fillList(byId("warnings"), [
    ...result.outliers.map((outlier) => format(text("outlier"), [outlier.row, signed(outlier.residual, 2)])),
    ...result.warnings.map(messageText),
  ]);
  renderDetails(result);
}

function markRows(rowIssues, outliers) {
  const flagged = new Set([...rowIssues.map((issue) => issue.row), ...outliers.map((outlier) => outlier.row)]);
  document.querySelectorAll("#measurements tbody tr").forEach((tr, index) => {
    tr.classList.toggle("flagged", flagged.has(index + 1));
  });
}

// A row with one field still empty is being typed, not wrong.
function isHalfTyped(issue) {
  return issue.code === "not_a_number" && issue.args[0] === "";
}

function showStatus(message) {
  byId("status").textContent = message;
  byId("status").hidden = message === "";
  byId("results").hidden = message !== "";
}

function calculate() {
  const request = state.rows.map((row, index) => ({
    row: index + 1, expected: row.expected, actual: row.actual, feature: row.feature,
  }));
  return JSON.parse(prncompcalcCalculate(JSON.stringify(request)));
}

function update() {
  save();
  if (loadFailed) {
    showStatus(text("loadFailed"));
    return;
  }
  if (!ready) {
    showStatus(text("loading"));
    return;
  }
  const response = calculate();
  const rowIssues = response.rowIssues.filter((issue) => !isHalfTyped(issue));
  fillList(byId("row-issues"), rowIssues.map((issue) => format(text("rowIssue"), [issue.row, messageText(issue)])));
  markRows(rowIssues, response.result ? response.result.outliers : []);

  if (response.result) {
    showStatus("");
    renderResult(response.result);
  } else if (response.error) {
    showStatus(messageText(response.error));
  } else {
    showStatus(text("needInput"));
  }
}

function renderLanguage() {
  document.documentElement.lang = state.language;
  for (const element of document.querySelectorAll("[data-i18n]")) {
    element.textContent = text(element.dataset.i18n);
  }
  for (const button of document.querySelectorAll("[data-language]")) {
    button.setAttribute("aria-pressed", String(button.dataset.language === state.language));
  }
  renderTable();
  update();
}

async function startCalculator() {
  const go = new Go();
  const source = await fetch("prncompcalc.wasm");
  if (!source.ok) {
    throw new Error(`prncompcalc.wasm: HTTP ${source.status}`);
  }
  // Not instantiateStreaming: it refuses a server that does not send the
  // application/wasm type, and how the lab server is set up is not ours to rely on.
  const { instance } = await WebAssembly.instantiate(await source.arrayBuffer(), go.importObject);
  go.run(instance);
}

function bindControls() {
  for (const button of document.querySelectorAll("[data-language]")) {
    button.addEventListener("click", () => {
      state.language = button.dataset.language;
      renderLanguage();
    });
  }
  byId("add-row").addEventListener("click", () => {
    state.rows.push(blankRow());
    renderTable();
    focusCell(state.rows.length - 1, "expected");
    update();
  });
  byId("load-example").addEventListener("click", () => {
    state.rows = exampleRows.map((row) => ({ ...row }));
    renderTable();
    update();
  });
  byId("clear").addEventListener("click", () => {
    if (confirm(text("clearConfirm"))) {
      state.rows = Array.from({ length: initialRows }, blankRow);
      renderTable();
      update();
    }
  });
}

load();
bindControls();
renderLanguage();
startCalculator().then(() => {
  ready = true;
  byId("version").textContent = prncompcalcVersion;
  update();
}).catch((error) => {
  console.error(error);
  loadFailed = true;
  update();
});
