import { AnimatePresence, LayoutGroup, motion } from "framer-motion";
import { useEffect, useMemo, useRef, useState } from "react";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { Dropzone } from "../components/Dropzone";
import { PreviewPanel, type PreviewItem } from "../components/PreviewPanel";
import { ResultPanel, type ResultItem } from "../components/ResultPanel";
import { Onboarding, OnboardingTrigger, readOnboardingDone } from "../components/Onboarding";
import { StepTimeline, type TimelineStep } from "../components/StepTimeline";
import {
  abortActiveUpload,
  fetchHealth,
  UploadAbortedError,
  uploadFile,
  validateClientFile,
  type UploadMode,
  type UploadResult,
} from "../lib/api";
import {
  bakeImageTransform,
  DEFAULT_TRANSFORM,
  type ImageTransform,
} from "../lib/imageTransform";
import { createThumbObjectURL, revokeObjectUrl } from "../lib/previewImage";

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
    thumbUrl: null,
    progress: 0,
    status: "ready" as const,
    transform: { ...DEFAULT_TRANSFORM },
  }));
}

function revokePreviewItem(item: PreviewItem) {
  revokeObjectUrl(item.previewUrl);
  if (item.thumbUrl && item.thumbUrl !== item.previewUrl) {
    revokeObjectUrl(item.thumbUrl);
  }
}

function revokeResultPreviews(list: ResultItem[] | null | undefined, keep?: Set<string | null>) {
  list?.forEach((item) => {
    if (!item.previewUrl) return;
    if (keep?.has(item.previewUrl)) return;
    revokeObjectUrl(item.previewUrl);
  });
}

function IconSave() {
  return (
    <svg className="mode-switch-icon" viewBox="0 0 20 20" fill="none" aria-hidden>
      <path
        d="M4 14.5V5.8A1.8 1.8 0 015.8 4h6.4L16 7.8v6.7A1.8 1.8 0 0114.2 16H5.8A1.8 1.8 0 014 14.2z"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinejoin="round"
      />
      <path d="M7 4v4h5V4" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M7 12.5h6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </svg>
  );
}

function IconShrink() {
  return (
    <svg className="mode-switch-icon" viewBox="0 0 20 20" fill="none" aria-hidden>
      <rect x="3.5" y="3.5" width="13" height="13" rx="2" stroke="currentColor" strokeWidth="1.5" />
      <path
        d="M7 13V9.5M7 13h3.5M7 13l6-6"
        stroke="currentColor"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  );
}

export default function UploadTool() {
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
  const [onboardingOpen, setOnboardingOpen] = useState(() => !readOnboardingDone());
  const [mascotHint, setMascotHint] = useState(false);
  const [narrow, setNarrow] = useState(false);

  const itemsRef = useRef(items);
  const resultsRef = useRef(results);

  useEffect(() => {
    const mq = window.matchMedia("(max-width: 860px)");
    const sync = () => setNarrow(mq.matches);
    sync();
    mq.addEventListener("change", sync);
    return () => mq.removeEventListener("change", sync);
  }, []);
  const uploadAbortRef = useRef<AbortController | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const addMoreInputRef = useRef<HTMLInputElement>(null);
  itemsRef.current = items;
  resultsRef.current = results;

  const modeLabel = mode === "wp" ? "thu nhỏ" : "lưu";
  const modeVerbCap = mode === "wp" ? "Thu nhỏ" : "Lưu";

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
      const keep = new Set(itemsRef.current.map((i) => i.previewUrl));
      itemsRef.current.forEach(revokePreviewItem);
      revokeResultPreviews(resultsRef.current, keep);
    };
  }, []);

  const phaseDone = Boolean(results?.length) && !busy;
  const timelineStep: TimelineStep = phaseDone ? 3 : busy ? 3 : items.length ? 2 : 1;

  const showDecor = !items.length && !results;
  const fileAccept = mode === "wp" ? "image/*" : "image/*,.pdf,.zip,.txt,.csv,.json";

  const homeLede =
    "Kho ảnh chung của Trường Việt Mỹ — lưu tập trung, tối ưu dung lượng và lấy link dùng ngay trên các hệ thống nội bộ.";

  const lede =
    mode === "wp"
      ? "Chọn ảnh — hệ thống thu nhỏ, tối ưu và tạo nhiều kích thước. Lấy link dùng ngay."
      : "Chọn ảnh hoặc tài liệu — lưu tập trung, hỗ trợ file lớn. Lấy link dùng ngay khi cần.";

  const modeHint =
    mode === "wp"
      ? `Chỉ ảnh · tối đa ${maxSizeLabel} · thu nhỏ + nhiều kích thước`
      : `Ảnh & tài liệu · tới ${absoluteMaxLabel} · lưu nguyên bản`;

  const valueAnchors = [
    {
      title: "Tập trung",
      body: "Một kho dùng chung — giảm gửi file rời, dễ quản lý và đồng bộ toàn trường.",
    },
    {
      title: "Tối ưu",
      body: "Thu nhỏ & nén thông minh — ảnh nhẹ hơn, trang và ứng dụng tải nhanh hơn.",
    },
    {
      title: "Liên thông",
      body: "Mỗi ảnh một đường dẫn riêng",
    },
  ] as const;

  function revokeAll(list: PreviewItem[]) {
    list.forEach(revokePreviewItem);
  }

  function clearAll() {
    if (busy) cancelUpload();
    const keep = new Set(items.map((i) => i.previewUrl));
    revokeResultPreviews(results, keep);
    revokeAll(items);
    setItems([]);
    setActiveId("");
    setResults(null);
    setError(null);
    setBusy(false);
    setConfirmOpen(false);
  }

  function attachThumbs(batch: PreviewItem[]) {
    batch.forEach((item) => {
      if (!item.previewUrl) return;
      void createThumbObjectURL(item.file).then((thumbUrl) => {
        setItems((prev) => {
          const stillThere = prev.some((i) => i.id === item.id);
          if (!stillThere) {
            revokeObjectUrl(thumbUrl);
            return prev;
          }
          return prev.map((i) => {
            if (i.id !== item.id) return i;
            // Fall back to full preview URL if downscale fails (e.g. odd formats)
            const nextThumb = thumbUrl ?? i.previewUrl;
            if (i.thumbUrl && i.thumbUrl !== nextThumb && i.thumbUrl !== i.previewUrl) {
              revokeObjectUrl(i.thumbUrl);
            }
            return { ...i, thumbUrl: nextThumb };
          });
        });
      });
    });
  }

  function openFilePicker() {
    if (busy) return;
    if (items.length > 0 && !results) {
      addMoreInputRef.current?.click();
      return;
    }
    fileInputRef.current?.click();
  }

  function handleSelect(files: File[]) {
    if (!files.length) return;
    setResults(null);
    setConfirmOpen(false);

    const accepted: File[] = [];
    const rejected: string[] = [];
    for (const file of files) {
      const err = validateClientFile(file, mode, limits);
      if (err) rejected.push(err);
      else accepted.push(file);
    }

    if (rejected.length) {
      const preview = rejected.slice(0, 3).join(" ");
      const extra = rejected.length > 3 ? ` (+${rejected.length - 3} file khác)` : "";
      setError(
        accepted.length
          ? `Đã bỏ ${rejected.length} file không hợp lệ. ${preview}${extra}`
          : `Không chọn được file. ${preview}${extra}`,
      );
    } else {
      setError(null);
    }

    if (!accepted.length) return;

    const next = toPreviewItems(accepted);
    setItems((prev) => {
      const merged = [...prev, ...next];
      setActiveId((current) =>
        current && merged.some((i) => i.id === current) ? current : (merged[0]?.id ?? ""),
      );
      return merged;
    });
    attachThumbs(next);
  }

  function removeItem(id: string) {
    setItems((prev) => {
      const target = prev.find((i) => i.id === id);
      if (target) revokePreviewItem(target);
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
    setError(mode === "wp" ? "Đã hủy thu nhỏ ảnh." : "Đã hủy lưu ảnh.");
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
    const keepPreview = new Set(itemsRef.current.map((i) => i.previewUrl));
    previous?.forEach((r) => {
      if (r.result && r.previewUrl) keepPreview.add(r.previewUrl);
    });
    revokeResultPreviews(resultsRef.current, keepPreview);
    setResults(null);

    const resultMap = new Map<string, ResultItem>();
    previous?.forEach((r) => {
      if (r.result) resultMap.set(r.id, r);
    });

    const orderIds = previous?.map((r) => r.id) ?? toUpload.map((t) => t.id);
    let aborted = false;
    const failFallback = mode === "wp" ? "Thu nhỏ không thành công" : "Lưu không thành công";

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
          const message = err instanceof Error ? err.message : failFallback;
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
          const firstError = ordered.find((c) => c.error)?.error;
          setError(
            firstError ||
              (mode === "wp"
                ? "Không thu nhỏ được ảnh nào. Kiểm tra kết nối rồi thử lại."
                : "Không lưu được ảnh nào. Kiểm tra kết nối rồi thử lại."),
          );
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

  const confirmTitle = useMemo(() => {
    if (items.length > 1) {
      return mode === "wp"
        ? `Thu nhỏ ${items.length} ảnh chứ?`
        : `Lưu ${items.length} ảnh chứ?`;
    }
    return mode === "wp" ? "Thu nhỏ ảnh này chứ?" : "Lưu ảnh này chứ?";
  }, [items.length, mode]);

  const confirmMessage = useMemo(() => {
    const n = items.length;
    const confirmBtn = mode === "wp" ? "Có, thu nhỏ" : "Có, lưu";
    if (n <= 1) {
      return `${modeVerbCap} “${items[0]?.file.name ?? "ảnh"}”? Ảnh chỉ lên kho khi bạn bấm “${confirmBtn}”.`;
    }
    return `${modeVerbCap} ${n} ảnh? Ảnh chỉ lên kho khi bạn bấm “${confirmBtn}”.`;
  }, [items, mode, modeVerbCap]);

  return (
    <LayoutGroup>
      <div className="bg-logo" aria-hidden />
      <div className="title-atmosphere" aria-hidden />

      {showDecor && (
        <div className="mascot-scene">
          <motion.div
            className="mascot mascot-silhouette"
            aria-hidden
            initial={{ opacity: 0, scale: 0.92 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ delay: 0.15, duration: 0.8, ease: [0.22, 1, 0.36, 1] }}
          >
            <img src="/images/vas-dragon-silhouette.png" alt="" />
          </motion.div>
          <motion.div
            className="mascot mascot-left"
            aria-hidden
            initial={{ opacity: 0, x: -28, y: 20 }}
            animate={{ opacity: 1, x: 0, y: 0 }}
            transition={{ delay: 0.28, duration: 0.7, ease: [0.22, 1, 0.36, 1] }}
          >
            <img src="/images/vas-mascot-wave.png" alt="" />
          </motion.div>
          <motion.div
            className={`mascot mascot-right${narrow ? "" : " mascot-hotspot"}`}
            data-hint={!narrow && mascotHint && !onboardingOpen ? "true" : undefined}
            initial={{ opacity: 0, x: 28, y: 16 }}
            animate={{ opacity: 1, x: 0, y: 0 }}
            transition={{ delay: 0.4, duration: 0.7, ease: [0.22, 1, 0.36, 1] }}
            onMouseEnter={() => {
              if (!narrow) setMascotHint(true);
            }}
            onMouseLeave={() => setMascotHint(false)}
            onFocusCapture={() => {
              if (!narrow) setMascotHint(true);
            }}
            onBlurCapture={(e) => {
              if (!e.currentTarget.contains(e.relatedTarget as Node | null)) {
                setMascotHint(false);
              }
            }}
          >
            {narrow ? (
              <img src="/images/vas-mascot-wave.png" alt="" />
            ) : (
              <>
                <button
                  type="button"
                  className="mascot-hit"
                  aria-label="Xem giới thiệu Kho ảnh"
                  onClick={() => setOnboardingOpen(true)}
                >
                  <img src="/images/vas-mascot-wave.png" alt="" />
                </button>
                <OnboardingTrigger
                  visible={mascotHint && !onboardingOpen}
                  onClick={() => setOnboardingOpen(true)}
                />
              </>
            )}
          </motion.div>
        </div>
      )}

      <Onboarding open={onboardingOpen} onDone={() => setOnboardingOpen(false)} />
      {(narrow || !showDecor) && (
        <div className="onboard-fallback">
          <OnboardingTrigger
            visible={!onboardingOpen}
            onClick={() => setOnboardingOpen(true)}
          />
        </div>
      )}

      <div className="app-shell">
        <header className="topbar">
          <div className="chrome-brand">
            <div className="chrome-logo-plate">
              <img src="/images/vas-white.png" alt="VA Schools" className="chrome-logo" />
            </div>
          </div>
          <div className="chrome-tools">
            <div
              className="health-dot"
              data-ok={healthy === true ? "true" : healthy === false ? "false" : undefined}
            >
              <span className="health-dot-pulse" aria-hidden />
              <span className="health-dot-label">
                {healthy === null ? "Đang mở…" : healthy ? "Sẵn sàng" : "Chưa kết nối"}
              </span>
            </div>
            <div
              className="limit-note"
              title={`Tối đa ${maxSizeLabel} · file lớn tới ${absoluteMaxLabel}`}
            >
              <span className="limit-note-kicker">Giới hạn</span>
              <span className="limit-note-value">
                {maxSizeLabel}
                <span className="limit-note-sep">/</span>
                {absoluteMaxLabel}
              </span>
            </div>
          </div>
        </header>

        <div className="workspace">
          <section className={`intro${showDecor ? " intro-home" : ""}`}>
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
              Kho ảnh · Hệ thống Trường Việt Mỹ
            </motion.p>
            <motion.h1 className="headline" custom={2} variants={fadeUp} initial="hidden" animate="show">
              {showDecor ? "Ảnh chuẩn cho cả hệ thống" : "Lưu một lần, dùng mọi nơi"}
            </motion.h1>
            <motion.p
              className="lede"
              key={showDecor ? "home" : mode}
              custom={3}
              variants={fadeUp}
              initial="hidden"
              animate="show"
            >
              {showDecor ? homeLede : lede}
            </motion.p>

            {showDecor && (
              <motion.ul
                className="value-anchors"
                custom={4}
                variants={fadeUp}
                initial="hidden"
                animate="show"
                aria-label="Giá trị mang lại cho Trường Việt Mỹ"
              >
                {valueAnchors.map((item) => (
                  <li key={item.title}>
                    <strong>{item.title}</strong>
                    <span>{item.body}</span>
                  </li>
                ))}
              </motion.ul>
            )}

            <StepTimeline current={timelineStep} done={phaseDone && !busy} mode={mode} />

            <motion.div className="cta-row" custom={5} variants={fadeUp} initial="hidden" animate="show">
              <div className="mode-block">
                <div className="mode-switch" role="group" aria-label="Cách xử lý ảnh">
                  <button
                    type="button"
                    data-active={mode === "s3"}
                    onClick={() => setMode("s3")}
                    disabled={busy}
                  >
                    <IconSave />
                    Lưu ảnh
                  </button>
                  <button
                    type="button"
                    data-active={mode === "wp"}
                    onClick={() => setMode("wp")}
                    disabled={busy}
                  >
                    <IconShrink />
                    Thu nhỏ ảnh
                  </button>
                </div>
                <p className="mode-hint" key={mode} aria-live="polite">
                  {showDecor
                    ? mode === "wp"
                      ? "Dùng khi cần ảnh nhẹ, nhiều kích thước cho web và LMS."
                      : "Dùng khi cần lưu gốc — ảnh hoặc tài liệu — để chia sẻ lâu dài."
                    : modeHint}
                </p>
              </div>
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
                  modeLabel={modeLabel}
                  onSelectItem={setActiveId}
                  onRemoveItem={removeItem}
                  onAddMore={openFilePicker}
                  onClearAll={clearAll}
                  onRequestSend={requestSend}
                  onCancelUpload={cancelUpload}
                  onTransformChange={updateTransform}
                />
              ) : (
                <Dropzone
                  key="drop"
                  busy={busy}
                  mode={mode}
                  onSelect={handleSelect}
                  accept={fileAccept}
                  inputRef={fileInputRef}
                />
              )}
            </AnimatePresence>

            {items.length > 0 && !results && (
              <input
                ref={addMoreInputRef}
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
          title={confirmTitle}
          message={confirmMessage}
          confirmLabel={mode === "wp" ? "Có, thu nhỏ" : "Có, lưu"}
          cancelLabel="Chưa"
          busyLabel={`Đang ${modeLabel}…`}
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
    </LayoutGroup>
  );
}
