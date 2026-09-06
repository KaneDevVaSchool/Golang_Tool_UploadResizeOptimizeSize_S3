import { AnimatePresence, motion } from "framer-motion";
import {
  CalendarDays,
  ChevronLeft,
  ChevronRight,
  Copy,
  Download,
  Eye,
  Info,
  Maximize2,
  MessageCircle,
  Minimize2,
  PanelRightClose,
  PanelRightOpen,
  Plus,
  RotateCcw,
  RotateCw,
  School,
  Share2,
  Sparkles,
  Trophy,
  X,
  ZoomIn,
  ZoomOut,
} from "lucide-react";
import { useCallback, useEffect, useRef, useState, type CSSProperties, type PointerEvent } from "react";
import { createPortal } from "react-dom";
import type { ArtworkWithMeta } from "../../lib/artworkApi";
import { artworkImageURL, artworkPictureSources } from "../../lib/artworkImage";
import { recordArtworkView, type ReactionCounts } from "../../lib/publicApi";
import { toast } from "../../lib/toastBus";
import { CommentBox } from "./CommentBox";
import { ReactionPicker } from "./ReactionPicker";

const MIN_ZOOM = 0.5;
const MAX_ZOOM = 4;
const ZOOM_STEP = 0.25;
// Ngưỡng để phân biệt "đang vuốt chuyển ảnh" với "chạm nhầm"/"đang cuộn
// dọc" - dưới ngưỡng này chưa tính là vuốt.
const SWIPE_DISTANCE_THRESHOLD = 60;

function formatDate(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime())
    ? "Đang cập nhật"
    : new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "long", year: "numeric" }).format(date);
}

function safeFileName(value: string): string {
  return value.trim().replace(/[<>:"/\\|?*\u0000-\u001F]/g, "-") || "tac-pham-vas";
}

function guessDownloadExtension(imageUrl: string): string {
  try {
    const pathname = new URL(imageUrl, window.location.origin).pathname;
    const ext = pathname.slice(pathname.lastIndexOf(".")).toLowerCase();
    if (/^\.(jpe?g|png|webp|gif)$/.test(ext)) return ext;
  } catch {
    // ignore
  }
  return ".jpg";
}

type Point = { x: number; y: number };
type ImageSize = { width: number; height: number };

export function PublicLightbox({
  items,
  activeIndex,
  onNavigate,
  onClose,
}: {
  items: ArtworkWithMeta[];
  activeIndex: number;
  onNavigate: (index: number) => void;
  onClose: () => void;
}) {
  const artwork = items[activeIndex];
  const open = Boolean(artwork);
  const [reactionCounts, setReactionCounts] = useState<ReactionCounts | null>(null);
  const [viewCount, setViewCount] = useState(0);
  const [zoom, setZoom] = useState(1);
  const [rotation, setRotation] = useState(0);
  const [pan, setPan] = useState<Point>({ x: 0, y: 0 });
  const [dragging, setDragging] = useState(false);
  const [detailsOpen, setDetailsOpen] = useState(() => typeof window === "undefined" || window.innerWidth > 940);
  const [composeOpen, setComposeOpen] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [naturalSize, setNaturalSize] = useState<ImageSize>({
    width: artwork?.width || 1,
    height: artwork?.height || 1,
  });
  const [fitSize, setFitSize] = useState<ImageSize>({ width: 1, height: 1 });
  // Ảnh nét đã tải xong chưa. Trước khi xong, hiển thị bản thumb (đã nằm sẵn
  // trong cache trình duyệt vì vừa thấy nó ở lưới) để lightbox mở ra là có
  // hình ngay, thay vì một khung trống trong lúc chờ ảnh lớn về.
  const [sharpLoaded, setSharpLoaded] = useState(false);
  const stageRef = useRef<HTMLDivElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const dragStartRef = useRef<Point | null>(null);
  // Theo dõi vuốt ngang trên cảm ứng khi CHƯA zoom - tách khỏi dragStartRef
  // (dùng để pan ảnh đã phóng to) vì đây là 2 cử chỉ khác nhau, không được
  // lẫn vào nhau: dragStartRef chỉ hoạt động khi zoom > 1 (xem
  // handlePointerDown), swipeStartRef chỉ khi zoom <= 1.
  const swipeStartRef = useRef<Point | null>(null);

  useEffect(() => {
    setReactionCounts(artwork?.reaction_counts ?? null);
    setViewCount(artwork?.view_count ?? 0);
    setZoom(1);
    setRotation(0);
    setPan({ x: 0, y: 0 });
    setNaturalSize({ width: artwork?.width || 1, height: artwork?.height || 1 });
    setSharpLoaded(false);
    setComposeOpen(false);
  }, [artwork?.id]);

  // Ghi nhận lượt xem — không truyền AbortSignal: StrictMode mount/unmount
  // mount sẽ abort request giữa chừng, client không cập nhật số và dễ tưởng
  // là chưa cộng (server có thể đã ghi xong nhưng UI không nhận response).
  useEffect(() => {
    if (!artwork?.id) return;
    let stale = false;
    recordArtworkView(artwork.id)
      .then((fresh) => {
        if (!stale) {
          setViewCount(fresh.view_count);
          setReactionCounts(fresh.reaction_counts ?? null);
        }
      })
      .catch(() => {});
    return () => {
      stale = true;
    };
  }, [artwork?.id]);

  useEffect(() => {
    if (!open) return;
    // Gỡ hẳn inline style khi đóng thay vì khôi phục snapshot: dưới
    // StrictMode effect chạy mount->unmount->mount, nên lần chạy thứ hai sẽ
    // "snapshot" đúng giá trị hidden do lần đầu vừa đặt, và cleanup khôi
    // phục lại hidden -> trang kẹt không cuộn được sau khi đóng lightbox.
    // Trạng thái cuộn của trang do class scrollable-page quyết định, inline
    // style ở đây chỉ là lớp phủ tạm thời nên xoá là về đúng mặc định.
    document.body.style.overflow = "hidden";
    return () => {
      document.body.style.removeProperty("overflow");
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    panelRef.current?.focus();
  }, [open]);

  useEffect(() => {
    const onFullscreenChange = () => setIsFullscreen(Boolean(document.fullscreenElement));
    document.addEventListener("fullscreenchange", onFullscreenChange);
    return () => document.removeEventListener("fullscreenchange", onFullscreenChange);
  }, []);

  const calculateFit = useCallback(() => {
    const stage = stageRef.current;
    if (!stage || naturalSize.width <= 1 || naturalSize.height <= 1) return;
    const availableWidth = Math.max(stage.clientWidth - 88, 120);
    const availableHeight = Math.max(stage.clientHeight - 132, 120);
    const quarterTurn = Math.abs(rotation / 90) % 2 === 1;
    const rotatedWidth = quarterTurn ? naturalSize.height : naturalSize.width;
    const rotatedHeight = quarterTurn ? naturalSize.width : naturalSize.height;
    const scale = Math.min(availableWidth / rotatedWidth, availableHeight / rotatedHeight);
    setFitSize({
      width: Math.max(1, Math.round(naturalSize.width * scale)),
      height: Math.max(1, Math.round(naturalSize.height * scale)),
    });
  }, [naturalSize, rotation]);

  useEffect(() => {
    calculateFit();
    const stage = stageRef.current;
    if (!stage || typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(calculateFit);
    observer.observe(stage);
    return () => observer.disconnect();
  }, [calculateFit, detailsOpen]);

  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (document.querySelector(".comment-modal-root")) return;
      const target = e.target as HTMLElement | null;
      const isEditing = target?.matches("input, textarea, select, [contenteditable='true']");
      if (isEditing && e.key !== "Escape") return;
      if (e.key === "Escape") {
        onClose();
      } else if (e.key === "ArrowLeft" && items.length > 1) {
        onNavigate((activeIndex - 1 + items.length) % items.length);
      } else if (e.key === "ArrowRight" && items.length > 1) {
        onNavigate((activeIndex + 1) % items.length);
      } else if (e.key === "+" || e.key === "=") {
        setZoom((value) => Math.min(MAX_ZOOM, value + ZOOM_STEP));
      } else if (e.key === "-") {
        setZoom((value) => Math.max(MIN_ZOOM, value - ZOOM_STEP));
      } else if (e.key === "0") {
        setZoom(1);
        setPan({ x: 0, y: 0 });
      } else if (e.key.toLowerCase() === "r") {
        setRotation((value) => (value + 90) % 360);
        setPan({ x: 0, y: 0 });
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, activeIndex, items.length, onNavigate, onClose]);

  if (typeof document === "undefined" || !artwork) return null;

  const shareUrl = new URL(window.location.href);
  shareUrl.searchParams.set("tranh", String(artwork.id));
  const facebookPreviewUrl = new URL(`/chia-se/tac-pham/${artwork.id}`, window.location.origin);
  const shareText = `Mời bạn ngắm “${artwork.title}” — tác phẩm của ${artwork.student_name} trong Khu vườn nghệ thuật VASchools.`;
  const imageStyle = {
    width: `${fitSize.width}px`,
    height: `${fitSize.height}px`,
    transform: `translate3d(${pan.x}px, ${pan.y}px, 0) rotate(${rotation}deg) scale(${zoom})`,
  } satisfies CSSProperties;

  // Dùng bản "large" (cạnh dài 1600px) chứ không phải ảnh gốc: đủ nét cho cả
  // màn hình lớn lẫn retina, mà nhẹ hơn ảnh dự thi gốc rất nhiều. Người muốn
  // xem nguyên bản vẫn có nút tải về, vốn trỏ thẳng image_url.
  // Tác phẩm chưa có biến thể thì helper tự lui về ảnh gốc.
  const sharpImageURL = artworkImageURL(artwork, "large");
  const sharpSources = artworkPictureSources(artwork, "large");

  function changeZoom(next: number) {
    const bounded = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, next));
    setZoom(bounded);
    if (bounded <= 1) setPan({ x: 0, y: 0 });
  }

  function rotate(delta: number) {
    setRotation((value) => (value + delta + 360) % 360);
    setPan({ x: 0, y: 0 });
  }

  function resetView() {
    setZoom(1);
    setRotation(0);
    setPan({ x: 0, y: 0 });
  }

  function handlePointerDown(event: PointerEvent<HTMLDivElement>) {
    if ((event.target as HTMLElement).closest("button")) return;
    // Chưa zoom + ngón tay chạm: đây là ứng viên vuốt-chuyển-ảnh, không
    // phải pan (pan chỉ có ý nghĩa khi ảnh đã phóng to hơn khung).
    if (zoom <= 1) {
      if (event.pointerType === "touch" && items.length > 1) {
        swipeStartRef.current = { x: event.clientX, y: event.clientY };
      }
      return;
    }
    if (event.button !== 0) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    dragStartRef.current = { x: event.clientX - pan.x, y: event.clientY - pan.y };
    setDragging(true);
  }

  function handlePointerMove(event: PointerEvent<HTMLDivElement>) {
    if (dragStartRef.current) {
      setPan({
        x: event.clientX - dragStartRef.current.x,
        y: event.clientY - dragStartRef.current.y,
      });
      return;
    }
  }

  function stopDragging(event: PointerEvent<HTMLDivElement>) {
    dragStartRef.current = null;
    setDragging(false);

    const swipeStart = swipeStartRef.current;
    swipeStartRef.current = null;
    if (!swipeStart) return;
    const deltaX = event.clientX - swipeStart.x;
    const deltaY = event.clientY - swipeStart.y;
    // |deltaX| > |deltaY| loại trường hợp người dùng đang cuộn dọc (vuốt
    // chéo lên/xuống) chứ không cố ý chuyển ảnh.
    if (Math.abs(deltaX) < SWIPE_DISTANCE_THRESHOLD || Math.abs(deltaX) <= Math.abs(deltaY)) return;
    if (deltaX < 0) {
      onNavigate((activeIndex + 1) % items.length);
    } else {
      onNavigate((activeIndex - 1 + items.length) % items.length);
    }
  }

  async function copyLink() {
    try {
      await navigator.clipboard.writeText(shareUrl.toString());
      toast.success("Đã sao chép liên kết đến đúng tác phẩm.");
    } catch {
      toast.error("Không thể sao chép tự động. Vui lòng sao chép liên kết trên thanh địa chỉ.");
    }
  }

  async function shareArtwork() {
    if (navigator.share) {
      try {
        await navigator.share({ title: artwork.title, text: shareText, url: shareUrl.toString() });
        return;
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") return;
      }
    }
    await copyLink();
  }

  function shareFacebook() {
    const url = `https://www.facebook.com/sharer/sharer.php?u=${encodeURIComponent(facebookPreviewUrl.toString())}`;
    window.open(url, "vas-facebook-share", "popup=yes,width=680,height=560,noopener,noreferrer");
  }

  async function downloadArtwork() {
    const downloadUrl = `/api/v1/public/artworks/${artwork.id}/download`;
    const suggestedName = `${safeFileName(artwork.title)}${guessDownloadExtension(artwork.image_url)}`;
    try {
      const response = await fetch(downloadUrl);
      if (!response.ok) throw new Error("download failed");
      const blobUrl = URL.createObjectURL(await response.blob());
      const anchor = document.createElement("a");
      anchor.href = blobUrl;
      anchor.download = suggestedName;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(blobUrl);
      toast.success("Ảnh gốc đang được tải xuống.");
    } catch {
      const anchor = document.createElement("a");
      anchor.href = downloadUrl;
      anchor.download = suggestedName;
      anchor.rel = "noopener";
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      toast.success("Ảnh gốc đang được tải xuống.");
    }
  }

  async function toggleFullscreen() {
    try {
      if (document.fullscreenElement) await document.exitFullscreen();
      else await panelRef.current?.requestFullscreen();
    } catch {
      toast.info("Trình duyệt không hỗ trợ chế độ toàn màn hình.");
    }
  }

  return createPortal(
    <AnimatePresence>
      {open && (
        <motion.div
          className="public-lightbox-root"
          role="dialog"
          aria-modal="true"
          aria-label={`Xem tác phẩm: ${artwork.title}`}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.22 }}
        >
          <button type="button" className="public-lightbox-backdrop" aria-label="Đóng" onClick={onClose} />

          <motion.div
            className="public-lightbox-panel"
            ref={panelRef}
            tabIndex={-1}
            initial={{ opacity: 0, scale: 0.96, y: 16 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: 12 }}
            transition={{ duration: 0.26, ease: [0.22, 1, 0.36, 1] }}
          >
            <header className="public-lightbox-header">
              <div className="public-lightbox-brand">
                <span className="public-lightbox-brand-mark" aria-hidden>
                  <Sparkles size={17} />
                </span>
                <span>
                  <strong>Khu vườn nghệ thuật</strong>
                  <small>
                    Tác phẩm {activeIndex + 1} / {items.length}
                  </small>
                </span>
              </div>

              <div className="public-lightbox-actions" aria-label="Công cụ xem tranh">
                <div className="public-lightbox-action-group">
                  <button type="button" onClick={() => changeZoom(zoom - ZOOM_STEP)} disabled={zoom <= MIN_ZOOM} title="Thu nhỏ (-)">
                    <ZoomOut size={18} />
                    <span>Thu nhỏ</span>
                  </button>
                  <button type="button" className="public-lightbox-zoom-value" onClick={resetView} title="Về kích thước vừa khung (0)">
                    {Math.round(zoom * 100)}%
                  </button>
                  <button type="button" onClick={() => changeZoom(zoom + ZOOM_STEP)} disabled={zoom >= MAX_ZOOM} title="Phóng to (+)">
                    <ZoomIn size={18} />
                    <span>Phóng to</span>
                  </button>
                </div>

                <div className="public-lightbox-action-group">
                  <button type="button" onClick={() => rotate(-90)} title="Xoay trái">
                    <RotateCcw size={18} />
                    <span>Xoay trái</span>
                  </button>
                  <button type="button" onClick={() => rotate(90)} title="Xoay phải (R)">
                    <RotateCw size={18} />
                    <span>Xoay phải</span>
                  </button>
                </div>

                <div className="public-lightbox-action-group public-lightbox-action-group--share">
                  <button type="button" onClick={shareFacebook} title="Chia sẻ qua Facebook">
                    <span className="public-lightbox-facebook" aria-hidden>f</span>
                    <span>Facebook</span>
                  </button>
                  <button type="button" onClick={copyLink} title="Sao chép liên kết">
                    <Copy size={17} />
                    <span>Copy link</span>
                  </button>
                  <button type="button" onClick={shareArtwork} title="Chia sẻ">
                    <Share2 size={17} />
                    <span>Chia sẻ</span>
                  </button>
                  <button type="button" onClick={downloadArtwork} title="Tải ảnh gốc">
                    <Download size={17} />
                    <span>Tải ảnh</span>
                  </button>
                </div>

                <button type="button" className="public-lightbox-icon-action" onClick={toggleFullscreen} title={isFullscreen ? "Thoát toàn màn hình" : "Toàn màn hình"}>
                  {isFullscreen ? <Minimize2 size={18} /> : <Maximize2 size={18} />}
                </button>
                <button type="button" className="public-lightbox-icon-action" onClick={() => setDetailsOpen((value) => !value)} title={detailsOpen ? "Ẩn chi tiết" : "Hiện chi tiết"}>
                  {detailsOpen ? <PanelRightClose size={18} /> : <PanelRightOpen size={18} />}
                </button>
              </div>

              <button type="button" className="public-lightbox-close" aria-label="Đóng phòng tranh" onClick={onClose}>
                <X size={21} />
              </button>
            </header>

            <div className="public-lightbox-content" data-details={detailsOpen ? "open" : "closed"}>
              <div
                ref={stageRef}
                className={`public-lightbox-image-wrap${dragging ? " is-dragging" : ""}${zoom > 1 ? " is-zoomed" : ""}`}
                onPointerDown={handlePointerDown}
                onPointerMove={handlePointerMove}
                onPointerUp={stopDragging}
                onPointerCancel={stopDragging}
                onWheel={(event) => {
                  event.preventDefault();
                  changeZoom(zoom + (event.deltaY < 0 ? ZOOM_STEP : -ZOOM_STEP));
                }}
              >
                <span className="public-lightbox-stage-glow" aria-hidden />
                <span className="public-lightbox-stage-leaf public-lightbox-stage-leaf--one" aria-hidden>❧</span>
                <span className="public-lightbox-stage-leaf public-lightbox-stage-leaf--two" aria-hidden>❧</span>

                {/* Hai lớp ảnh chồng nhau: bản thumb hiện ngay (lấy từ cache
                    của lưới) rồi mờ dần đi khi bản nét đã tải xong. Trình duyệt
                    tự phóng to thumb nên trong khoảnh khắc đầu ảnh hơi nhoè,
                    vẫn hơn hẳn việc nhìn vào khung trống. */}
                {!sharpLoaded && artworkImageURL(artwork, "thumb") !== sharpImageURL && (
                  <img
                    src={artworkImageURL(artwork, "thumb")}
                    alt=""
                    aria-hidden
                    draggable={false}
                    style={{ ...imageStyle, position: "absolute", filter: "blur(6px)" }}
                  />
                )}

                <picture>
                  {sharpSources.map((s) => (
                    <source key={s.type} srcSet={s.srcSet} type={s.type} />
                  ))}
                  <img
                    src={sharpImageURL}
                    alt={artwork.title}
                    draggable={false}
                    decoding="async"
                    style={{ ...imageStyle, opacity: sharpLoaded ? 1 : 0, transition: "opacity 0.25s ease-out" }}
                    onLoad={(event) => {
                      setNaturalSize({
                        width: event.currentTarget.naturalWidth || artwork.width || 1,
                        height: event.currentTarget.naturalHeight || artwork.height || 1,
                      });
                      setSharpLoaded(true);
                    }}
                    onError={() => setSharpLoaded(true)}
                  />
                </picture>

                {items.length > 1 && (
                  <>
                    <button
                      type="button"
                      className="public-lightbox-nav public-lightbox-nav--prev"
                      aria-label="Tác phẩm trước"
                      onClick={(event) => {
                        event.stopPropagation();
                        onNavigate((activeIndex - 1 + items.length) % items.length);
                      }}
                    >
                      <ChevronLeft size={24} />
                    </button>
                    <button
                      type="button"
                      className="public-lightbox-nav public-lightbox-nav--next"
                      aria-label="Tác phẩm sau"
                      onClick={(event) => {
                        event.stopPropagation();
                        onNavigate((activeIndex + 1) % items.length);
                      }}
                    >
                      <ChevronRight size={24} />
                    </button>
                  </>
                )}

                <div className="public-lightbox-stage-footer">
                  <span>{rotation ? `Đã xoay ${rotation}°` : "Ảnh gốc"}</span>
                  <span aria-hidden>•</span>
                  <span>{zoom > 1 ? "Kéo để khám phá chi tiết" : "Lăn chuột để phóng to"}</span>
                </div>
              </div>

              <AnimatePresence initial={false}>
                {detailsOpen && (
                  <motion.aside
                    className="public-lightbox-sidebar"
                    initial={{ opacity: 0, x: 32 }}
                    animate={{ opacity: 1, x: 0 }}
                    exit={{ opacity: 0, x: 32 }}
                    transition={{ duration: 0.2 }}
                  >
                    <div className="public-lightbox-meta">
                      <div className="public-lightbox-detail-label">
                        <Info size={14} />
                        Câu chuyện tác phẩm
                      </div>
                      {artwork.awards && artwork.awards.length > 0 && (
                        <div className="public-lightbox-honors">
                          {artwork.awards.map((award) => (
                            <div key={award.id} className="public-lightbox-honor" style={{ "--honor-color": award.color_hex } as CSSProperties}>
                              <span className="public-lightbox-honor-icon"><Trophy size={19} /></span>
                              <span>
                                <small>Thành tích nổi bật</small>
                                <strong>{award.name}</strong>
                              </span>
                              <i aria-hidden />
                            </div>
                          ))}
                        </div>
                      )}
                      <h2>“{artwork.title}”</h2>
                      <p className="public-lightbox-poem">
                        Một khoảng trời được kể bằng sắc màu, nơi trí tưởng tượng hồn nhiên chạm vào những điều thân thương.
                        Mỗi đường nét là một hạt mầm ký ức — nhỏ bé, trong trẻo và mang theo cách nhìn rất riêng của tuổi học trò.
                      </p>

                      <div className="public-lightbox-artist">
                        <span className="public-lightbox-artist-avatar">
                          {artwork.student_name.charAt(0).toUpperCase()}
                        </span>
                        <span>
                          <small>Họa sĩ nhí</small>
                          <strong>{artwork.student_name}</strong>
                          <em>{artwork.grade_label}{artwork.class_name ? ` · Lớp ${artwork.class_name}` : ""}</em>
                        </span>
                      </div>

                      <dl className="public-lightbox-facts">
                        <div>
                          <dt><School size={15} /> Mái trường</dt>
                          <dd>{artwork.school_name}</dd>
                        </div>
                        <div>
                          <dt><CalendarDays size={15} /> Ngày giới thiệu</dt>
                          <dd>{formatDate(artwork.created_at)}</dd>
                        </div>
                      </dl>

                      <div className="public-lightbox-stats">
                        <span><Eye size={15} /> {viewCount.toLocaleString("vi-VN")} lượt xem</span>
                        <span><MessageCircle size={15} /> {artwork.comment_count.toLocaleString("vi-VN")} cảm nhận</span>
                      </div>

                      <div className="public-lightbox-mascot-note">
                        <motion.img
                          src="/images/vas-mascot-wave.png"
                          alt=""
                          aria-hidden
                          animate={{ y: [0, -5, 0], rotate: [-2, 2, -2] }}
                          transition={{ duration: 3.2, repeat: Infinity, ease: "easeInOut" }}
                        />
                        <p><strong>VAers mách nhỏ:</strong> Phóng thật gần để gặp những nét vẽ bé xíu đang cất giấu cả một câu chuyện nhé!</p>
                      </div>

                      <div className="public-lightbox-reaction-row">
                        <span>Bạn thấy tác phẩm thế nào?</span>
                        <ReactionPicker artworkId={artwork.id} counts={reactionCounts} onCountsChange={setReactionCounts} />
                      </div>
                    </div>

                    <div className="public-lightbox-comments">
                      <div className="public-lightbox-comments-head">
                        <div className="public-lightbox-detail-label">
                          <MessageCircle size={14} />
                          Gửi một lời yêu thương
                        </div>
                        <button
                          type="button"
                          className="comment-add-btn"
                          aria-label="Viết bình luận"
                          title="Viết bình luận"
                          onClick={() => setComposeOpen(true)}
                        >
                          <Plus size={18} />
                        </button>
                      </div>
                      <p className="public-lightbox-comments-intro">Một lời động viên đẹp có thể nuôi lớn thêm một ước mơ.</p>
                      <CommentBox artworkId={artwork.id} composeOpen={composeOpen} onComposeOpenChange={setComposeOpen} />
                    </div>
                  </motion.aside>
                )}
              </AnimatePresence>
            </div>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>,
    document.body,
  );
}
