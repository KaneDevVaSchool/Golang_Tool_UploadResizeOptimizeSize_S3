import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useMemo, useRef, useState } from "react";
import { ConfirmDialog } from "./components/ConfirmDialog";
import { Dropzone } from "./components/Dropzone";
import { PreviewPanel, type PreviewItem } from "./components/PreviewPanel";
import { ResultPanel, type ResultItem } from "./components/ResultPanel";
import { StepTimeline, type TimelineStep } from "./components/StepTimeline";
import {
  abortActiveUpload,
  fetchHealth,
  UploadAbortedError,
  uploadFile,
  validateClientFile,
  type UploadMode,
  type UploadResult,
} from "./lib/api";
import {
  bakeImageTransform,
  DEFAULT_TRANSFORM,
  type ImageTransform,
} from "./lib/imageTransform";

const fadeUp = {
  hidden: { opacity: 0, y: 18 },
  show: (i: number) => ({
    opacity: 1,
    y: 0,
    transition: { delay: 0.07 * i, duration: 0.5, ease: [0.22, 1, 0.36, 1] as const },
  }),
};

function makeId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

function toPreviewItems(files: File[]): PreviewItem[] {
  return files.map((file) => ({
    id: makeId(),
    file,
    previewUrl: file.type.startsWith("image/") ? URL.createObjectURL(file) : null,
    progress: 0,
    status: "ready" as const,
    transform: { ...DEFAULT_TRANSFORM },
  }));
}

export default function App() {
  const [mode, setMode] = useState<UploadMode>("s3");
  const [healthy, setHealthy] = useState<boolean | null>(null);
  const [maxSizeLabel, setMaxSizeLabel] = useState("20 MB");
  const [absoluteMaxLabel, setAbsoluteMaxLabel] = useState("200 MB");
  const [limits, setLimits] = useState({
    maxSize: 20 * 1024 * 1024,
    absoluteMaxSize: 200 * 1024 * 1024,
    chunkUpload: true,
  });
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [items, setItems] = useState<PreviewItem[]>([]);
  const [activeId, setActiveId] = useState<string>("");
  const [results, setResults] = useState<ResultItem[] | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const itemsRef = useRef(items);
  const uploadAbortRef = useRef<AbortController | null>(null);
  itemsRef.current = items;

  useEffect(() => {
    let cancelled = false;
    fetchHealth()
      .then((res) => {
        if (cancelled) return;
        const s3Ok = res.data?.checks?.s3_service;
        const s3Status =
          typeof s3Ok === "object" && s3Ok && "status" in s3Ok
            ? (s3Ok as { status?: string }).status
            : s3Ok;
        const ready =
          Boolean(res.data?.status === "ok") && s3Status !== "error" && res.success !== false;
        setHealthy(ready);
        if (res.data?.max_size_formatted) setMaxSizeLabel(res.data.max_size_formatted);
        if (res.data?.absolute_max_size_formatted) {
          setAbsoluteMaxLabel(res.data.absolute_max_size_formatted);
        }
        if (res.data?.max_size && res.data?.absolute_max_size) {
          setLimits({
            maxSize: res.data.max_size,
            absoluteMaxSize: res.data.absolute_max_size,
            chunkUpload: res.data.chunk_upload !== false,
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
      abortActiveUpload();
      uploadAbortRef.current?.abort();
      itemsRef.current.forEach((item) => {
        if (item.previewUrl) URL.revokeObjectURL(item.previewUrl);
      });
    };
  }, []);

  const phaseDone = Boolean(results?.length) && !busy;
  const timelineStep: TimelineStep = phaseDone ? 3 : busy ? 3 : items.length ? 2 : 1;

  const showDecor = !items.length && !results;
  const fileAccept = mode === "wp" ? "image/*" : "image/*,.pdf,.zip,.txt,.csv,.json";

  function revokeAll(list: PreviewItem[]) {
    list.forEach((item) => {
      if (item.previewUrl) URL.revokeObjectURL(item.previewUrl);
    });
  }

  function clearAll() {
    if (busy) cancelUpload();
    revokeAll(items);
    setItems([]);
    setActiveId("");
    setResults(null);
    setError(null);
    setBusy(false);
    setConfirmOpen(false);
  }

  function handleSelect(files: File[]) {
    if (!files.length) return;
    setError(null);
    setResults(null);
    setConfirmOpen(false);
    const next = toPreviewItems(files);
    setItems((prev) => {
      const merged = [...prev, ...next];
      setActiveId((current) =>
        current && merged.some((i) => i.id === current) ? current : (merged[0]?.id ?? ""),
      );
      return merged;
    });
  }

  function removeItem(id: string) {
    setItems((prev) => {
      const target = prev.find((i) => i.id === id);
      if (target?.previewUrl) URL.revokeObjectURL(target.previewUrl);
      const next = prev.filter((i) => i.id !== id);
      if (activeId === id) setActiveId(next[0]?.id ?? "");
      if (!next.length) setResults(null);
      return next;
    });
  }

  function updateTransform(id: string, transform: ImageTransform) {
    setItems((prev) => prev.map((item) => (item.id === id ? { ...item, transform } : item)));
  }

  function requestSend() {
    if (!items.length || busy) return;
    const firstInvalid = items
      .map((item) => validateClientFile(item.file, mode, limits))
      .find(Boolean);
    if (firstInvalid) {
      setError(firstInvalid);
      return;
    }
    setError(null);
    setConfirmOpen(true);
  }

  function cancelUpload() {
    uploadAbortRef.current?.abort();
    abortActiveUpload();
    setBusy(false);
    setError("Đã hủy gửi ảnh.");
    setItems((prev) =>
      prev.map((item) =>
        item.status === "uploading"
          ? { ...item, status: "error", error: "Đã hủy", progress: 0 }
          : item,
      ),
    );
  }

  async function runUpload(toUpload: PreviewItem[], previous?: ResultItem[] | null) {
    const controller = new AbortController();
    uploadAbortRef.current = controller;
    setBusy(true);
    setError(null);
    setResults(null);

    const resultMap = new Map<string, ResultItem>();
    previous?.forEach((r) => {
      if (r.result) resultMap.set(r.id, r);
    });

    const orderIds = previous?.map((r) => r.id) ?? toUpload.map((t) => t.id);
    let aborted = false;

    try {
      const health = await fetchHealth(controller.signal).catch(() => undefined);
      if (health?.data?.chunk_upload === false) {
        setLimits((prev) => ({ ...prev, chunkUpload: false }));
      }

      for (const current of toUpload) {
        if (controller.signal.aborted) {
          aborted = true;
          break;
        }

        setActiveId(current.id);
        setItems((prev) =>
          prev.map((item) =>
            item.id === current.id
              ? { ...item, status: "uploading", progress: 0, error: undefined }
              : item,
          ),
        );

        try {
          const fileToSend = await bakeImageTransform(current.file, current.transform);
          if (controller.signal.aborted) throw new UploadAbortedError();

          const data: UploadResult = await uploadFile(
            fileToSend,
            mode,
            limits,
            (e) => {
              setItems((prev) =>
                prev.map((item) =>
                  item.id === current.id ? { ...item, progress: e.percent } : item,
                ),
              );
            },
            controller.signal,
          );
          setItems((prev) =>
            prev.map((item) =>
              item.id === current.id ? { ...item, status: "done", progress: 100 } : item,
            ),
          );
          const bakedPreview =
            fileToSend !== current.file ? URL.createObjectURL(fileToSend) : current.previewUrl;
          resultMap.set(current.id, {
            id: current.id,
            fileName: fileToSend.name,
            previewUrl: bakedPreview,
            result: data,
          });
        } catch (err) {
          if (err instanceof UploadAbortedError || controller.signal.aborted) {
            aborted = true;
            resultMap.set(current.id, {
              id: current.id,
              fileName: current.file.name,
              previewUrl: current.previewUrl,
              result: null,
              error: "Đã hủy",
            });
            break;
          }
          const message = err instanceof Error ? err.message : "Gửi không thành công";
          setItems((prev) =>
            prev.map((item) =>
              item.id === current.id ? { ...item, status: "error", error: message } : item,
            ),
          );
          resultMap.set(current.id, {
            id: current.id,
            fileName: current.file.name,
            previewUrl: current.previewUrl,
            result: null,
            error: message,
          });
        }
      }

      if (!aborted) {
        const ordered = orderIds.map((id) => resultMap.get(id)).filter(Boolean) as ResultItem[];
        setResults(ordered);
        if (ordered.every((c) => !c.result)) {
          setError("Không gửi được ảnh nào. Kiểm tra kết nối rồi thử lại.");
        }
      }
    } finally {
      if (uploadAbortRef.current === controller) uploadAbortRef.current = null;
      setBusy(false);
    }
  }

  async function confirmSend() {
    if (!items.length || busy) return;
    setConfirmOpen(false);
    await runUpload([...items]);
  }

  async function retryFailed() {
    if (!results || busy) return;
    const failedIds = new Set(results.filter((r) => !r.result).map((r) => r.id));
    if (!failedIds.size) return;
    const failedItems = items.filter((i) => failedIds.has(i.id));
    if (!failedItems.length) return;
    await runUpload(failedItems, results);
  }

  const confirmMessage = useMemo(() => {
    const n = items.length;
    const verb = mode === "wp" ? "thu nhỏ" : "lưu";
    if (n <= 1) {
      return `Gửi “${items[0]?.file.name ?? "ảnh"}” để ${verb}? Ảnh chỉ lên máy chủ khi bạn bấm “Có, gửi”.`;
    }
    return `Gửi ${n} ảnh để ${verb}? Ảnh chỉ lên máy chủ khi bạn bấm “Có, gửi”.`;
  }, [items, mode]);

  return (
    <>
      <div className="bg-logo" aria-hidden />
      <div className="title-atmosphere" aria-hidden />

      {showDecor && (
        <div className="mascot-scene" aria-hidden>
          <motion.div
            className="mascot mascot-b"
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.35, duration: 0.65, ease: [0.22, 1, 0.36, 1] }}
          >
            <img src="/images/vas-mascot-wave.png" alt="" />
          </motion.div>
        </div>
      )}

      <div className="app-shell">
        <header className="topbar">
          <div className="chrome-mark">
            <img src="/images/vas-white.png" alt="VA Schools" className="chrome-logo" />
            <div className="chrome-divider" aria-hidden />
            <div
              className="health-dot"
              data-ok={healthy === true ? "true" : healthy === false ? "false" : undefined}
            >
              <span className="health-dot-pulse" aria-hidden />
              <span className="health-dot-label">
                {healthy === null ? "Đang mở…" : healthy ? "Sẵn sàng" : "Chưa kết nối"}
              </span>
            </div>
          </div>
          <div className="limit-note" title={`Tối đa ${maxSizeLabel} · file lớn tới ${absoluteMaxLabel}`}>
            <span className="limit-note-kicker">Giới hạn</span>
            <span className="limit-note-value">
              {maxSizeLabel}
              <span className="limit-note-sep">/</span>
              {absoluteMaxLabel}
            </span>
          </div>
        </header>

        <div className="workspace">
          <section className="intro">
            <motion.img
              className="wordmark"
              src="/images/vas-wordmark-stacked.png"
              alt="VA Schools"
              custom={0}
              variants={fadeUp}
              initial="hidden"
              animate="show"
            />
            <motion.p className="kicker" custom={1} variants={fadeUp} initial="hidden" animate="show">
              Gửi ảnh dễ dàng
            </motion.p>
            <motion.h1 className="headline" custom={2} variants={fadeUp} initial="hidden" animate="show">
              Chọn ảnh, chỉnh rồi gửi
            </motion.h1>
            <motion.p className="lede" custom={3} variants={fadeUp} initial="hidden" animate="show">
              Kéo nhiều ảnh vào, chỉnh nhẹ nếu cần, rồi gửi khi bạn sẵn sàng.
            </motion.p>

            <StepTimeline current={timelineStep} done={phaseDone && !busy} />

            <motion.div className="cta-row" custom={5} variants={fadeUp} initial="hidden" animate="show">
              <div className="mode-switch" role="group" aria-label="Cách gửi ảnh">
                <button type="button" data-active={mode === "s3"} onClick={() => setMode("s3")} disabled={busy}>
                  Lưu ảnh
                </button>
                <button type="button" data-active={mode === "wp"} onClick={() => setMode("wp")} disabled={busy}>
                  Thu nhỏ ảnh
                </button>
              </div>
              {!items.length && !results && (
                <motion.button
                  type="button"
                  className="btn btn-primary"
                  disabled={busy}
                  whileHover={{ scale: 1.03 }}
                  whileTap={{ scale: 0.97 }}
                  onClick={() => document.querySelector<HTMLInputElement>(".file-input")?.click()}
                >
                  Chọn nhiều ảnh
                </motion.button>
              )}
            </motion.div>
          </section>

          <section className="stage">
            <AnimatePresence mode="wait">
              {results ? (
                <ResultPanel
                  key="result"
                  mode={mode}
                  items={results}
                  onReset={clearAll}
                  onRetryFailed={retryFailed}
                />
              ) : items.length ? (
                <PreviewPanel
                  key="preview"
                  items={items}
                  activeId={activeId || items[0].id}
                  busy={busy}
                  modeLabel={mode === "wp" ? "thu nhỏ" : "lưu"}
                  onSelectItem={setActiveId}
                  onRemoveItem={removeItem}
                  onAddMore={() => document.querySelector<HTMLInputElement>(".file-input")?.click()}
                  onClearAll={clearAll}
                  onRequestSend={requestSend}
                  onCancelUpload={cancelUpload}
                  onTransformChange={updateTransform}
                />
              ) : (
                <Dropzone key="drop" busy={busy} onSelect={handleSelect} accept={fileAccept} />
              )}
            </AnimatePresence>

            {items.length > 0 && !results && (
              <input
                className="file-input"
                type="file"
                multiple
                accept={fileAccept}
                onChange={(e) => {
                  handleSelect(Array.from(e.target.files ?? []));
                  e.target.value = "";
                }}
              />
            )}
          </section>
        </div>

        <ConfirmDialog
          open={confirmOpen}
          title={items.length > 1 ? `Gửi ${items.length} ảnh chứ?` : "Gửi ảnh này chứ?"}
          message={confirmMessage}
          confirmLabel="Có, gửi"
          cancelLabel="Chưa"
          busy={busy}
          onConfirm={confirmSend}
          onCancel={() => setConfirmOpen(false)}
        />

        <AnimatePresence>
          {error && (
            <motion.div
              className="error-banner"
              role="alert"
              initial={{ opacity: 0, y: 16 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 10 }}
            >
              <p>{error}</p>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </>
  );
}
