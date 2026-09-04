import { useEffect, useRef, useState } from "react";
import { AwardsBillboard } from "../../components/public/AwardsBillboard";
import { EducationLevelGate } from "../../components/public/EducationLevelGate";
import { FeaturedGallery } from "../../components/public/FeaturedGallery";
import { GradeLevelExplorer } from "../../components/public/GradeLevelExplorer";
import { HeroSection } from "../../components/public/HeroSection";
import type { Region } from "../../components/public/RegionTabs";
import { useScrollableBody } from "../../hooks/useScrollableBody";
import { fetchPublicArtworks } from "../../lib/publicApi";
import "../../styles/public.css";

/**
 * Trang public "20 năm VAS" - gộp 5 section theo đúng thứ tự: Hero -> Cổng
 * chọn khối -> Tác phẩm tiêu biểu -> Phòng triển lãm theo khối -> Billboard
 * vinh danh. Không cần đăng nhập. Quản lý scroll giữa section bằng
 * scrollIntoView đơn giản (không cần thư viện scroll-spy phức tạp).
 */
export default function PublicGallery() {
  useScrollableBody();
  const [initialRegion, setInitialRegion] = useState<Region>("all");
  const [selectedLevel, setSelectedLevel] = useState<"primary" | "secondary" | null>(null);

  const featuredRef = useRef<HTMLElement>(null);
  const explorerRef = useRef<HTMLElement>(null);

  function scrollToFeatured(region: Region) {
    setInitialRegion(region);
    featuredRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
  }

  function handleSelectLevel(level: "primary" | "secondary") {
    setSelectedLevel(level);
  }

  useEffect(() => {
    if (selectedLevel) {
      // Đợi 1 tick để GradeLevelExplorer render (level đổi từ null -> giá
      // trị thật) rồi mới scroll, tránh scroll tới vị trí cũ.
      const id = window.setTimeout(() => {
        explorerRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
      }, 80);
      return () => window.clearTimeout(id);
    }
  }, [selectedLevel]);

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

  return (
    <div className="public-gallery-page">
      <HeroSection onSelectRegion={scrollToFeatured} />
      <EducationLevelGate
        primaryCount={gradeCounts.primary}
        secondaryCount={gradeCounts.secondary}
        onSelectLevel={handleSelectLevel}
      />
      <FeaturedGallery ref={featuredRef} initialRegion={initialRegion} />
      <GradeLevelExplorer ref={explorerRef} level={selectedLevel} />
      <AwardsBillboard />
    </div>
  );
}
