import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";

const root = document.getElementById("root");

function showBootError(error: unknown) {
  const message = error instanceof Error ? error.message : String(error);
  document.body.innerHTML = `<pre style="white-space:pre-wrap;padding:16px;color:#be123c;font:14px/1.5 ui-monospace,Consolas,monospace">imv GUI boot error\n${escapeHTML(message)}</pre>`;
}

window.addEventListener("error", (event) => showBootError(event.error ?? event.message));
window.addEventListener("unhandledrejection", (event) => showBootError(event.reason));

try {
  if (!root) {
    throw new Error("Root element not found");
  }
  ReactDOM.createRoot(root).render(
    <React.StrictMode>
      <App />
    </React.StrictMode>
  );
} catch (error) {
  showBootError(error);
}

function escapeHTML(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}
