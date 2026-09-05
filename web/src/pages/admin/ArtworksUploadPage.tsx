import { ArrowLeft, CheckCircle2, Loader2, UploadCloud } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Dropzone } from "../../components/Dropzone";
import { AdminPageHeader } from "../../components/admin/AdminPageHeader";
import {
  ArtworkMetaForm,
  EMPTY_META_FORM_VALUES,
  isMetaFormValid,
  type ArtworkMetaFormValues,
} from "../../components/admin/ArtworkMetaForm";
import {
  bulkUploadArtworks,
  createArtwork,
  fetchGradeLevels,
  fetchSchools,
  type Award,
  type BulkUploadItem,
  type GradeLevel,
  type School,
} from "../../lib/artworkApi";
import { fetchAwards } from "../../lib/awardApi";
import { toast } from "../../lib/toastBus";

type Stage = "select" | "uploading" | "edit" | "saving";

type EditableItem = BulkUploadItem & {
  // localId luôn duy nhất (sinh ở FE) - dùng làm key React/state thay vì
  // temp_key, vì temp_key backend trả về là tên file đã sanitize và có thể
  // trùng nhau nếu user chọn 2 file cùng tên trong 1 lượt.
  localId: string;
  file: File;
  localPreview: string;
  values: ArtworkMetaFormValues;
  saveStatus: "pending" | "saving" | "done" | "error";
  saveError?: string;
};

function makeLocalId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

/**
 * Trang upload hàng loạt tác phẩm - 4 bước:
 * 1. Chọn nhiều ảnh (tái dùng Dropzone.tsx với multiple).
 * 2. Upload song song lên S3 qua bulk-upload endpoint (chưa ghi DB).
 * 3. Chế độ "sửa từng ảnh": thumbnail trái, form phải (ArtworkMetaForm).
 * 4. "Lưu tất cả" gọi POST /api/v1/admin/artworks cho từng item.
 */
export default function ArtworksUploadPage() {
  const navigate = useNavigate();
  const [stage, setStage] = useState<Stage>("select");
  const [items, setItems] = useState<EditableItem[]>([]);
  const [activeId, setActiveId] = useState<string>("");
  const [schools, setSchools] = useState<School[]>([]);
  const [gradeLevels, setGradeLevels] = useState<GradeLevel[]>([]);
  const [awards, setAwards] = useState<Award[]>([]);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    Promise.all([fetchSchools(), fetchGradeLevels(), fetchAwards(true)])
      .then(([s, g, a]) => {
        setSchools(s ?? []);
        setGradeLevels(g ?? []);
        setAwards(a ?? []);
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : "Không tải được dữ liệu nền"));
  }, []);

  useEffect(() => {
    return () => {
      items.forEach((item) => URL.revokeObjectURL(item.localPreview));
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleSelect(files: File[]) {
    if (!files.length) return;
    setStage("uploading");

    try {
      const result = await bulkUploadArtworks(files);

      // Backend giữ nguyên thứ tự input trong mảng kết quả (results[idx] =
      // item theo đúng index gốc, xem ArtworkService.BulkUploadToS3) - khớp
      // theo index, KHÔNG theo tên, vì temp_key có thể trùng nếu 2 file
      // cùng tên được chọn trong 1 lượt.
      const editable: EditableItem[] = files.map((file, index) => {
        const uploadResult: BulkUploadItem = result.items[index] ?? {
          temp_key: file.name,
          file_name: file.name,
          error: "Không tìm thấy kết quả upload cho file này",
        };
        return {
          ...uploadResult,
          localId: makeLocalId(),
          file,
          localPreview: URL.createObjectURL(file),
          values: { ...EMPTY_META_FORM_VALUES, title: file.name.replace(/\.[^.]+$/, "") },
          saveStatus: "pending",
        };
      });

      if (result.open_errors.length > 0) {
        toast.warning(`${result.open_errors.length} file không hợp lệ đã bị bỏ qua.`);
      }
      const failedCount = editable.filter((it) => it.error).length;
      if (failedCount > 0) {
        toast.warning(`${failedCount} ảnh tải lên thất bại, có thể thử lại từ đầu.`);
      }
      if (editable.every((it) => it.error)) {
        toast.error("Không có ảnh nào tải lên thành công.");
        setStage("select");
        return;
      }

      setItems(editable);
      setActiveId(editable.find((it) => !it.error)?.localId ?? editable[0].localId);
      setStage("edit");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Tải ảnh lên thất bại, vui lòng thử lại");
      setStage("select");
    }
  }

  function updateActiveValues(values: ArtworkMetaFormValues) {
    setItems((prev) => prev.map((it) => (it.localId === activeId ? { ...it, values } : it)));
  }

  async function saveAll() {
    const toSave = items.filter((it) => !it.error && it.saveStatus !== "done");
    const invalid = toSave.filter((it) => !isMetaFormValid(it.values));
    if (invalid.length > 0) {
      toast.error(`Còn ${invalid.length} tác phẩm chưa điền đủ thông tin bắt buộc.`);
      setActiveId(invalid[0].localId);
      return;
    }

    setStage("saving");
    let successCount = 0;
    let failCount = 0;

    for (const item of toSave) {
      setItems((prev) => prev.map((it) => (it.localId === item.localId ? { ...it, saveStatus: "saving" } : it)));
      try {
        await createArtwork({
          title: item.values.title,
          student_name: item.values.studentName,
          school_id: Number(item.values.schoolId),
          grade_level_id: Number(item.values.gradeLevelId),
          class_name: item.values.className || undefined,
          s3_key: item.s3_key!,
          s3_url: item.s3_url!,
          file_size: item.file_size ?? item.file.size,
          width: item.width,
          height: item.height,
          award_id: item.values.awardId ? Number(item.values.awardId) : null,
        });
        successCount++;
        setItems((prev) => prev.map((it) => (it.localId === item.localId ? { ...it, saveStatus: "done" } : it)));
      } catch (err) {
        failCount++;
        const message = err instanceof Error ? err.message : "Lưu thất bại";
        setItems((prev) =>
          prev.map((it) => (it.localId === item.localId ? { ...it, saveStatus: "error", saveError: message } : it)),
        );
      }
    }

    setStage("edit");
    if (failCount === 0) {
      toast.success(`Đã lưu ${successCount} tác phẩm thành công.`);
      navigate("/admin/artworks");
    } else {
      toast.warning(`Đã lưu ${successCount} tác phẩm, còn ${failCount} tác phẩm lỗi - kiểm tra lại bên dưới.`);
    }
  }

  const active = items.find((it) => it.localId === activeId);
  const validCount = items.filter((it) => !it.error).length;
  const doneCount = items.filter((it) => it.saveStatus === "done").length;

  return (
    <div className="artworks-upload-page">
      <AdminPageHeader
        title="Tải tác phẩm mới"
        subtitle={validCount > 0 ? `${doneCount}/${validCount} đã lưu` : undefined}
        primaryAction={{
          label: "Quay lại danh sách",
          icon: ArrowLeft,
          variant: "ghost",
          onClick: () => navigate("/admin/artworks"),
        }}
      />

      {stage === "select" && (
        <div className="artworks-upload-dropzone-wrap">
          <Dropzone
            busy={false}
            mode="wp"
            accept="image/*"
            inputRef={fileInputRef}
            onSelect={handleSelect}
          />
        </div>
      )}

      {stage === "uploading" && (
        <div className="admin-page-placeholder artworks-upload-loading">
          <Loader2 className="spin" size={28} />
          <p>Đang tải ảnh lên, vui lòng đợi…</p>
        </div>
      )}

      {(stage === "edit" || stage === "saving") && (
        <div className="artworks-edit-layout">
          <aside className="artworks-edit-thumbs">
            {items.map((item) => (
              <button
                key={item.localId}
                type="button"
                className={`artworks-edit-thumb${item.localId === activeId ? " artworks-edit-thumb--active" : ""}`}
                onClick={() => setActiveId(item.localId)}
                disabled={stage === "saving"}
              >
                <img src={item.localPreview} alt={item.file.name} />
                {item.error && <span className="artworks-edit-thumb-badge artworks-edit-thumb-badge--error">Lỗi</span>}
                {item.saveStatus === "done" && (
                  <span className="artworks-edit-thumb-badge artworks-edit-thumb-badge--done">
                    <CheckCircle2 size={12} />
                  </span>
                )}
                {item.saveStatus === "saving" && (
                  <span className="artworks-edit-thumb-badge">
                    <Loader2 size={12} className="spin" />
                  </span>
                )}
                <span className="artworks-edit-thumb-name" title={item.file.name}>
                  {item.values.title || item.file.name}
                </span>
              </button>
            ))}
          </aside>

          <div className="artworks-edit-main">
            {active?.error ? (
              <div className="admin-page-placeholder">
                <p>Ảnh này tải lên thất bại: {active.error}</p>
              </div>
            ) : active ? (
              <>
                <div className="artworks-edit-preview">
                  <img src={active.localPreview} alt={active.file.name} />
                </div>
                <ArtworkMetaForm
                  values={active.values}
                  onChange={updateActiveValues}
                  schools={schools}
                  gradeLevels={gradeLevels}
                  awards={awards}
                  disabled={stage === "saving" || active.saveStatus === "done"}
                />
                {active.saveError && <p className="artworks-edit-error">{active.saveError}</p>}
              </>
            ) : (
              <div className="admin-page-placeholder">Chọn 1 ảnh bên trái để nhập thông tin.</div>
            )}
          </div>
        </div>
      )}

      {(stage === "edit" || stage === "saving") && (
        <div className="artworks-upload-footer">
          <span>
            {doneCount}/{validCount} tác phẩm đã lưu
          </span>
          <button type="button" className="btn btn-primary" onClick={saveAll} disabled={stage === "saving"}>
            {stage === "saving" ? (
              <>
                <Loader2 size={16} className="spin" /> Đang lưu…
              </>
            ) : (
              <>
                <UploadCloud size={16} /> Lưu tất cả
              </>
            )}
          </button>
        </div>
      )}
    </div>
  );
}
