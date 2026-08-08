type ProgressRingProps = {
  percent: number;
  /** e.g. "lưu" | "thu nhỏ" */
  modeLabel?: string;
};

export function ProgressRing({ percent, modeLabel = "lưu" }: ProgressRingProps) {
  const size = 96;
  const stroke = 6;
  const radius = (size - stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const clamped = Math.max(0, Math.min(100, Math.round(percent)));
  const offset = circumference - (clamped / 100) * circumference;

  return (
    <div
      className="dropzone-inner"
      role="progressbar"
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={clamped}
      aria-label={modeLabel === "thu nhỏ" ? "Tiến độ thu nhỏ ảnh" : "Tiến độ lưu ảnh"}
    >
      <svg className="progress-ring" width={size} height={size} viewBox={`0 0 ${size} ${size}`}>
        <circle className="track" cx={size / 2} cy={size / 2} r={radius} />
        <circle
          className="value"
          cx={size / 2}
          cy={size / 2}
          r={radius}
          strokeDasharray={circumference}
          strokeDashoffset={offset}
        />
      </svg>
      <div className="progress-label">{clamped}%</div>
      <p className="progress-copy" aria-live="polite">
        {clamped < 100 ? `Đang ${modeLabel}…` : "Xong rồi!"}
      </p>
    </div>
  );
}
