const form = document.getElementById("analyze-form");
const urlInput = document.getElementById("url-input");
const statusEl = document.getElementById("status");
const reportSection = document.getElementById("report-section");
const headingCountsList = document.getElementById("heading-counts-list");

const POLL_INTERVAL_MS = 1000; // 1 second
const POLL_MAX_ATTEMPTS = 90; // 90 seconds

function setText(id, value) {
  const el = document.getElementById(id);
  el.textContent = value;
}

function resetReport() {
  reportSection.classList.add("d-none");
  headingCountsList.innerHTML = "";
  setText("report-html-version", "-");
  setText("report-page-title", "-");
  setText("report-is-login-page", "-");
  setText("report-external-links", "-");
  setText("report-internal-links", "-");
  setText("report-inaccessible-links", "-");
}

function renderHeadingCounts(headingCounts) {
  const entries =
    headingCounts && typeof headingCounts === "object"
      ? Object.entries(headingCounts)
      : [];

  if (entries.length === 0) {
    const item = document.createElement("li");
    item.className = "list-group-item";
    item.textContent = "No heading data available.";
    headingCountsList.appendChild(item);
    return;
  }

  for (const [heading, count] of entries) {
    const item = document.createElement("li");
    item.className = "list-group-item d-flex justify-content-between";
    item.innerHTML = `<span>${String(heading)}</span><strong>${String(count)}</strong>`;
    headingCountsList.appendChild(item);
  }
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function statusLabel(status) {
  switch (status) {
    case "queued":
      return "Queued…";
    case "running":
      return "Analyzing…";
    case "completed":
      return "Analysis complete.";
    case "failed":
      return "Analysis failed.";
    default:
      return "Waiting…";
  }
}

async function readErrorMessage(response) {
  try {
    const body = await response.json();
    if (body && body.error && typeof body.error.message === "string") {
      return body.error.message;
    }
  } catch {
    /* ignore */
  }
  return response.statusText || "Request failed";
}

/**
 * Polls GET /api/v1/jobs/{jobId} until completed or failed.
 * @returns {Promise<{ status: string, result?: object }>}
 */
async function pollJobUntilDone(jobId, onStatus) {
  const path = `/api/v1/jobs/${encodeURIComponent(jobId)}`;
  for (let attempt = 0; attempt < POLL_MAX_ATTEMPTS; attempt++) {
    const res = await fetch(path);
    if (!res.ok) {
      throw new Error(await readErrorMessage(res));
    }
    const body = await res.json();
    const st = body.status;
    if (typeof onStatus === "function") {
      onStatus(st);
    }
    if (st === "completed") {
      return body;
    }
    if (st === "failed") {
      const err = body.error;
      const msg = err && err.message ? err.message : "Analysis failed";
      const code = err && err.code ? ` (${err.code})` : "";
      throw new Error(`${msg}${code}`);
    }
    await sleep(POLL_INTERVAL_MS);
  }
  throw new Error("Timed out waiting for analysis. Try again later.");
}

function renderAnalyzeResult(data) {
  setText("report-html-version", String(data.htmlVersion ?? "-"));
  setText("report-page-title", String(data.pageTitle ?? "-"));
  setText("report-is-login-page", String(Boolean(data.containsLogin)));
  setText("report-external-links", String(data.externalLinks ?? 0));
  setText("report-internal-links", String(data.internalLinks ?? 0));
  setText("report-inaccessible-links", String(data.inaccessibleLinks ?? 0));
  renderHeadingCounts(data.headingCounts);
  reportSection.classList.remove("d-none");
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  resetReport();
  statusEl.textContent = "Submitting…";

  const input = urlInput.value.trim();

  try {
    const response = await fetch("/api/v1/links/analyze", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ url: input }),
    });

    if (!response.ok) {
      throw new Error(await readErrorMessage(response));
    }

    if (response.status !== 202) {
      throw new Error(`Unexpected response (${response.status}). Expected 202 Accepted.`);
    }

    const accepted = await response.json();
    const jobId = accepted && accepted.jobId;
    if (!jobId || typeof jobId !== "string") {
      throw new Error("Server did not return a job id.");
    }

    const final = await pollJobUntilDone(jobId, (st) => {
      statusEl.textContent = statusLabel(st);
    });

    if (!final.result || typeof final.result !== "object") {
      throw new Error("Analysis finished but no result was returned.");
    }

    renderAnalyzeResult(final.result);
    statusEl.textContent = statusLabel("completed");
  } catch (error) {
    statusEl.textContent = error.message;
  }
});
