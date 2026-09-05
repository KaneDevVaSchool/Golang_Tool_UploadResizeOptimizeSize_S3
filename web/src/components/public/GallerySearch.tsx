import { Search, X } from "lucide-react";

/**
 * Ô tìm trên /phong-trien-lam — khớp title hoặc tên học sinh (cùng hợp đồng
 * LIKE với API public / admin).
 */
export function GallerySearch({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <form className="gallery-search" role="search" onSubmit={(event) => event.preventDefault()}>
      <label className="gallery-search-field">
        <Search size={18} strokeWidth={1.75} aria-hidden />
        <input
          type="search"
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder="Tìm tranh của con — nhập tên học sinh hoặc tên tác phẩm…"
          aria-label="Tìm tác phẩm hoặc học sinh"
          maxLength={80}
          autoComplete="off"
          enterKeyHint="search"
        />
        {value ? (
          <button type="button" className="gallery-search-clear" onClick={() => onChange("")} aria-label="Xóa tìm kiếm">
            <X size={16} strokeWidth={1.75} />
          </button>
        ) : null}
      </label>
    </form>
  );
}
