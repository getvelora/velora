import React from 'react';
import { createRoot } from 'react-dom/client';
import { Activity, Database, Film, HardDrive, Play, Server } from 'lucide-react';
import './styles.css';

type HealthResponse = {
  status: string;
  database: string;
  databaseDriver: string;
};

const DRIVER_LABELS: Record<string, string> = {
  sqlite: 'SQLite',
  postgres: 'Postgres',
};

const databaseLabel = (driver: string | undefined) =>
  (driver && DRIVER_LABELS[driver]) ?? 'Database';

function App() {
  const [health, setHealth] = React.useState<HealthResponse | null>(null);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    fetch('/api/health')
      .then((response) => {
        if (!response.ok) {
          throw new Error(`Health check returned ${response.status}`);
        }
        return response.json() as Promise<HealthResponse>;
      })
      .then(setHealth)
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : 'Health check failed');
      });
  }, []);

  return (
    <main className="app-shell">
      <aside className="sidebar" aria-label="Velora navigation">
        <div className="brand">
          <div className="brand-mark">V</div>
          <div>
            <strong>Velora</strong>
            <span>Local media</span>
          </div>
        </div>

        <nav className="nav-list">
          <a className="nav-item active" href="/">
            <Film size={18} />
            Library
          </a>
          <a className="nav-item" href="/">
            <Play size={18} />
            Playback
          </a>
          <a className="nav-item" href="/">
            <HardDrive size={18} />
            Cache
          </a>
          <a className="nav-item" href="/">
            <Server size={18} />
            Server
          </a>
        </nav>
      </aside>

      <section className="content">
        <header className="topbar">
          <div>
            <h1>Media Library</h1>
            <p>Docker-local shell ready for scanning, playback, and cache controls.</p>
          </div>

          <div className={`status-pill ${health?.status === 'ok' ? 'online' : ''}`}>
            <Activity size={16} />
            {health?.status ?? (error ? 'offline' : 'checking')}
          </div>
        </header>

        <section className="summary-grid" aria-label="Velora system summary">
          <article className="summary-tile">
            <Film size={22} />
            <span>Library</span>
            <strong>Awaiting scan</strong>
          </article>
          <article className="summary-tile">
            <Database size={22} />
            <span>{databaseLabel(health?.databaseDriver)}</span>
            <strong>{health?.database ?? (error ? 'Unavailable' : 'Checking')}</strong>
          </article>
          <article className="summary-tile">
            <HardDrive size={22} />
            <span>Cache</span>
            <strong>/cache</strong>
          </article>
        </section>

        <section className="empty-state">
          <div>
            <h2>No media scanned yet</h2>
            <p>
              Mount sample files into <code>dev/media</code>, register a library, and scan it
              through the API. Media browsing will arrive with movie and series grouping.
            </p>
          </div>
        </section>
      </section>
    </main>
  );
}

createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
