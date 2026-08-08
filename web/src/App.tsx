import { motion } from "framer-motion";
import { useEffect, useState } from "react";
import { Dropzone } from "./components/Dropzone";
import { ResultPanel } from "./components/ResultPanel";
import {
  fetchHealth,
  uploadFile,
  type S3UploadData,
  type UploadMode,
  type WPUploadData,
} from "./lib/api";

const fadeUp = {
  hidden: { opacity: 0, y: 18 },
  show: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: { delay: 0.08 * i, duration: 0.55, ease: [0.22, 1, 0.36, 1] as const },
  }),
};

export default function App() {
  const [mode, setMode] = useState<UploadMode>("s3");
  const [healthy, setHealthy] = useState<boolean | null>(null);
  const [maxSizeLabel, setMaxSizeLabel] = useState("20 MB");
  const [absoluteMaxLabel, setAbsoluteMaxLabel] = useState("200 MB");
  const [limits, setLimits] = useState({ maxSize: 20 * 1024 * 1024, absoluteMaxSize: 200 * 1024 * 1024 });
  const [busy, setBusy] = useState(false);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [result, setResult] = useState<S3UploadData | WPUploadData | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetchHealth()
      .then((res) => {
        if (cancelled) return;
        setHealthy(res.success && res.data?.status === "ok");
        if (res.data?.max_size_formatted) {
          setMaxSizeLabel(res.data.max_size_formatted);
        }
        if (res.data?.absolute_max_size_formatted) {
          setAbsoluteMaxLabel(res.data.absolute_max_size_formatted);
        }
        if (res.data?.max_size && res.data?.absolute_max_size) {
          setLimits({
            maxSize: res.data.max_size,
            absoluteMaxSize: res.data.absolute_max_size,
          });
        }
      })
      .catch(() => {
        if (!cancelled) setHealthy(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    return () => {
      if (previewUrl) URL.revokeObjectURL(previewUrl);
    };
  }, [previewUrl]);

  function reset() {
    if (previewUrl) URL.revokeObjectURL(previewUrl);
    setFile(null);
    setPreviewUrl(null);
    setResult(null);
    setError(null);
    setProgress(0);
    setBusy(false);
  }

  async function handleSelect(selected: File) {
    setError(null);
    setResult(null);
    setFile(selected);
    if (previewUrl) URL.revokeObjectURL(previewUrl);
    setPreviewUrl(selected.type.startsWith("image/") ? URL.createObjectURL(selected) : null);

    setBusy(true);
    setProgress(0);
    try {
      // Refresh CSRF cookie before mutating request
      await fetchHealth().catch(() => undefined);
      const data = await uploadFile(selected, mode, limits, (e) => setProgress(e.percent));
      setProgress(100);
      setResult(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="health-dot" data-ok={healthy === true ? "true" : healthy === false ? "false" : undefined}>
          <span />
          {healthy === null ? "Checking API…" : healthy ? "API online" : "API degraded"}
        </div>
        <span style={{ color: "var(--ink-soft)", fontSize: "0.85rem" }}>
          ≤{maxSizeLabel} direct · chunked to {absoluteMaxLabel}
        </span>
      </header>

      <section className="hero">
        <motion.h1
          className="brand"
          custom={0}
          variants={fadeUp}
          initial="hidden"
          animate="show"
        >
          Lum<em>en</em>
        </motion.h1>
        <motion.p
          className="headline"
          custom={1}
          variants={fadeUp}
          initial="hidden"
          animate="show"
        >
          Upload. Resize. Ship to the edge.
        </motion.p>
        <motion.p
          className="lede"
          custom={2}
          variants={fadeUp}
          initial="hidden"
          animate="show"
        >
          A focused media pipeline for S3 delivery and WordPress-sized derivatives — built for speed, not dashboards.
        </motion.p>

        <motion.div
          className="cta-row"
          custom={3}
          variants={fadeUp}
          initial="hidden"
          animate="show"
        >
          <div className="mode-switch" role="group" aria-label="Upload mode">
            <button
              type="button"
              data-active={mode === "s3"}
              onClick={() => setMode("s3")}
              disabled={busy}
            >
              S3 upload
            </button>
            <button
              type="button"
              data-active={mode === "wp"}
              onClick={() => setMode("wp")}
              disabled={busy}
            >
              WP resize
            </button>
          </div>
          <button
            type="button"
            className="btn btn-primary"
            disabled={busy}
            onClick={() => document.querySelector<HTMLInputElement>(".file-input")?.click()}
          >
            Choose file
          </button>
        </motion.div>

        <Dropzone busy={busy} progress={progress} onSelect={handleSelect} />
      </section>

      {error && (
        <div className="error-banner">
          <p>{error}</p>
        </div>
      )}

      <ResultPanel
        mode={mode}
        previewUrl={previewUrl}
        file={file}
        result={result}
        onReset={reset}
      />

      <p className="footer-note">Lumen · powered by s3-upload-api</p>
    </div>
  );
}
