type ProgressRingProps = {
  percent: number;
};

export function ProgressRing({ percent }: ProgressRingProps) {
  const size = 120;
  const stroke = 6;
  const radius = (size - stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (percent / 100) * circumference;

  return (
    <div className="dropzone-inner">
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
      <div className="progress-label">{percent}%</div>
      <p>{percent < 100 ? "Uploading chunks into the pipeline…" : "Finalizing…"}</p>
    </div>
  );
}
