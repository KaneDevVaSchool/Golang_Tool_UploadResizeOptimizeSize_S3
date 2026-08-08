import { AnimatePresence, motion } from "framer-motion";
import { useEffect, useState } from "react";

const STORAGE_KEY = "vas-kho-anh-onboarding-v1";

const POINTS = [
  {
    title: "Giảm dung lượng",
    body: "Thu nhỏ và tối ưu ảnh — nhẹ hơn, tải nhanh hơn, vẫn rõ nét.",
  },
  {
    title: "Lưu trữ tập trung",
    body: "Một kho dùng chung cho toàn hệ thống Trường Việt Mỹ.",
  },
  {
    title: "Một ảnh — nhiều nơi",
    body: "Đường dẫn riêng cho SIS, LMS, website và hệ thống nội bộ.",
  },
  {
    title: "Xử lý hàng loạt",
    body: "Tải và xử lý nhiều ảnh cùng lúc — tiết kiệm thời gian vận hành.",
  },
  {
    title: "Tự động xử lý",
    body: "Giải nén, thu nhỏ, tối ưu và lưu trong một quy trình.",
  },
  {
    title: "Tiết kiệm chi phí",
    body: "Ít bản sao trùng lặp — giảm tải lưu trữ và chi phí hạ tầng.",
  },
  {
    title: "Dễ sử dụng",
    body: "Chọn ảnh → xử lý → lưu → lấy link dùng ngay.",
  },
] as const;

const ease = [0.22, 1, 0.36, 1] as const;

type OnboardingProps = {
  open: boolean;
  onDone: () => void;
};

export function readOnboardingDone(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === "1";
  } catch {
    return false;
  }
}

export function writeOnboardingDone() {
  try {
    localStorage.setItem(STORAGE_KEY, "1");
  } catch {
    /* ignore */
  }
}

export function OnboardingTrigger({
  onClick,
  visible = true,
}: {
  onClick: () => void;
  visible?: boolean;
}) {
  return (
    <AnimatePresence>
      {visible && (
        <motion.button
          type="button"
          className="onboard-reopen"
          onClick={(e) => {
            e.stopPropagation();
            onClick();
          }}
          aria-label="Xem lại giới thiệu Kho ảnh"
          title="Giới thiệu"
          layoutId="onboard-shell"
          initial={{ opacity: 0, scale: 0.55, y: 8 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.6, y: 6 }}
          whileHover={{ scale: 1.08 }}
          whileTap={{ scale: 0.94 }}
          transition={{ type: "spring", stiffness: 420, damping: 28 }}
        >
          <span className="onboard-reopen-glow" aria-hidden />
          <svg className="onboard-reopen-icon" viewBox="0 0 24 24" aria-hidden>
            <circle cx="12" cy="12" r="9.25" />
            <path d="M12 10.75v5" />
            <circle cx="12" cy="7.6" r="0.95" fill="currentColor" stroke="none" />
          </svg>
        </motion.button>
      )}
    </AnimatePresence>
  );
}

export function Onboarding({ open, onDone }: OnboardingProps) {
  const [step, setStep] = useState(0);
  const [review, setReview] = useState(false);
  const total = 3;

  function finish() {
    writeOnboardingDone();
    onDone();
  }

  useEffect(() => {
    if (!open) return;
    setStep(0);
    setReview(readOnboardingDone());
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") {
        writeOnboardingDone();
        onDone();
      }
      if (e.key === "ArrowRight") setStep((s) => Math.min(total - 1, s + 1));
      if (e.key === "ArrowLeft") setStep((s) => Math.max(0, s - 1));
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onDone]);

  function next() {
    if (step >= total - 1) finish();
    else setStep((s) => s + 1);
  }

  return (
    <AnimatePresence>
      {open && (
        <motion.div
          className="onboard"
          role="dialog"
          aria-modal="true"
          aria-labelledby="onboard-title"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.22 }}
        >
          <motion.button
            type="button"
            className="onboard-backdrop"
            aria-label="Đóng"
            onClick={finish}
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
          />

          <motion.div
            className="onboard-panel"
            layoutId="onboard-shell"
            initial={{ opacity: 0.88, scale: 0.92 }}
            animate={{ opacity: 1, scale: 1 }}
            exit={{ opacity: 0.9, scale: 0.94 }}
            transition={{ type: "spring", stiffness: 380, damping: 32 }}
          >
            <header className="onboard-top">
              <p className="onboard-kicker">Giá trị hệ thống</p>
              <button type="button" className="onboard-skip" onClick={finish}>
                {review ? "Đóng" : "Bỏ qua"}
              </button>
            </header>

            <div className="onboard-body">
              <AnimatePresence mode="wait">
                {step === 0 && (
                  <motion.div
                    key="s0"
                    className="onboard-slide"
                    initial={{ opacity: 0, y: 12, filter: "blur(4px)" }}
                    animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
                    exit={{ opacity: 0, y: -10, filter: "blur(3px)" }}
                    transition={{ duration: 0.32, ease }}
                  >
                    <h2 id="onboard-title" className="onboard-title">
                      Kho ảnh VA Schools
                    </h2>
                    <p className="onboard-sub">Hạ tầng ảnh · Hệ thống Trường Việt Mỹ</p>
                    <p className="onboard-lead">
                      Ảnh chuẩn — dùng chung toàn trường, phục vụ mọi hệ thống.
                    </p>
                    <p className="onboard-copy">
                      Thay vì gửi file rời qua Zalo hay email, VA Schools xây dựng{" "}
                      <strong>kho ảnh tập trung</strong>: chỉnh sửa, giải nén, thu nhỏ và tối ưu trước
                      khi lưu. Mỗi ảnh có <strong>đường dẫn riêng</strong> — dán vào SIS, LMS, website
                      hoặc tài liệu nội bộ, không cần tải lại hay gửi lại file.
                    </p>
                    <p className="onboard-quote">
                      Một lần lưu. Một nơi quản lý. Phục vụ cả hệ thống.
                    </p>
                  </motion.div>
                )}

                {step === 1 && (
                  <motion.div
                    key="s1"
                    className="onboard-slide"
                    initial={{ opacity: 0, y: 12, filter: "blur(4px)" }}
                    animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
                    exit={{ opacity: 0, y: -10, filter: "blur(3px)" }}
                    transition={{ duration: 0.32, ease }}
                  >
                    <h2 className="onboard-title">Quy trình đơn giản</h2>
                    <p className="onboard-lead">Chọn ảnh — hệ thống xử lý — lấy link để dùng.</p>
                    <ol className="onboard-flow">
                      <li>
                        <span>1</span>
                        <div>
                          <strong>Chọn ảnh</strong>
                          <em>Một hoặc nhiều file cùng lúc</em>
                        </div>
                      </li>
                      <li>
                        <span>2</span>
                        <div>
                          <strong>Hệ thống xử lý</strong>
                          <em>Thu nhỏ, tối ưu, ảnh nhẹ hơn</em>
                        </div>
                      </li>
                      <li>
                        <span>3</span>
                        <div>
                          <strong>Lưu tập trung</strong>
                          <em>Kho ảnh chung Trường Việt Mỹ</em>
                        </div>
                      </li>
                      <li>
                        <span>4</span>
                        <div>
                          <strong>Lấy link · Dùng ngay</strong>
                          <em>SIS, LMS, website và hệ thống nội bộ</em>
                        </div>
                      </li>
                    </ol>
                  </motion.div>
                )}

                {step === 2 && (
                  <motion.div
                    key="s2"
                    className="onboard-slide"
                    initial={{ opacity: 0, y: 12, filter: "blur(4px)" }}
                    animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
                    exit={{ opacity: 0, y: -10, filter: "blur(3px)" }}
                    transition={{ duration: 0.32, ease }}
                  >
                    <h2 className="onboard-title">Giá trị mang lại</h2>
                    <p className="onboard-lead">
                      Từ file rời rạc thành kho ảnh tổ chức — nhẹ hơn, thống nhất, sẵn sàng dùng
                      trên toàn hệ thống Trường Việt Mỹ.
                    </p>
                    <ul className="onboard-points">
                      {POINTS.map((p, i) => (
                        <motion.li
                          key={p.title}
                          initial={{ opacity: 0, y: 8 }}
                          animate={{ opacity: 1, y: 0 }}
                          transition={{ delay: 0.04 * i, duration: 0.28, ease }}
                        >
                          <strong>{p.title}</strong>
                          <span>{p.body}</span>
                        </motion.li>
                      ))}
                    </ul>
                  </motion.div>
                )}
              </AnimatePresence>
            </div>

            <footer className="onboard-foot">
              <div className="onboard-dots" aria-hidden>
                {Array.from({ length: total }, (_, i) => (
                  <button
                    key={i}
                    type="button"
                    className="onboard-dot"
                    data-active={i === step}
                    onClick={() => setStep(i)}
                    aria-label={`Bước ${i + 1}`}
                  />
                ))}
              </div>
              <div className="onboard-actions">
                {step > 0 && (
                  <button type="button" className="btn btn-ghost" onClick={() => setStep((s) => s - 1)}>
                    Quay lại
                  </button>
                )}
                <button type="button" className="btn btn-primary" onClick={next}>
                  {step >= total - 1 ? (review ? "Xong" : "Bắt đầu lưu ảnh") : "Tiếp tục"}
                </button>
              </div>
            </footer>
          </motion.div>
        </motion.div>
      )}
    </AnimatePresence>
  );
}
