import { AnimatePresence, motion } from "framer-motion";
import { Menu, X } from "lucide-react";
import { useEffect, useState } from "react";
import { NavLink, useLocation } from "react-router-dom";

const NAV_LINKS = [
  { to: "/", label: "Trang chủ", end: true },
  { to: "/tac-pham-tieu-bieu", label: "Tác phẩm tiêu biểu" },
  { to: "/phong-trien-lam", label: "Phòng triển lãm" },
  { to: "/bang-vang", label: "Bảng vàng" },
  { to: "/thu-ngo", label: "Thư ngỏ" },
];

/**
 * Navbar dính mép trên (sticky/fixed, full-width) dùng chung cho mọi trang
 * public /trien-lam/*. Nền kính mờ trắng xuyên suốt ngay từ đầu trang (chữ
 * luôn tối, đọc được trên mọi nền Hero phía sau) - khi cuộn quá một chút
 * chỉ tăng nhẹ độ đặc + shadow để tách khỏi nội dung, không đổi tông màu.
 */
export function PublicNavbar() {
  const [open, setOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);
  const location = useLocation();

  useEffect(() => {
    function onScroll() {
      setScrolled(window.scrollY > 16);
    }
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  useEffect(() => {
    setOpen(false);
  }, [location.pathname]);

  return (
    <header className={`public-navbar${scrolled ? " public-navbar--scrolled" : ""}`}>
      <div className="public-navbar-inner">
        <div className="public-navbar-brand">
          {/* Logo trỏ ra website trường, còn tiêu đề mới về trang chủ triển lãm:
              hai đích khác nhau nên phải là hai thẻ <a> riêng, không lồng nhau. */}
          <a
            href="https://vaschools.edu.vn/vi/"
            target="_blank"
            rel="noreferrer"
            className="public-navbar-brand-link"
            aria-label="Website Trường Quốc tế Việt Mỹ VAS"
            onClick={() => setOpen(false)}
          >
            <img className="public-navbar-brand-mark" src="/images/vas-wordmark-stacked.png" alt="Vietnam America Schools" />
          </a>
          <NavLink to="/" className="public-navbar-title" onClick={() => setOpen(false)}>
            <span className="public-navbar-title-text">Sắc Màu VASchools</span>
          </NavLink>
        </div>

        <nav className="public-navbar-links" aria-label="Điều hướng triển lãm">
          {NAV_LINKS.map((link) => (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.end}
              className={({ isActive }) => `public-navbar-link${isActive ? " public-navbar-link--active" : ""}`}
            >
              {link.label}
            </NavLink>
          ))}
        </nav>

        <button
          type="button"
          className="public-navbar-toggle"
          aria-label={open ? "Đóng menu" : "Mở menu"}
          aria-expanded={open}
          onClick={() => setOpen((v) => !v)}
        >
          {open ? <X size={20} /> : <Menu size={20} />}
        </button>
      </div>

      <AnimatePresence>
        {open && (
          <motion.nav
            className="public-navbar-mobile"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.22 }}
            aria-label="Điều hướng triển lãm (di động)"
          >
            {NAV_LINKS.map((link) => (
              <NavLink
                key={link.to}
                to={link.to}
                end={link.end}
                className={({ isActive }) => `public-navbar-mobile-link${isActive ? " public-navbar-mobile-link--active" : ""}`}
                onClick={() => setOpen(false)}
              >
                {link.label}
              </NavLink>
            ))}
          </motion.nav>
        )}
      </AnimatePresence>
    </header>
  );
}
