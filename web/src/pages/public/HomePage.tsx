import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { EducationLevelGate } from "../../components/public/EducationLevelGate";
import { HeroSection } from "../../components/public/HeroSection";
import type { Region } from "../../components/public/RegionTabs";
import { fetchPublicArtworks } from "../../lib/publicApi";

/**
 * Trang chủ /trien-lam - Hero + cổng chọn khối. Không còn cuộn tới các
 * section khác trong cùng trang (kiểu landing page); chọn khu vực ở Hero
 * hoặc chọn khối ở Gate đều điều hướng (navigate) sang trang riêng tương
 * ứng, dùng chung với navbar ở PublicLayout.
 */
export default function HomePage() {
  const navigate = useNavigate();
  const [gradeCounts, setGradeCounts] = useState({ primary: 0, secondary: 0 });

  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
      fetchPublicArtworks({ education_level: "primary", page_size: 1 }, controller.signal),
      fetchPublicArtworks({ education_level: "secondary", page_size: 1 }, controller.signal),
    ])
      .then(([primary, secondary]) => {
        if (controller.signal.aborted) return;
        setGradeCounts({ primary: primary.total_count, secondary: secondary.total_count });
      })
      .catch(() => {
        // im lặng bỏ qua - card vẫn hiện, chỉ thiếu số đếm chính xác
      });
    return () => controller.abort();
  }, []);

  function handleSelectRegion(region: Region) {
    navigate(`/tac-pham-tieu-bieu?khu-vuc=${region}`);
  }

  function handleSelectLevel(level: "primary" | "secondary") {
    navigate(`/phong-trien-lam?khoi=${level}`);
  }

  return (
    <div className="public-home-page">
      <HeroSection onSelectRegion={handleSelectRegion} />
      <EducationLevelGate
        primaryCount={gradeCounts.primary}
        secondaryCount={gradeCounts.secondary}
        onSelectLevel={handleSelectLevel}
      />
    </div>
  );
}
