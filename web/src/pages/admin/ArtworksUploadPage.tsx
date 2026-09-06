import { AlertTriangle, ArrowLeft, Loader2, UploadCloud, Users } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AdminDropzone } from "../../components/admin/AdminDropzone";
import { AdminPageHeader } from "../../components/admin/AdminPageHeader";
import { ArtworkPickerGrid } from "../../components/admin/ArtworkPickerGrid";
import type { BulkRow } from "../../components/admin/artworkUploadTypes";
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
  type GradeLevel,
  type School,
} from "../../lib/artworkApi";
import { fetchAwards } from "../../lib/awardApi";
import { fetchTopicCategories, type TopicCategory } from "../../lib/topicCategoryApi";
import { toast } from "../../lib/toastBus";

function makeLocalId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
}

function titleFromFileName(name: string): string {
  return name.replace(/\.[^.]+$/, "");
}

function makeRow(file: File): BulkRow {
  return {
    localId: makeLocalId(),
    file,
    localPreview: URL.createObjectURL(file),
    values: { ...EMPTY_META_FORM_VALUES, title: titleFromFileName(file.name) },
    status: "pending",
  };
}

/**
 * Có field nào điền trong `patch` thì merge vào `values`; field để trống ở
 * `patch` giữ nguyên giá trị hiện có của từng ảnh. Dùng cho panel sửa hàng
 * loạt khi chọn từ 2 ảnh trở lên - xem comment ở BulkEditPanel bên dưới.
 * `awardIds` ghi đè toàn bộ danh sách (không cộng dồn) khi patch có chọn ít
 * nhất 1 giải - đơn giản và dễ đoán hơn "hợp nhất 2 mảng".
 */
function applyPatch(values: ArtworkMetaFormValues, patch: ArtworkMetaFormValues): ArtworkMetaFormValues {
  const next = { ...values };
  if (patch.title.trim()) next.title = patch.title;
  if (patch.studentName.trim()) next.studentName = patch.studentName;
  if (patch.schoolId !== "") next.schoolId = patch.schoolId;
  if (patch.educationLevel !== "") {
    next.educationLevel = patch.educationLevel;
    // Đổi cấp học hàng loạt mà không kèm khối lớp mới thì khối lớp cũ (thuộc
    // cấp khác) không còn hợp lệ - xoá theo, giống hành vi ArtworkMetaForm
    // khi đổi educationLevel tại chỗ.
    if (patch.gradeLevelId === "") next.gradeLevelId = "";
  }
  if (patch.gradeLevelId !== "") next.gradeLevelId = patch.gradeLevelId;
  if (patch.className.trim()) next.className = patch.className;
  if (patch.topicCategoryId !== "") next.topicCategoryId = patch.topicCategoryId;
  if (patch.awardIds.length > 0) next.awardIds = patch.awardIds;
  return next;
}

/**
 * Trang tải tác phẩm lên - một luồng duy nhất, không còn tách "1 ảnh"/"nhiều
 * ảnh" bằng tab như bản trước: chọn 1 hay nhiều ảnh đều vào chung 1 giao
 * diện (lưới thẻ ảnh trái + panel sửa phải). Chọn đúng 1 thẻ thì panel là
 * form đầy đủ cho riêng ảnh đó; chọn từ 2 thẻ trở lên thì panel chuyển sang
 * "áp dụng cho N ảnh" - field nào điền thì ghi đè lên mọi ảnh đang chọn,
 * field để trống giữ nguyên (xem applyPatch).
 *
 * Khác với thiết kế cũ (upload S3 ngay khi chọn ảnh rồi mới điền form), ở
 * đây ảnh chỉ preview tại client (objectURL) - CHƯA chạm S3. Việc upload +
 * tạo bản ghi artwork chỉ chạy khi bấm "Lưu tất cả", theo đúng yêu cầu
 * nghiệp vụ: người dùng muốn xem lại toàn bộ trước khi bất kỳ thứ gì rời
 * khỏi máy. Đánh đổi đã biết: lỗi mạng ngay lúc Lưu có thể buộc nhập lại -
 * xem docs/detail_design/03-artwork-domain.md mục "Vì sao upload chia hai
 * bước" để rõ lý do và đánh đổi.
 */
export default function ArtworksUploadPage() {
  const navigate = useNavigate();
  const [schools, setSchools] = useState<School[]>([]);
  const [gradeLevels, setGradeLevels] = useState<GradeLevel[]>([]);
  const [topicCategories, setTopicCategories] = useState<TopicCategory[]>([]);
  const [awards, setAwards] = useState<Award[]>([]);

  const [rows, setRows] = useState<BulkRow[]>([]);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [saving, setSaving] = useState(false);
  const [showErrors, setShowErrors] = useState(false);
  // localId của các thẻ cần buộc hiện lỗi ngay (bấm "Lưu tất cả" mà thẻ đó
  // chưa điền đủ) - không bật toàn bộ lưới để thẻ đã hợp lệ không bị tô đỏ
  // theo.
  const [errorRowIds, setErrorRowIds] = useState<Set<string>>(new Set());
  const inputRef = useRef<HTMLInputElement>(null);

  const rowsRef = useRef(rows);
  rowsRef.current = rows;

  useEffect(() => {
    Promise.all([fetchSchools(), fetchGradeLevels(), fetchTopicCategories(true), fetchAwards(true)])
      .then(([s, g, t, a]) => {
        setSchools(s ?? []);
        setGradeLevels(g ?? []);
        setTopicCategories(t ?? []);
        setAwards(a ?? []);
      })
      .catch((err: unknown) => toast.error(err instanceof Error ? err.message : "Không tải được dữ liệu nền"));
  }, []);

  // Dọn toàn bộ objectURL khi rời trang.
  useEffect(() => {
    return () => {
      rowsRef.current.forEach((r) => URL.revokeObjectURL(r.localPreview));
    };
  }, []);

  function handleFilesSelected(files: File[]) {
    if (!files.length) return;
    const newRows = files.map(makeRow);
    setRows((prev) => [...prev, ...newRows]);
    // Ảnh vừa thêm được tự chọn - panel bên phải luôn có gì đó để hiện ngay,
    // không bắt người dùng tự đi tìm thẻ để bấm vào.
    setSelectedIds(new Set(newRows.map((r) => r.localId)));
  }

  function selectOnly(localId: string) {
    setSelectedIds(new Set([localId]));
  }

  function toggleSelect(localId: string) {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(localId)) next.delete(localId);
      else next.add(localId);
      return next;
    });
  }

  function updateRowValues(localId: string, values: ArtworkMetaFormValues) {
    setRows((prev) => prev.map((r) => (r.localId === localId ? { ...r, values } : r)));
    setErrorRowIds((prev) => {
      if (!prev.has(localId)) return prev;
      const next = new Set(prev);
      next.delete(localId);
      return next;
    });
  }

  function applyBulkPatch(patch: ArtworkMetaFormValues) {
    setRows((prev) =>
      prev.map((r) => (selectedIds.has(r.localId) ? { ...r, values: applyPatch(r.values, patch) } : r)),
    );
    setErrorRowIds((prev) => {
      if (prev.size === 0) return prev;
      const next = new Set(prev);
      selectedIds.forEach((id) => next.delete(id));
      return next;
    });
    toast.success(`Đã áp dụng cho ${selectedIds.size} ảnh.`);
  }

  function removeRow(localId: string) {
    setRows((prev) => {
      const target = prev.find((r) => r.localId === localId);
      if (target) URL.revokeObjectURL(target.localPreview);
      return prev.filter((r) => r.localId !== localId);
    });
    setSelectedIds((prev) => {
      if (!prev.has(localId)) return prev;
      const next = new Set(prev);
      next.delete(localId);
      return next;
    });
  }

  function clearAll() {
    rows.forEach((r) => URL.revokeObjectURL(r.localPreview));
    setRows([]);
    setSelectedIds(new Set());
  }

  async function saveAll() {
    const toSave = rows.filter((r) => r.status !== "done");
    if (!toSave.length) return;
    const invalid = toSave.filter((r) => !isMetaFormValid(r.values));
    if (invalid.length > 0) {
      setErrorRowIds(new Set(invalid.map((r) => r.localId)));
      toast.error(`Còn ${invalid.length} ảnh chưa điền đủ thông tin bắt buộc - thẻ lỗi đã tô đỏ trong lưới.`);
      return;
    }
    setErrorRowIds(new Set());

    setSaving(true);
    let successCount = 0;
    let failCount = 0;

    // Upload từng ảnh một thay vì gộp 1 request nhiều file: người dùng đã
    // điền xong toàn bộ trước khi bấm Lưu, nên cần biết CHÍNH XÁC ảnh nào
    // lỗi để sửa lại tại chỗ - tách request giúp cập nhật trạng thái từng
    // thẻ độc lập, không phải chờ cả lô xong mới biết kết quả.
    for (const row of toSave) {
      setRows((prev) => prev.map((r) => (r.localId === row.localId ? { ...r, status: "saving" } : r)));
      try {
        const uploadResult = await bulkUploadArtworks([row.file]);
        const item = uploadResult.items[0];
        if (!item || item.error || !item.s3_key || !item.s3_url) {
          throw new Error(item?.error || "Tải ảnh lên thất bại");
        }
        await createArtwork({
          title: row.values.title,
          student_name: row.values.studentName,
          school_id: Number(row.values.schoolId),
          grade_level_id: Number(row.values.gradeLevelId),
          topic_category_id: row.values.topicCategoryId ? Number(row.values.topicCategoryId) : undefined,
          class_name: row.values.className || undefined,
          s3_key: item.s3_key,
          s3_url: item.s3_url,
          file_size: item.file_size ?? row.file.size,
          width: item.width,
          height: item.height,
          variants: item.variants,
          award_ids: row.values.awardIds,
        });
        successCount++;
        setRows((prev) => prev.map((r) => (r.localId === row.localId ? { ...r, status: "done" } : r)));
      } catch (err) {
        failCount++;
        const message = err instanceof Error ? err.message : "Lưu thất bại";
        setRows((prev) =>
          prev.map((r) => (r.localId === row.localId ? { ...r, status: "error", error: message } : r)),
        );
      }
    }

    setSaving(false);
    if (failCount === 0) {
      toast.success(`Đã lưu ${successCount} tác phẩm thành công.`);
      navigate("/admin/artworks");
    } else {
      toast.warning(`Đã lưu ${successCount} tác phẩm, còn ${failCount} tác phẩm lỗi - sửa lại rồi lưu tiếp.`);
    }
  }

  const doneCount = rows.filter((r) => r.status === "done").length;
  const selectedRows = useMemo(() => rows.filter((r) => selectedIds.has(r.localId)), [rows, selectedIds]);

  return (
    <div className="artworks-upload-page">
      <AdminPageHeader
        title="Đưa tác phẩm lên"
        subtitle={rows.length > 0 ? `Đã lưu ${doneCount}/${rows.length} tác phẩm` : undefined}
        primaryAction={{
          label: "Quay lại thư viện",
          icon: ArrowLeft,
          variant: "ghost",
          onClick: () => navigate("/admin/artworks"),
        }}
      />

      <div className="artworks-upload-dropzone-wrap">
        <AdminDropzone busy={saving} inputRef={inputRef} onSelect={handleFilesSelected} />
      </div>

      {rows.length > 0 && (
        <>
          <div className="artwork-picker-layout">
            <ArtworkPickerGrid
              rows={rows}
              selectedIds={selectedIds}
              disabled={saving}
              errorRowIds={errorRowIds}
              onSelectOnly={selectOnly}
              onToggleSelect={toggleSelect}
              onRemoveRow={removeRow}
            />

            <div className="artwork-edit-panel">
              {selectedRows.length === 0 && (
                <p className="artwork-edit-panel-empty">Chọn một ảnh trong lưới để điền thông tin.</p>
              )}
              {selectedRows.length === 1 && (
                <SingleEditPanel
                  row={selectedRows[0]}
                  schools={schools}
                  gradeLevels={gradeLevels}
                  topicCategories={topicCategories}
                  onTopicCategoryCreated={(category) => setTopicCategories((prev) => [...prev, category])}
                  awards={awards}
                  disabled={saving || selectedRows[0].status === "saving" || selectedRows[0].status === "done"}
                  showAllErrors={showErrors}
                  onChange={(values) => updateRowValues(selectedRows[0].localId, values)}
                />
              )}
              {selectedRows.length > 1 && (
                <BulkEditPanel
                  count={selectedRows.length}
                  schools={schools}
                  gradeLevels={gradeLevels}
                  topicCategories={topicCategories}
                  onTopicCategoryCreated={(category) => setTopicCategories((prev) => [...prev, category])}
                  awards={awards}
                  disabled={saving}
                  onApply={applyBulkPatch}
                />
              )}
            </div>
          </div>

          <div className="artworks-upload-footer">
            <span>
              {doneCount}/{rows.length} tác phẩm đã lưu
            </span>
            <div className="artwork-bulk-footer-actions">
              <button type="button" className="btn btn-ghost" onClick={clearAll} disabled={saving}>
                Xoá tất cả
              </button>
              <button
                type="button"
                className="btn btn-primary"
                onClick={() => {
                  setShowErrors(true);
                  saveAll();
                }}
                disabled={saving}
              >
                {saving ? (
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
          </div>
        </>
      )}
    </div>
  );
}

function SingleEditPanel({
  row,
  schools,
  gradeLevels,
  topicCategories,
  onTopicCategoryCreated,
  awards,
  disabled,
  showAllErrors,
  onChange,
}: {
  row: BulkRow;
  schools: School[];
  gradeLevels: GradeLevel[];
  topicCategories: TopicCategory[];
  onTopicCategoryCreated: (category: TopicCategory) => void;
  awards: Award[];
  disabled?: boolean;
  showAllErrors: boolean;
  onChange: (values: ArtworkMetaFormValues) => void;
}) {
  return (
    <>
      <div className="artwork-edit-panel-header">
        <img src={row.localPreview} alt={row.file.name} className="artwork-edit-panel-thumb" />
        <div className="artwork-edit-panel-heading">
          <strong>{row.file.name}</strong>
          {row.error && (
            <p className="artworks-edit-error">
              <AlertTriangle size={14} /> {row.error}
            </p>
          )}
        </div>
      </div>
      <ArtworkMetaForm
        values={row.values}
        onChange={onChange}
        schools={schools}
        gradeLevels={gradeLevels}
        topicCategories={topicCategories}
        onTopicCategoryCreated={onTopicCategoryCreated}
        awards={awards}
        disabled={disabled}
        showAllErrors={showAllErrors}
      />
    </>
  );
}

function BulkEditPanel({
  count,
  schools,
  gradeLevels,
  topicCategories,
  onTopicCategoryCreated,
  awards,
  disabled,
  onApply,
}: {
  count: number;
  schools: School[];
  gradeLevels: GradeLevel[];
  topicCategories: TopicCategory[];
  onTopicCategoryCreated: (category: TopicCategory) => void;
  awards: Award[];
  disabled?: boolean;
  onApply: (patch: ArtworkMetaFormValues) => void;
}) {
  const [patch, setPatch] = useState<ArtworkMetaFormValues>(EMPTY_META_FORM_VALUES);
  const hasAnyField =
    patch.title.trim() ||
    patch.studentName.trim() ||
    patch.schoolId !== "" ||
    patch.educationLevel !== "" ||
    patch.gradeLevelId !== "" ||
    patch.className.trim() ||
    patch.topicCategoryId !== "" ||
    patch.awardIds.length > 0;

  return (
    <>
      <div className="artwork-bulk-edit-banner">
        <Users size={16} />
        <span>
          Áp dụng cho <strong>{count}</strong> ảnh đã chọn — để trống nghĩa là giữ nguyên giá trị riêng của
          từng ảnh.
        </span>
      </div>
      {/* Bulk-edit không bắt buộc field nào - mỗi field điền thì mới ghi đè,
          nên KHÔNG bật showAllErrors và không kiểm tra isMetaFormValid ở
          đây. Tái dùng nguyên ArtworkMetaForm để không phải viết lại UI
          input/select, giữ đúng một chỗ định nghĩa field (06-frontend.md
          mục 11). */}
      <ArtworkMetaForm
        values={patch}
        onChange={setPatch}
        schools={schools}
        gradeLevels={gradeLevels}
        topicCategories={topicCategories}
        onTopicCategoryCreated={onTopicCategoryCreated}
        awards={awards}
        disabled={disabled}
        showAllErrors={false}
      />
      <button
        type="button"
        className="btn btn-primary artwork-bulk-edit-apply"
        disabled={disabled || !hasAnyField}
        onClick={() => {
          onApply(patch);
          setPatch(EMPTY_META_FORM_VALUES);
        }}
      >
        Áp dụng cho {count} ảnh
      </button>
    </>
  );
}
