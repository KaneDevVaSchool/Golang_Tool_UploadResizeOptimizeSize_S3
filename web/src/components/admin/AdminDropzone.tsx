import { UploadCloud } from "lucide-react";
import { useEffect, useRef, useState, type DragEvent, type RefObject } from "react";

/**
 * Khung kéo-thả riêng cho khu quản trị - KHÔNG dùng chung với `Dropzone.tsx`
 * (component đó phục vụ trang `/upload` độc lập, nền tối kiểu glassmorphism
 * qua `index.css`, nạp global cho toàn app). Nhét thẳng `Dropzone` gốc vào
 * trang admin nền sáng là nguyên nhân giao diện "lệch tông" trước đây - tách
 * riêng component này để đổi hẳn phong cách mà không ảnh hưởng `/upload`.
 *
 * Cố ý không dùng framer-motion: đây là công cụ quản trị, không phải trang
 * trình diễn - phẳng, nhẹ, không hiệu ứng xoay 3D/viền chạy.
 */
export function AdminDropzone({
  busy,
  onSelect,
  inputRef: externalRef,
}: {
  busy: boolean;
  onSelect: (files: File[]) => void;
  inputRef?: RefObject<HTMLInputElement | null>;
}) {
  const localRef = useRef<HTMLInputElement>(null);
  const inputRef = externalRef ?? localRef;
  const [dragging, setDragging] = useState(false);

  function pickFiles(fileList: FileList | null) {
    if (!fileList?.length) return;
    onSelect(Array.from(fileList));
  }

  function openPicker() {
    if (!busy) inputRef.current?.click();
  }

  function onDragOver(e: DragEvent) {
    e.preventDefault();
    if (!busy) setDragging(true);
  }

  function onDragLeave() {
    setDragging(false);
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    setDragging(false);
    if (!busy) pickFiles(e.dataTransfer.files);
  }

  const onSelectRef = useRef(onSelect);
  onSelectRef.current = onSelect;

  // Dán ảnh từ clipboard (Ctrl/Cmd+V) - giữ hành vi đã có ở Dropzone gốc,
  // hữu ích khi admin chụp màn hình hoặc copy ảnh từ nơi khác.
  useEffect(() => {
    function onPaste(e: ClipboardEvent) {
      if (busy) return;
      const clipItems = e.clipboardData?.items;
      if (!clipItems?.length) return;
      const files: File[] = [];
      for (const item of Array.from(clipItems)) {
        if (item.kind === "file" && item.type.startsWith("image/")) {
          const file = item.getAsFile();
          if (file) files.push(file);
        }
      }
      if (!files.length) return;
      e.preventDefault();
      onSelectRef.current(files);
    }
    window.addEventListener("paste", onPaste);
    return () => window.removeEventListener("paste", onPaste);
  }, [busy]);

  return (
    <div
      className="admin-dropzone"
      role="button"
      tabIndex={0}
      aria-label="Kéo ảnh vào đây, dán từ clipboard, hoặc bấm để chọn"
      data-active={dragging ? "true" : "false"}
      data-busy={busy ? "true" : "false"}
      onClick={openPicker}
      onKeyDown={(e) => {
        if (busy) return;
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          openPicker();
        }
      }}
      onDragOver={onDragOver}
      onDragLeave={onDragLeave}
      onDrop={onDrop}
    >
      <input
        ref={inputRef}
        className="admin-dropzone-input"
        type="file"
        multiple
        accept="image/*"
        onChange={(e) => {
          pickFiles(e.target.files);
          e.target.value = "";
        }}
      />
      <UploadCloud size={22} className="admin-dropzone-icon" aria-hidden />
      <p className="admin-dropzone-title">
        {dragging ? "Thả để thêm ảnh" : "Kéo ảnh vào đây, hoặc "}
        {!dragging && <span className="admin-dropzone-link">chọn từ máy</span>}
      </p>
      <p className="admin-dropzone-hint">Có thể chọn nhiều ảnh cùng lúc · hoặc dán bằng Ctrl/Cmd+V</p>
    </div>
  );
}
