import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { Award, Search } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { createPortal } from "react-dom";
import { artworkImageURL } from "../../lib/artworkImage";
import { detectIntent, isQueryTooShort, runMascotSearch, type MascotResult } from "../../lib/mascotSearch";

const HIDE_STORAGE_KEY = "vas_mascot_hidden";

/** Câu mở đầu ngắn, đổi mỗi lần mở panel. */
const GREETINGS = [
  "Hôm nay mình tìm bức nào nhỉ?",
  "Gõ tên tranh, tên bạn, hoặc hỏi ai đạt giải.",
  "Khu vườn tranh đang chờ — con muốn nghe chuyện ai?",
  "Một cái tên là đủ. Mình đi tìm cùng!",
];

const INSPIRE_IDLE = "Mỗi nét vẽ là một câu chuyện. Con thử tìm câu chuyện của mình.";

const EXAMPLE_CHIPS = ["ai đạt giải Nhất", "giải Nhì", "tranh khối Tiểu học"];

function inspireForResult(result: MascotResult): string {
  if (result.intent === "awards") {
    if (result.entries.length === 0) {
      return "Chưa thấy giải khớp — nhưng gửi tranh rồi là đã thắng một lần.";
    }
    return result.matchedBy === "loose"
      ? "Không khớp y hệt. Mình đoán con muốn nói đến những gương mặt này."
      : "Chúc mừng những họa sĩ nhí đã dám vẽ và dám gửi tranh.";
  }
  return result.items.length > 0
    ? "Đây rồi — mỗi bức đều được vẽ bằng cả trái tim."
    : "Chưa khớp. Thử một phần tên, hoặc bỏ dấu xem sao.";
}

/**
 * Mascot rồng nổi góc dưới-phải, chỉ desktop (≥1024px). Click mascot
 * mở/đóng panel tìm kiếm nội bộ (không gọi AI ngoài).
 *
 * Đóng panel: click lại mascot, click ra ngoài dock, hoặc Escape.
 * Ẩn trợ lý: chữ "Ẩn" trên panel — thu thành nút "Rồng nhỏ" cùng góc.
 * Hai trạng thái này không được unmount chung một nhịp, kẻo cắt animation exit.
 */
export function MascotAssistant() {
  const [hidden, setHidden] = useState<boolean>(() => {
    try {
      return localStorage.getItem(HIDE_STORAGE_KEY) === "1";
    } catch {
      return false;
    }
  });
  const [leaving, setLeaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [result, setResult] = useState<MascotResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [greeting] = useState(() => GREETINGS[Math.floor(Math.random() * GREETINGS.length)]);
  const inputRef = useRef<HTMLInputElement>(null);
  const dockRef = useRef<HTMLDivElement>(null);
  const reduceMotion = useReducedMotion();

  useEffect(() => {
    if (!open) return;
    const focusTimer = window.setTimeout(() => inputRef.current?.focus(), 260);

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") setOpen(false);
    }
    function onClickOutside(event: MouseEvent) {
      if (dockRef.current && !dockRef.current.contains(event.target as Node)) setOpen(false);
    }
    document.addEventListener("keydown", onKeyDown);
    document.addEventListener("mousedown", onClickOutside);
    return () => {
      window.clearTimeout(focusTimer);
      document.removeEventListener("keydown", onKeyDown);
      document.removeEventListener("mousedown", onClickOutside);
    };
  }, [open]);

  useEffect(() => {
    const trimmed = query.trim();
    if (!trimmed || isQueryTooShort(trimmed)) {
      setResult(null);
      setLoading(false);
      return;
    }
    const controller = new AbortController();
    setLoading(true);
    const timer = window.setTimeout(() => {
      runMascotSearch(trimmed, controller.signal)
        .then((res) => setResult(res))
        .catch(() => {
          if (!controller.signal.aborted) setResult(null);
        })
        .finally(() => {
          if (!controller.signal.aborted) setLoading(false);
        });
    }, 300);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [query]);

  function handleHide() {
    setOpen(false);
    setLeaving(true);
    try {
      localStorage.setItem(HIDE_STORAGE_KEY, "1");
    } catch {
      // Ẩn danh chặn localStorage — vẫn ẩn trong phiên này.
    }
  }

  function handleRestore() {
    setHidden(false);
    setLeaving(false);
    setOpen(true);
    try {
      localStorage.removeItem(HIDE_STORAGE_KEY);
    } catch {
      // Không ghi được thì chỉ khôi phục trong phiên này.
    }
  }

  const trimmedQuery = query.trim();
  const intent = trimmedQuery ? detectIntent(trimmedQuery) : null;
  const tooShort = isQueryTooShort(trimmedQuery);

  return createPortal(
    <div className="mascot-assistant">
      <AnimatePresence mode="wait" onExitComplete={() => { if (leaving) setHidden(true); }}>
        {hidden ? (
          <motion.button
            key="mascot-restore"
            type="button"
            className="mascot-restore"
            onClick={handleRestore}
            aria-label="Gọi lại rồng nhỏ"
            initial={{ opacity: 0, y: 10, scale: 0.92 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 8, scale: 0.94, transition: { duration: 0.16 } }}
            transition={{ duration: 0.28, ease: [0.16, 1, 0.3, 1] }}
          >
            <span className="mascot-restore-avatar" aria-hidden>
              <img src="/images/vas-mascot-wave.png" alt="" />
            </span>
          </motion.button>
        ) : !leaving ? (
          <motion.div
            key="mascot-dock"
            ref={dockRef}
            className="mascot-dock"
            exit={{ opacity: 0, scale: 0.85, transition: { duration: 0.28, ease: [0.4, 0, 1, 1] } }}
          >
            <AnimatePresence>
              {open && (
                <motion.div
                  className="mascot-panel"
                  role="dialog"
                  aria-modal="false"
                  aria-label="Rồng nhỏ, trợ lý tìm kiếm"
                  initial={{ opacity: 0, y: 24, scale: 0.92 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: 18, scale: 0.94, transition: { duration: 0.18, ease: [0.4, 0, 1, 1] } }}
                  transition={{ duration: 0.32, ease: [0.16, 1, 0.3, 1] }}
                >
                  <header className="mascot-panel-header">
                    <div className="mascot-panel-heading">
                      <span className="mascot-panel-heading-label">Rồng nhỏ</span>
                      <span className="mascot-panel-heading-greeting">{greeting}</span>
                    </div>
                    <button
                      type="button"
                      className="mascot-panel-dismiss"
                      onClick={handleHide}
                      aria-label="Ẩn trợ lý rồng nhỏ"
                    >
                      Ẩn
                    </button>
                  </header>

                  <label className="mascot-search-field">
                    <Search size={15} strokeWidth={2} aria-hidden />
                    <input
                      ref={inputRef}
                      type="search"
                      value={query}
                      onChange={(event) => setQuery(event.target.value)}
                      placeholder="Tên tranh, tên bạn, hoặc giải thưởng…"
                      aria-label="Tìm tác phẩm, tác giả hoặc giải thưởng"
                      maxLength={80}
                      autoComplete="off"
                      enterKeyHint="search"
                    />
                    <AnimatePresence>
                      {loading && (
                        <motion.span
                          className="mascot-search-spinner"
                          aria-hidden
                          initial={{ opacity: 0 }}
                          animate={{ opacity: 1 }}
                          exit={{ opacity: 0 }}
                        />
                      )}
                    </AnimatePresence>
                  </label>

                  <div className="mascot-panel-body">
                    {!trimmedQuery && (
                      <>
                        <p className="mascot-inspire">{INSPIRE_IDLE}</p>
                        <div className="mascot-chip-row">
                          {EXAMPLE_CHIPS.map((chip) => (
                            <button
                              key={chip}
                              type="button"
                              className="mascot-chip"
                              onClick={() => setQuery(chip)}
                            >
                              {chip}
                            </button>
                          ))}
                        </div>
                      </>
                    )}

                    {tooShort && <p className="mascot-status">Gõ thêm một chút nữa nhé…</p>}

                    <AnimatePresence mode="wait">
                      {!tooShort && loading && (
                        <motion.p
                          key="loading"
                          className="mascot-status mascot-status--loading"
                          initial={{ opacity: 0 }}
                          animate={{ opacity: 1 }}
                          exit={{ opacity: 0 }}
                        >
                          Đang lật từng bức tranh…
                        </motion.p>
                      )}

                      {!tooShort && !loading && result && (
                        <motion.div
                          key={`${result.intent}-${trimmedQuery}`}
                          initial={{ opacity: 0, y: 6 }}
                          animate={{ opacity: 1, y: 0 }}
                          exit={{ opacity: 0 }}
                          transition={{ duration: 0.22 }}
                        >
                          {result.intent === "search" && (
                            <>
                              {result.items.length > 0 ? (
                                <ul className="mascot-result-list">
                                  {result.items.map((item, i) => (
                                    <motion.li
                                      key={item.id}
                                      initial={{ opacity: 0, x: -8 }}
                                      animate={{ opacity: 1, x: 0 }}
                                      transition={{ delay: i * 0.035, duration: 0.22 }}
                                    >
                                      <Link
                                        className="mascot-result-item"
                                        to={`/phong-trien-lam?tim=${encodeURIComponent(trimmedQuery)}&tranh=${item.id}`}
                                        onClick={() => setOpen(false)}
                                      >
                                        <img src={artworkImageURL(item, "thumb")} alt="" className="mascot-result-thumb" />
                                        <span className="mascot-result-meta">
                                          <span className="mascot-result-title">{item.title}</span>
                                          <span className="mascot-result-sub">
                                            {item.student_name} · {item.school_name}
                                          </span>
                                        </span>
                                      </Link>
                                    </motion.li>
                                  ))}
                                </ul>
                              ) : (
                                <p className="mascot-status">
                                  Chưa thấy tranh tên “{trimmedQuery}”. Thử gõ ngắn hơn xem?
                                </p>
                              )}
                              {result.totalCount > result.items.length && (
                                <Link
                                  className="mascot-see-all"
                                  to={`/phong-trien-lam?tim=${encodeURIComponent(trimmedQuery)}`}
                                  onClick={() => setOpen(false)}
                                >
                                  Xem hết {result.totalCount} kết quả
                                </Link>
                              )}
                            </>
                          )}

                          {result.intent === "awards" && (
                            <>
                              {result.matchedBy === "loose" && result.entries.length > 0 && (
                                <p className="mascot-hint">Không khớp y hệt — có lẽ là:</p>
                              )}
                              {result.entries.length > 0 ? (
                                <ul className="mascot-result-list">
                                  {result.entries.slice(0, 8).map((entry, i) => (
                                    <motion.li
                                      key={entry.id}
                                      initial={{ opacity: 0, x: -8 }}
                                      animate={{ opacity: 1, x: 0 }}
                                      transition={{ delay: i * 0.035, duration: 0.22 }}
                                    >
                                      <Link
                                        className="mascot-result-item"
                                        to={`/bang-vang?tranh=${entry.id}`}
                                        onClick={() => setOpen(false)}
                                      >
                                        <img src={artworkImageURL(entry, "thumb")} alt="" className="mascot-result-thumb" />
                                        <span className="mascot-result-meta">
                                          <span className="mascot-result-title">
                                            <Award
                                              size={12}
                                              strokeWidth={2.4}
                                              aria-hidden
                                              style={{ color: entry.award.color_hex }}
                                            />
                                            {entry.award.name}
                                          </span>
                                          <span className="mascot-result-sub">
                                            {entry.student_name} · “{entry.title}”
                                          </span>
                                        </span>
                                      </Link>
                                    </motion.li>
                                  ))}
                                </ul>
                              ) : (
                                <p className="mascot-status">Chưa thấy giải nào khớp. Thử tên tranh hoặc tên bạn.</p>
                              )}
                              <Link className="mascot-see-all" to="/bang-vang" onClick={() => setOpen(false)}>
                                Mở Bảng vàng
                              </Link>
                            </>
                          )}

                          <p className="mascot-inspire mascot-inspire--result">{inspireForResult(result)}</p>
                        </motion.div>
                      )}

                      {intent === "awards" && !tooShort && !loading && !result && (
                        <motion.p key="opening" className="mascot-status" initial={{ opacity: 0 }} animate={{ opacity: 1 }}>
                          Đang mở Bảng vàng…
                        </motion.p>
                      )}
                    </AnimatePresence>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>

            <motion.button
              type="button"
              className={`mascot-fab${open ? " mascot-fab--active" : ""}`}
              onClick={() => setOpen((v) => !v)}
              aria-label={open ? "Đóng rồng nhỏ" : "Mở rồng nhỏ"}
              aria-expanded={open}
              initial={{ scale: 0, rotate: -20 }}
              animate={
                open || reduceMotion
                  ? { scale: 1, rotate: 0 }
                  : { scale: 1, rotate: [0, -4, 4, -4, 0], y: [0, -5, 0] }
              }
              transition={
                open || reduceMotion
                  ? { duration: 0.3, ease: [0.34, 1.56, 0.64, 1] }
                  : { duration: 4.2, repeat: Infinity, ease: "easeInOut", repeatDelay: 1.4 }
              }
              whileHover={{ scale: 1.08 }}
              whileTap={{ scale: 0.94 }}
            >
              <img src="/images/vas-mascot-wave.png" alt="" className="mascot-fab-img" />
            </motion.button>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </div>,
    document.body,
  );
}
