import { useEffect, useState } from "react";

const API = import.meta.env.VITE_API_URL || "/api/v2";

// Polls the Go API so the UI refreshes when new matches are ingested.
function useApi(path, intervalMs = 15000) {
  const [data, setData] = useState(null);
  const [error, setError] = useState(null);
  useEffect(() => {
    let live = true;
    const load = () =>
      fetch(API + path)
        .then((r) => (r.ok ? r.json() : Promise.reject(new Error(r.status))))
        .then((j) => live && (setData(j), setError(null)))
        .catch((e) => live && setError(e.message));
    load();
    const t = setInterval(load, intervalMs);
    return () => { live = false; clearInterval(t); };
  }, [path, intervalMs]);
  return { data, error };
}

function SplitCards({ season }) {
  const { data, error } = useApi(`/summary?season=${season}`);
  if (error) return <p role="alert">API error: {error}</p>;
  if (!data) return <p>Loading...</p>;
  return (
    <div className="cards">
      {["home", "away"].map((v) => data[v] && (
        <section key={v} className="card" aria-label={`${v} summary`}>
          <h3>{v}</h3>
          <b>{data[v].wins}-{data[v].draws}-{data[v].losses}</b>
          <span>{data[v].ppg.toFixed(2)} PPG · {data[v].gf} GF · {data[v].ga} GA</span>
        </section>
      ))}
    </div>
  );
}

function Momentum({ season }) {
  const { data } = useApi(`/matches?season=${season}`);
  if (!data?.length) return null;
  let gd = 0;
  const pts = data.map((m) => (gd += m.gf - m.ga));
  const lo = Math.min(0, ...pts), hi = Math.max(1, ...pts);
  const x = (i) => 20 + (i * 600) / Math.max(1, pts.length - 1);
  const y = (v) => 180 - ((v - lo) / (hi - lo)) * 160;
  return (
    <svg viewBox="0 0 640 200" role="img" aria-label="Cumulative goal difference">
      <path d={pts.map((v, i) => `${i ? "L" : "M"}${x(i)},${y(v)}`).join("")} fill="none" stroke="currentColor" strokeWidth="3" />
      {pts.map((v, i) => <circle key={i} cx={x(i)} cy={y(v)} r="5"><title>{data[i].opponent}: {v}</title></circle>)}
    </svg>
  );
}

export default function App() {
  const [season, setSeason] = useState(2025);
  return (
    <main>
      <h1>NCAA Women's Soccer Analytics</h1>
      <nav>{[2024, 2025, 2026].map((s) => (
        <button key={s} aria-pressed={s === season} onClick={() => setSeason(s)}>{s}</button>
      ))}</nav>
      <SplitCards season={season} />
      <h2>Goal-difference momentum</h2>
      <Momentum season={season} />
    </main>
  );
}
