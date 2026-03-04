import { useEffect, useState } from "react";

const API_BASE = import.meta.env.VITE_API_BASE ?? "/api";

export default function App() {
  const [health, setHealth] = useState("loading");
  const [metricsPreview, setMetricsPreview] = useState("loading");

  useEffect(() => {
    async function load() {
      try {
        const healthResponse = await fetch("/actuator/health");
        const healthData = await healthResponse.json();
        setHealth(healthData.status ?? "unknown");
      } catch {
        setHealth("unreachable");
      }

      try {
        const metricsResponse = await fetch("/actuator/prometheus");
        const text = await metricsResponse.text();
        setMetricsPreview(text.split("\n").slice(0, 20).join("\n"));
      } catch {
        setMetricsPreview("metrics unavailable");
      }
    }

    void load();
  }, []);

  return (
    <main className="admin">
      <header>
        <h1>gnusocial-next admin</h1>
        <p>Moderation and observability console (milestone scaffold)</p>
      </header>

      <section className="card">
        <h2>Platform Health</h2>
        <p>
          API base: <code>{API_BASE}</code>
        </p>
        <p>
          Health status: <strong>{health}</strong>
        </p>
      </section>

      <section className="card">
        <h2>Prometheus Preview</h2>
        <pre>{metricsPreview}</pre>
      </section>

      <section className="card">
        <h2>Moderation Operations</h2>
        <ul>
          <li>Hide post: available from main user UI</li>
          <li>Mute user: available from main user UI</li>
          <li>Follow/unfollow: available from main user UI</li>
          <li>Dedicated admin workflows: next milestone</li>
        </ul>
      </section>
    </main>
  );
}

