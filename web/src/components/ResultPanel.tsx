import { motion } from "framer-motion";
import { useState } from "react";
import {
  formatBytes,
  isWPResult,
  resultName,
  resultSize,
  resultUrl,
  type UploadMode,
  type UploadResult,
} from "../lib/api";

export type ResultItem = {
  id: string;
  fileName: string;
  previewUrl: string | null;
  result: UploadResult | null;
  error?: string;
};

type ResultPanelProps = {
  mode: UploadMode;
  items: ResultItem[];
  onReset: () => void;
  onRetryFailed?: () => void;
};

export function ResultPanel({ mode, items, onReset, onRetryFailed }: ResultPanelProps) {
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [copiedAll, setCopiedAll] = useState(false);
  const okCount = items.filter((i) => i.result).length;
  const failCount = items.length - okCount;
  const urls = items.filter((i) => i.result).map((i) => resultUrl(i.result!));

  async function copyUrl(id: string, url: string) {
    try {
      await navigator.clipboard.writeText(url);
      setCopiedId(id);
      window.setTimeout(() => setCopiedId(null), 1600);
    } catch {
      setCopiedId(null);
    }
  }

  async function copyAll() {
    if (!urls.length) return;
    try {
      await navigator.clipboard.writeText(urls.join("\n"));
      setCopiedAll(true);
      window.setTimeout(() => setCopiedAll(false), 1600);
    } catch {
      setCopiedAll(false);
    }
  }

  return (
    <motion.section
      className="result-inline"
      initial={{ opacity: 0, y: 16, scale: 0.98 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: -12 }}
      transition={{ duration: 0.45, ease: [0.22, 1, 0.36, 1] }}
    >
      <div className="result-header">
        <div className="meta-head">
          <motion.img
            className="mascot-mini"
            src="/images/vas-mascot-wave.png"
            alt=""
            aria-hidden
            animate={{ y: [0, -6, 0], rotate: [-3, 3, -3] }}
            transition={{ duration: 2.8, repeat: Infinity, ease: "easeInOut" }}
          />
          <div>
            <h3>
              {failCount === 0
                ? mode === "wp"
                  ? `Đã thu nhỏ ${okCount} ảnh!`
                  : `Đã lưu ${okCount} ảnh!`
                : `Xong ${okCount}/${items.length} ảnh`}
            </h3>
            <p className="result-sub">
              {failCount > 0 ? `${failCount} ảnh lỗi — thử gửi lại bên dưới.` : "Chép đường dẫn khi cần dùng."}
            </p>
          </div>
        </div>
        <div className="result-header-actions">
          {urls.length > 1 && (
            <button type="button" className="btn btn-primary" onClick={copyAll}>
              {copiedAll ? "Đã chép hết" : "Chép tất cả"}
            </button>
          )}
          {failCount > 0 && onRetryFailed && (
            <button type="button" className="btn btn-primary" onClick={onRetryFailed}>
              Gửi lại ảnh lỗi
            </button>
          )}
          <button type="button" className="btn btn-ghost" onClick={onReset}>
            Gửi ảnh khác
          </button>
        </div>
      </div>

      <div className="result-list">
        {items.map((item, i) => {
          const url = item.result ? resultUrl(item.result) : null;
          return (
            <motion.article
              key={item.id}
              className="result-card"
              data-ok={item.result ? "true" : "false"}
              initial={{ opacity: 0, y: 14 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.06 + i * 0.05, duration: 0.35 }}
            >
              <div className="result-card-media">
                {item.previewUrl ? (
                  <img src={item.previewUrl} alt={item.fileName} />
                ) : (
                  <span>{item.fileName.slice(0, 1)}</span>
                )}
              </div>
              <div className="result-card-body">
                <strong title={item.result ? resultName(item.result) : item.fileName}>
                  {item.result ? resultName(item.result) : item.fileName}
                </strong>
                {item.result ? (
                  <>
                    <span>{formatBytes(resultSize(item.result))}</span>
                    <a href={url!} target="_blank" rel="noreferrer">
                      {url}
                    </a>
                    {isWPResult(item.result) && item.result.sizes?.length > 0 && (
                      <div className="sizes">
                        {item.result.sizes.map((s) => (
                          <a
                            key={`${s.name}-${s.url}`}
                            className="size-item"
                            href={s.url}
                            target="_blank"
                            rel="noreferrer"
                          >
                            <strong>{s.name}</strong>
                            <span>
                              {s.width}×{s.height || "tự động"}
                            </span>
                          </a>
                        ))}
                      </div>
                    )}
                    <div className="meta-actions">
                      <button type="button" className="btn btn-primary" onClick={() => copyUrl(item.id, url!)}>
                        {copiedId === item.id ? "Đã chép" : "Chép đường dẫn"}
                      </button>
                    </div>
                  </>
                ) : (
                  <span className="result-error">{item.error || "Gửi không thành công"}</span>
                )}
              </div>
            </motion.article>
          );
        })}
      </div>
    </motion.section>
  );
}
