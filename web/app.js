const form = document.getElementById("analyze-form");
const urlInput = document.getElementById("url-input");
const statusEl = document.getElementById("status");
const resultBody = document.getElementById("result-body");

function clearRows() {
  resultBody.innerHTML = "";
}

function addRow(field, value) {
  const tr = document.createElement("tr");
  const tdField = document.createElement("th");
  const tdValue = document.createElement("td");
  tdField.textContent = String(field);
  tdValue.textContent = String(value);
  tr.appendChild(tdField);
  tr.appendChild(tdValue);
  resultBody.appendChild(tr);
}

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  clearRows();

  const input = urlInput.value.trim();
  addRow("url", input || "(empty)");
});
