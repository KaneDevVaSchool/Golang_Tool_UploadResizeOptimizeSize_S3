import { AlertCircle, Check, CheckCircle2, Loader2, X } from "lucide-react";
import type { BulkRow } from "./artworkUploadTypes";

/**
 * Lưới thẻ ảnh cho trang tải tác phẩm lên - thay thế `ArtworkBulkTable`
 * (bảng HTML mỗi ảnh 1 hàng, phải cuộn ngang trên mobile). Mỗi ảnh là 1 thẻ
 * vuông đủ lớn để nhận diện, không cần popover phóng to như bảng cũ.
 *
 * Click vào thẻ = CHỌN RIÊNG đúng thẻ đó (thay thế toàn bộ lựa chọn hiện
 * có), giống hành vi file-manager thông thường. Click ô tick = bật/tắt thẻ
 * đó trong lựa chọn hiện có (multi-select) để sửa hàng loạt - xem
 * `ArtworksUploadPage.tsx` phần panel bên phải.
 */
export function ArtworkPickerGrid({
  rows,
  selectedIds,
  disabled,
  errorRowIds,
  onSelectOnly,
  onToggleSelect,
  onRemoveRow,
}: {
  rows: BulkRow[];
  selectedIds: Set<string>;
  disabled?: boolean;
  /** localId của các thẻ cần buộc hiện lỗi ngay (bấm "Lưu tất cả" mà còn thiếu). */
  errorRowIds?: Set<string>;
  onSelectOnly: (localId: string) => void;
  onToggleSelect: (localId: string) => void;
  onRemoveRow: (localId: string) => void;
}) {
  return (
    <div className="artwork-picker-grid" role="listbox" aria-label="Danh sách ảnh đã chọn" aria-multiselectable="true">
      {rows.map((row) => {
        const selected = selectedIds.has(row.localId);
        const hasError = errorRowIds?.has(row.localId) ?? false;
        const locked = row.status === "saving" || row.status === "done";
        return (
          <div
            key={row.localId}
            role="option"
            aria-selected={selected}
            tabIndex={0}
            className={`artwork-picker-card artwork-picker-card--${row.status}${
              selected ? " artwork-picker-card--selected" : ""
            }${hasError ? " artwork-picker-card--invalid" : ""}`}
            onClick={() => onSelectOnly(row.localId)}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                onSelectOnly(row.localId);
              }
            }}
          >
            <button
              type="button"
              className="artwork-picker-checkbox"
              aria-label={selected ? `Bỏ chọn ${row.file.name}` : `Chọn thêm ${row.file.name}`}
              aria-pressed={selected}
              onClick={(e) => {
                e.stopPropagation();
                onToggleSelect(row.localId);
              }}
            >
              {selected && <Check size={13} strokeWidth={3} />}
            </button>

            <div className="artwork-picker-thumb">
              <img src={row.localPreview} alt={row.file.name} loading="lazy" />
              <ArtworkPickerStatusBadge row={row} />
            </div>

            <p className="artwork-picker-name" title={row.values.title || row.file.name}>
              {row.values.title || row.file.name}
            </p>

            {!locked && (
              <button
                type="button"
                className="artwork-picker-remove"
                aria-label={`Bỏ ảnh ${row.file.name}`}
                disabled={disabled}
                onClick={(e) => {
                  e.stopPropagation();
                  onRemoveRow(row.localId);
                }}
              >
                <X size={13} />
              </button>
            )}
          </div>
        );
      })}
    </div>
  );
}

function ArtworkPickerStatusBadge({ row }: { row: BulkRow }) {
  switch (row.status) {
    case "saving":
      return (
        <span className="artwork-picker-badge artwork-picker-badge--saving" title="Đang lưu">
          <Loader2 size={12} className="spin" />
        </span>
      );
    case "done":
      return (
        <span className="artwork-picker-badge artwork-picker-badge--done" title="Đã lưu">
          <CheckCircle2 size={12} />
        </span>
      );
    case "error":
      return (
        <span className="artwork-picker-badge artwork-picker-badge--error" title={row.error || "Lỗi"}>
          <AlertCircle size={12} />
        </span>
      );
    default:
      return null;
  }
}
