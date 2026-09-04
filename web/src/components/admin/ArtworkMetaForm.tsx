import { useMemo } from "react";
import type { Award } from "../../lib/artworkApi";
import type { GradeLevel, School } from "../../lib/artworkApi";

export type ArtworkMetaFormValues = {
  title: string;
  studentName: string;
  schoolId: number | "";
  educationLevel: "primary" | "secondary" | "";
  gradeLevelId: number | "";
  className: string;
  awardId: number | "";
};

export const EMPTY_META_FORM_VALUES: ArtworkMetaFormValues = {
  title: "",
  studentName: "",
  schoolId: "",
  educationLevel: "",
  gradeLevelId: "",
  className: "",
  awardId: "",
};

/**
 * Form nhập metadata 1 tác phẩm - dùng chung giữa chế độ edit sau bulk
 * upload (ArtworksUploadPage) và modal edit trong danh sách
 * (ArtworkEditModal). Khối lớp chọn theo 2 cấp: chọn "Tiểu học"/"Trung học"
 * trước để lọc select khối lớp tương ứng (1-5 hoặc 6-12), đúng yêu cầu phân
 * loại 2 cấp.
 */
export function ArtworkMetaForm({
  values,
  onChange,
  schools,
  gradeLevels,
  awards,
  disabled,
}: {
  values: ArtworkMetaFormValues;
  onChange: (next: ArtworkMetaFormValues) => void;
  schools: School[];
  gradeLevels: GradeLevel[];
  awards: Award[];
  disabled?: boolean;
}) {
  const filteredGrades = useMemo(
    () => gradeLevels.filter((g) => g.education_level === values.educationLevel),
    [gradeLevels, values.educationLevel],
  );

  function set<K extends keyof ArtworkMetaFormValues>(key: K, value: ArtworkMetaFormValues[K]) {
    onChange({ ...values, [key]: value });
  }

  return (
    <div className="artwork-meta-form">
      <div className="form-field form-field--full">
        <label htmlFor="artwork-title">Tên tác phẩm</label>
        <input
          id="artwork-title"
          type="text"
          placeholder="Ví dụ: Ngôi trường mơ ước"
          value={values.title}
          disabled={disabled}
          onChange={(e) => set("title", e.target.value)}
        />
      </div>

      <div className="form-field form-field--full">
        <label htmlFor="artwork-student">Học sinh sáng tác</label>
        <input
          id="artwork-student"
          type="text"
          placeholder="Họ và tên học sinh"
          value={values.studentName}
          disabled={disabled}
          onChange={(e) => set("studentName", e.target.value)}
        />
      </div>

      <div className="form-field">
        <label htmlFor="artwork-school">Cơ sở trực thuộc</label>
        <select
          id="artwork-school"
          value={values.schoolId}
          disabled={disabled}
          onChange={(e) => set("schoolId", e.target.value ? Number(e.target.value) : "")}
        >
          <option value="">— Chọn cơ sở —</option>
          {schools.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </select>
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

      <div className="form-field">
        <label htmlFor="artwork-level">Phân loại</label>
        <select
          id="artwork-level"
          value={values.educationLevel}
          disabled={disabled}
          onChange={(e) => {
            const level = e.target.value as ArtworkMetaFormValues["educationLevel"];
            onChange({ ...values, educationLevel: level, gradeLevelId: "" });
          }}
        >
          <option value="">— Chọn cấp học —</option>
          <option value="primary">Tiểu học</option>
          <option value="secondary">Trung học (THCS &amp; THPT)</option>
        </select>
      </div>

      <div className="form-field">
        <label htmlFor="artwork-grade">Khối lớp</label>
        <select
          id="artwork-grade"
          value={values.gradeLevelId}
          disabled={disabled || !values.educationLevel}
          onChange={(e) => set("gradeLevelId", e.target.value ? Number(e.target.value) : "")}
        >
          <option value="">{values.educationLevel ? "— Chọn khối —" : "Chọn cấp học trước"}</option>
          {filteredGrades.map((g) => (
            <option key={g.id} value={g.id}>
              {g.label}
            </option>
          ))}
        </select>
      </div>

      <div className="form-field form-field--full">
        <label htmlFor="artwork-award">Giải thưởng (nếu có)</label>
        <select
          id="artwork-award"
          value={values.awardId}
          disabled={disabled}
          onChange={(e) => set("awardId", e.target.value ? Number(e.target.value) : "")}
        >
          <option value="">— Chưa có giải —</option>
          {awards.map((a) => (
            <option key={a.id} value={a.id}>
              {a.name}
            </option>
          ))}
        </select>
      </div>
    </div>
  );
}

export function isMetaFormValid(values: ArtworkMetaFormValues): boolean {
  return Boolean(
    values.title.trim() && values.studentName.trim() && values.schoolId && values.gradeLevelId,
  );
}
