import { Plus } from "lucide-react";
import { useMemo, useState } from "react";
import type { Award } from "../../lib/artworkApi";
import type { GradeLevel, School } from "../../lib/artworkApi";
import type { TopicCategory } from "../../lib/topicCategoryApi";
import { TopicCategoryQuickCreateModal } from "./TopicCategoryQuickCreateModal";

export type ArtworkMetaFormValues = {
  title: string;
  studentName: string;
  schoolId: number | "";
  educationLevel: "primary" | "secondary" | "";
  gradeLevelId: number | "";
  topicCategoryId: number | "";
  className: string;
  /** Một tác phẩm có thể nhận nhiều giải cùng lúc (giải chính + Đặc biệt phụ). */
  awardIds: number[];
};

export const EMPTY_META_FORM_VALUES: ArtworkMetaFormValues = {
  title: "",
  studentName: "",
  schoolId: "",
  educationLevel: "",
  gradeLevelId: "",
  topicCategoryId: "",
  className: "",
  awardIds: [],
};

/** Field bắt buộc phải điền - dùng chung để validate và để quyết định nơi
 * nào hiện dấu * bắt buộc. Giữ một danh sách duy nhất tránh lệch giữa UI
 * (dấu *) và logic (isMetaFormValid) như trước đây. */
export type RequiredMetaField = "title" | "studentName" | "schoolId" | "educationLevel" | "gradeLevelId";

export type MetaFormErrors = Partial<Record<RequiredMetaField, string>>;

/**
 * Validate toàn bộ form, trả về lỗi theo từng field (rỗng = hợp lệ). Tách
 * riêng khỏi isMetaFormValid (giữ lại cho chỗ chỉ cần biết đúng/sai, như
 * điều kiện disable nút Lưu) vì UI cần biết LỖI GÌ ở TRƯỜNG NÀO để hiện
 * ngay cạnh ô nhập, không chỉ một thông báo chung chung.
 */
export function validateMetaForm(values: ArtworkMetaFormValues): MetaFormErrors {
  const errors: MetaFormErrors = {};
  if (!values.title.trim()) errors.title = "Bắt buộc nhập tên tác phẩm.";
  if (!values.studentName.trim()) errors.studentName = "Bắt buộc nhập tên học sinh.";
  if (!values.schoolId) errors.schoolId = "Bắt buộc chọn cơ sở.";
  if (!values.educationLevel) errors.educationLevel = "Bắt buộc chọn cấp học.";
  if (!values.gradeLevelId) errors.gradeLevelId = "Bắt buộc chọn khối lớp.";
  return errors;
}

export function isMetaFormValid(values: ArtworkMetaFormValues): boolean {
  return Object.keys(validateMetaForm(values)).length === 0;
}

/** Dấu * bắt buộc, tách riêng để mọi form trong hệ thống dùng đúng một kiểu
 * (màu, khoảng cách) thay vì mỗi nơi tự viết `<span style="color:red">*</span>`. */
export function RequiredMark() {
  return (
    <span className="form-required-mark" aria-hidden="true">
      {" "}
      *
    </span>
  );
}

/**
 * Form nhập metadata 1 tác phẩm - dùng chung giữa chế độ edit sau bulk
 * upload (ArtworksUploadPage) và modal edit trong danh sách
 * (ArtworkEditModal). Khối lớp chọn theo 2 cấp: chọn "Tiểu học"/"Trung học"
 * trước để lọc select khối lớp tương ứng (1-5 hoặc 6-12), đúng yêu cầu phân
 * loại 2 cấp. Nhóm chủ đề và danh sách giải cũng lọc theo cấp học/khối khi
 * áp dụng, vì thể lệ Tiểu học và THCS-THPT dùng bộ nhóm chủ đề khác nhau.
 *
 * Lỗi bắt buộc hiện ngay khi rời khỏi ô (blur) chứ không đợi submit - field
 * nào chưa "touched" thì chưa tô đỏ, tránh trang đỏ lòm ngay khi vừa mở
 * form trống. `showAllErrors` (truyền từ nơi gọi khi bấm Lưu mà form còn
 * lỗi) buộc hiện toàn bộ để người dùng không phải rà tay từng ô.
 */
export function ArtworkMetaForm({
  values,
  onChange,
  schools,
  gradeLevels,
  topicCategories,
  onTopicCategoryCreated,
  awards,
  disabled,
  showAllErrors,
}: {
  values: ArtworkMetaFormValues;
  onChange: (next: ArtworkMetaFormValues) => void;
  schools: School[];
  gradeLevels: GradeLevel[];
  topicCategories: TopicCategory[];
  /** Nơi gọi giữ danh sách topicCategories (state ở trang cha) - truyền hàm
   * này để nút "+" cạnh dropdown ghi nhóm chủ đề vừa tạo nhanh vào đúng chỗ
   * đó, thay vì ArtworkMetaForm tự giữ một bản sao lệch khỏi nguồn sự thật.
   * Không bắt buộc: nơi nào chưa truyền thì ẩn hẳn nút "+" (an toàn khi có
   * nơi gọi ArtworkMetaForm chưa kịp cập nhật theo tính năng này). */
  onTopicCategoryCreated?: (category: TopicCategory) => void;
  awards: Award[];
  disabled?: boolean;
  /** Bắt buộc hiện mọi lỗi ngay cả field chưa bị chạm - dùng khi submit thất bại. */
  showAllErrors?: boolean;
}) {
  const [touched, setTouched] = useState<Partial<Record<RequiredMetaField, boolean>>>({});
  const [quickCreateOpen, setQuickCreateOpen] = useState(false);
  const errors = useMemo(() => validateMetaForm(values), [values]);

  function errorFor(field: RequiredMetaField): string | undefined {
    if (!showAllErrors && !touched[field]) return undefined;
    return errors[field];
  }

  function touch(field: RequiredMetaField) {
    setTouched((prev) => (prev[field] ? prev : { ...prev, [field]: true }));
  }

  const filteredGrades = useMemo(
    () => gradeLevels.filter((g) => g.education_level === values.educationLevel),
    [gradeLevels, values.educationLevel],
  );

  // Nhóm chủ đề không gắn cấp học (education_level = null) áp dụng cho mọi
  // cấp - hiện luôn kèm với nhóm khớp đúng cấp học đang chọn.
  const filteredTopics = useMemo(
    () =>
      topicCategories.filter(
        (t) => !t.education_level || t.education_level === values.educationLevel,
      ),
    [topicCategories, values.educationLevel],
  );

  // Giải chỉ hiện những giải dùng chung toàn hệ thống (grade_level_id rỗng)
  // hoặc đúng khối lớp tác phẩm đang chọn - tránh admin gán nhầm giải của
  // khối khác cho tác phẩm này.
  const filteredAwards = useMemo(
    () =>
      awards.filter(
        (a) => !a.grade_level_id || (values.gradeLevelId !== "" && a.grade_level_id === values.gradeLevelId),
      ),
    [awards, values.gradeLevelId],
  );

  function set<K extends keyof ArtworkMetaFormValues>(key: K, value: ArtworkMetaFormValues[K]) {
    onChange({ ...values, [key]: value });
  }

  function toggleAward(id: number) {
    const next = values.awardIds.includes(id)
      ? values.awardIds.filter((a) => a !== id)
      : [...values.awardIds, id];
    set("awardIds", next);
  }

  const titleError = errorFor("title");
  const studentError = errorFor("studentName");
  const schoolError = errorFor("schoolId");
  const levelError = errorFor("educationLevel");
  const gradeError = errorFor("gradeLevelId");

  return (
    <div className="artwork-meta-form">
      <div className={`form-field form-field--full${titleError ? " form-field--error" : ""}`}>
        <label htmlFor="artwork-title">
          Tên tác phẩm
          <RequiredMark />
        </label>
        <input
          id="artwork-title"
          type="text"
          placeholder="Ví dụ: Ngôi trường mơ ước"
          value={values.title}
          disabled={disabled}
          aria-required="true"
          aria-invalid={titleError ? "true" : undefined}
          aria-describedby={titleError ? "artwork-title-error" : undefined}
          onChange={(e) => set("title", e.target.value)}
          onBlur={() => touch("title")}
        />
        {titleError && (
          <p className="form-field-error-text" id="artwork-title-error">
            {titleError}
          </p>
        )}
      </div>

      <div className={`form-field form-field--full${studentError ? " form-field--error" : ""}`}>
        <label htmlFor="artwork-student">
          Học sinh sáng tác
          <RequiredMark />
        </label>
        <input
          id="artwork-student"
          type="text"
          placeholder="Họ và tên học sinh"
          value={values.studentName}
          disabled={disabled}
          aria-required="true"
          aria-invalid={studentError ? "true" : undefined}
          aria-describedby={studentError ? "artwork-student-error" : undefined}
          onChange={(e) => set("studentName", e.target.value)}
          onBlur={() => touch("studentName")}
        />
        {studentError && (
          <p className="form-field-error-text" id="artwork-student-error">
            {studentError}
          </p>
        )}
      </div>

      <div className={`form-field${schoolError ? " form-field--error" : ""}`}>
        <label htmlFor="artwork-school">
          Cơ sở trực thuộc
          <RequiredMark />
        </label>
        <select
          id="artwork-school"
          value={values.schoolId}
          disabled={disabled}
          aria-required="true"
          aria-invalid={schoolError ? "true" : undefined}
          aria-describedby={schoolError ? "artwork-school-error" : undefined}
          onChange={(e) => set("schoolId", e.target.value ? Number(e.target.value) : "")}
          onBlur={() => touch("schoolId")}
        >
          <option value="">— Chọn cơ sở —</option>
          {schools.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </select>
        {schoolError && (
          <p className="form-field-error-text" id="artwork-school-error">
            {schoolError}
          </p>
        )}
      </div>

      <div className="form-field">
        <label htmlFor="artwork-class">Lớp (tuỳ chọn)</label>
        <input
          id="artwork-class"
          type="text"
          placeholder="Ví dụ: 3A1"
          value={values.className}
          disabled={disabled}
          onChange={(e) => set("className", e.target.value)}
        />
      </div>

      <div className={`form-field${levelError ? " form-field--error" : ""}`}>
        <label htmlFor="artwork-level">
          Phân loại
          <RequiredMark />
        </label>
        <select
          id="artwork-level"
          value={values.educationLevel}
          disabled={disabled}
          aria-required="true"
          aria-invalid={levelError ? "true" : undefined}
          aria-describedby={levelError ? "artwork-level-error" : undefined}
          onChange={(e) => {
            const level = e.target.value as ArtworkMetaFormValues["educationLevel"];
            onChange({ ...values, educationLevel: level, gradeLevelId: "", topicCategoryId: "" });
          }}
          onBlur={() => touch("educationLevel")}
        >
          <option value="">— Chọn cấp học —</option>
          <option value="primary">Tiểu học</option>
          <option value="secondary">Trung học (THCS &amp; THPT)</option>
        </select>
        {levelError && (
          <p className="form-field-error-text" id="artwork-level-error">
            {levelError}
          </p>
        )}
      </div>

      <div className={`form-field${gradeError ? " form-field--error" : ""}`}>
        <label htmlFor="artwork-grade">
          Khối lớp
          <RequiredMark />
        </label>
        <select
          id="artwork-grade"
          value={values.gradeLevelId}
          disabled={disabled || !values.educationLevel}
          aria-required="true"
          aria-invalid={gradeError ? "true" : undefined}
          aria-describedby={gradeError ? "artwork-grade-error" : undefined}
          onChange={(e) => set("gradeLevelId", e.target.value ? Number(e.target.value) : "")}
          onBlur={() => touch("gradeLevelId")}
        >
          <option value="">{values.educationLevel ? "— Chọn khối —" : "Chọn cấp học trước"}</option>
          {filteredGrades.map((g) => (
            <option key={g.id} value={g.id}>
              {g.label}
            </option>
          ))}
        </select>
        {gradeError && (
          <p className="form-field-error-text" id="artwork-grade-error">
            {gradeError}
          </p>
        )}
      </div>

      <div className="form-field form-field--full">
        <label htmlFor="artwork-topic" className="form-field-label-row">
          Nhóm chủ đề sáng tạo (tuỳ chọn)
          {onTopicCategoryCreated && (
            <button
              type="button"
              className="form-field-inline-add-btn"
              title="Thêm nhanh nhóm chủ đề mới"
              aria-label="Thêm nhanh nhóm chủ đề mới"
              disabled={disabled}
              onClick={() => setQuickCreateOpen(true)}
            >
              <Plus size={14} />
            </button>
          )}
        </label>
        <select
          id="artwork-topic"
          value={values.topicCategoryId}
          disabled={disabled}
          onChange={(e) => set("topicCategoryId", e.target.value ? Number(e.target.value) : "")}
        >
          <option value="">— Chưa chọn nhóm chủ đề —</option>
          {filteredTopics.map((t) => (
            <option key={t.id} value={t.id}>
              {t.name}
            </option>
          ))}
        </select>
      </div>

      {onTopicCategoryCreated && (
        <TopicCategoryQuickCreateModal
          open={quickCreateOpen}
          defaultEducationLevel={values.educationLevel}
          onClose={() => setQuickCreateOpen(false)}
          onCreated={(category) => {
            onTopicCategoryCreated(category);
            set("topicCategoryId", category.id);
            setQuickCreateOpen(false);
          }}
        />
      )}

      <div className="form-field form-field--full">
        <label>Giải thưởng (nếu có)</label>
        {filteredAwards.length === 0 ? (
          <p className="form-field-hint">Chưa có giải thưởng phù hợp khối lớp này.</p>
        ) : (
          <div className="artwork-award-checklist">
            {filteredAwards.map((a) => (
              <label key={a.id} className="artwork-award-checkbox">
                <input
                  type="checkbox"
                  checked={values.awardIds.includes(a.id)}
                  disabled={disabled}
                  onChange={() => toggleAward(a.id)}
                />
                <span className="artwork-award-swatch" style={{ background: a.color_hex }} />
                {a.name}
              </label>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
