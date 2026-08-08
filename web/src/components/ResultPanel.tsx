import { AnimatePresence, motion } from "framer-motion";
import { useState } from "react";
import {
  formatBytes,
  type S3UploadData,
  type UploadMode,
  type WPUploadData,
} from "../lib/api";

type ResultPanelProps = {
  mode: UploadMode;
  previewUrl: string | null;
  file: File | null;
  result: S3UploadData | WPUploadData | null;
  onReset: () => void;
};

function isWP(data: S3UploadData | WPUploadData): data is WPUploadData {
  return "file" in data && "sizes" in data;
}

export function ResultPanel({ mode, previewUrl, file, result, onReset }: ResultPanelProps) {
  const [copied, setCopied] = useState(false);

  if (!result || !file) return null;

  const url = isWP(result) ? result.file.url : result.url;
  const name = isWP(result) ? result.file.name : result.name;
  const size = isWP(result) ? result.file.size : result.size;

  async function copyUrl() {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      setCopied(false);
    }
  }

  return (
    <AnimatePresence>
      <motion.section
        className="result"
        initial={{ opacity: 0, y: 28 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: 12 }}
        transition={{ duration: 0.55, ease: [0.22, 1, 0.36, 1] }}
      >
        <div className="result-panel">
          <motion.div
            className="preview"
            initial={{ opacity: 0, scale: 0.97 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 0.1, duration: 0.5 }}
          >
            {previewUrl ? (
              <img src={previewUrl} alt={name} />
            ) : (
              <p>Upload complete — preview unavailable for this type.</p>
            )}
          </motion.div>

          <div className="meta">
            <h3>{mode === "wp" ? "Resized & ready" : "Landed on S3"}</h3>
            <div className="meta-row">
              <label>File</label>
              <code>{name}</code>
            </div>
            <div className="meta-row">
              <label>Size</label>
              <code>{formatBytes(size)}</code>
            </div>
            {!isWP(result) && (
              <div className="meta-row">
                <label>Key</label>
                <code>{result.key}</code>
              </div>
            )}
            <div className="meta-row">
              <label>URL</label>
              <a href={url} target="_blank" rel="noreferrer">
                {url}
              </a>
            </div>
            <div className="meta-actions">
              <button type="button" className="btn btn-primary" onClick={copyUrl}>
                {copied ? "Copied" : "Copy URL"}
              </button>
              <button type="button" className="btn btn-ghost" onClick={onReset}>
                Upload another
              </button>
            </div>

            {isWP(result) && result.sizes?.length > 0 && (
              <div className="sizes">
                {result.sizes.map((s, i) => (
                  <motion.a
                    key={`${s.name}-${s.url}`}
                    className="size-item"
                    href={s.url}
                    target="_blank"
                    rel="noreferrer"
                    initial={{ opacity: 0, y: 12 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: 0.15 + i * 0.05 }}
                  >
                    <strong>{s.name}</strong>
                    <span>
                      {s.width}×{s.height || "auto"}
                    </span>
                  </motion.a>
                ))}
              </div>
            )}
          </div>
        </div>
      </motion.section>
    </AnimatePresence>
  );
}
