const form = document.getElementById("analyze-form");
const urlInput = document.getElementById("url-input");
const statusEl = document.getElementById("status");
const reportSection = document.getElementById("report-section");
const headingCountsList = document.getElementById("heading-counts-list");

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

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  resetReport();
  statusEl.textContent = "Analyzing...";

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
      let body = await response.json();
      throw new Error(`Request failed: ${response.status}, ${body.error.message}`);
    }

    const data = await response.json();

    setText("report-html-version", String(data.htmlVersion ?? "-"));
    setText("report-page-title", String(data.pageTitle ?? "-"));
    setText("report-is-login-page", String(Boolean(data.containsLogin)));
    setText("report-external-links", String(data.externalLinks ?? 0));
    setText("report-internal-links", String(data.internalLinks ?? 0));
    setText("report-inaccessible-links", String(data.inaccessibleLinks ?? 0));
    renderHeadingCounts(data.headingCounts);

    reportSection.classList.remove("d-none");
    statusEl.textContent = "Analysis complete.";
  } catch (error) {
    statusEl.textContent = `Failed to analyze URL: ${error.message}`;
  }
});
