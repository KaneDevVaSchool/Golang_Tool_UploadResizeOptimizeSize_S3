type Region = "all" | "saigon" | "cantho" | "vungtau";

const TABS: { key: Region; label: string }[] = [
  { key: "all", label: "Tất cả" },
  { key: "saigon", label: "Sài Gòn" },
  { key: "cantho", label: "Cần Thơ" },
  { key: "vungtau", label: "Vũng Tàu" },
];

export function RegionTabs({ value, onChange }: { value: Region; onChange: (region: Region) => void }) {
  return (
    <div className="region-tabs" role="tablist" aria-label="Chọn khu vực">
      {TABS.map((tab) => (
        <button
          key={tab.key}
          type="button"
          role="tab"
          aria-selected={value === tab.key}
          className={`region-tab${value === tab.key ? " region-tab--active" : ""}`}
          onClick={() => onChange(tab.key)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
}

export type { Region };
