import { motion, useReducedMotion } from "framer-motion";
import { useEffect, useState } from "react";

const PARAGRAPHS = [
  `Hệ thống trường Việt Mỹ – VASchools là nơi tập hợp đội ngũ giáo viên, cán bộ nhân viên có
  trình độ chuyên môn và tâm huyết trong công việc. Chúng tôi đang làm việc hết sức nhiệt tình
  để mang đến mô hình giáo dục chất lượng cao cho tất cả học sinh trong hệ thống.`,
  `Chúng tôi đã và đang xây dựng một môi trường giáo dục thúc đẩy sự phát triển năng lực ngoại
  ngữ và nhân cách của mỗi học sinh. Chương trình tiếng Anh ESL giúp các em tự tin sử dụng tiếng
  Anh như ngôn ngữ thứ hai và đảm bảo đủ năng lực theo học tại các trường Đại học quốc tế trong
  và ngoài nước. Bên cạnh đó, chúng tôi luôn kết hợp chương trình học tập với các hoạt động
  ngoại khóa để học sinh có thể rèn luyện thêm nhiều kỹ năng nhằm mục tiêu phát huy tính sáng
  tạo, khả năng tư duy, lý luận; tự khám phá bản thân cũng như tôn trọng người khác; đoàn kết,
  hòa đồng với bạn bè, cùng nhau tham gia các hoạt động thiện nguyện trong cộng đồng để phát huy
  lòng nhân ái. Hệ thống trường Việt Mỹ chúng tôi mong muốn tạo dựng nền tảng giá trị vững chắc
  cho học sinh, giúp các em ngày càng tự tin qua từng giai đoạn phát triển đến khi trưởng thành.`,
  `Chúng tôi tin rằng mục đích quan trọng của giáo dục là phát triển con người toàn diện cả về
  tri thức lẫn tâm hồn. Do đó, sứ mệnh của Hệ thống trường Việt Mỹ chúng tôi là đào tạo thế hệ
  trẻ Việt Nam trở thành những công dân có trách nhiệm với bản thân, gia đình và xã hội; đủ bản
  lĩnh giúp đất nước phát triển và hội nhập với thế giới.`,
];

/** 5 giá trị cốt lõi, thu về dạng nhãn chữ chạy dọc chân thư. */
const CORE_VALUES = [
  { name: "Nhân Ái", color: "var(--vas-nhan-ai)" },
  { name: "Bản Lĩnh", color: "var(--vas-ban-linh)" },
  { name: "Tri Thức", color: "var(--vas-tri-thuc)" },
  { name: "Trách Nhiệm", color: "var(--vas-trach-nhiem)" },
  { name: "Khai Phóng", color: "var(--vas-khai-phong)" },
];

/**
 * Trang /thu-ngo - Thư ngỏ của Chủ tịch Hội đồng Quản trị VASchools.
 *
 * Bố cục KHÔNG CUỘN TRANG: cả trang là một "sân khấu" cao đúng 100dvh trừ đi
 * navbar, ở giữa đặt một tờ thư giấy cũ. Trang trước đây xếp chồng ba section
 * cao gần cả màn hình (banner → dải giá trị → thân thư) nên phải cuộn ba lần
 * mới đọc hết một lá thư ngắn; giờ mọi thứ nằm trong một khung nhìn, chỉ phần
 * VĂN BẢN cuộn bên trong tờ giấy khi màn hình quá thấp.
 *
 * Vì sao "tờ giấy" chứ không phải card phẳng: đây là thư ngỏ, không phải bài
 * blog. Nền giấy ngả vàng + mép răng cưa (deckled edge) + nếp gấp ngang giữa
 * tờ khiến người đọc nhận ra ngay đây là một lá thư trước khi đọc chữ đầu
 * tiên. Toàn bộ hoạ tiết vẽ bằng gradient/SVG trong CSS, không tải ảnh nào.
 *
 * Hoạt cảnh mở thư (chỉ chạy 1 lần lúc vào trang): phong bì gập mở ra rồi tờ
 * thư trượt lên. Nếu người dùng tắt chuyển động thì bỏ qua hẳn, hiện thẳng
 * tờ thư - xem `reduceMotion` bên dưới.
 */
export default function OpenLetterPage() {
  const reduceMotion = useReducedMotion();
  // "sealed" = phong bì còn đóng. Bỏ qua hẳn giai đoạn này khi người dùng đã
  // tắt chuyển động: với họ, hoạt cảnh mở thư chỉ là một quãng chờ vô nghĩa.
  const [opened, setOpened] = useState(() => Boolean(reduceMotion));

  useEffect(() => {
    if (reduceMotion) {
      setOpened(true);
      return;
    }
    // Trễ một nhịp ngắn trước khi bật nắp: mắt cần kịp thấy phong bì đang
    // đóng, nếu mở ngay lúc trang vừa hiện thì không ai nhận ra đó là phong bì.
    const timer = window.setTimeout(() => setOpened(true), 620);
    return () => window.clearTimeout(timer);
  }, [reduceMotion]);

  return (
    <div className="letter-stage">
      {/* Nền sân khấu: hai quầng màu thương hiệu trôi rất chậm + lớp vân giấy
          phủ toàn khung, đứng sau tờ thư. */}
      <div className="letter-stage-bg" aria-hidden>
        <span className="letter-glow letter-glow--teal" />
        <span className="letter-glow letter-glow--gold" />
      </div>

      <div className={`letter-scene${opened ? " is-opened" : ""}`}>
        {/* Phong bì - chỉ là lớp trang trí cho hoạt cảnh mở đầu, ẩn khỏi
            trình đọc màn hình vì không mang thông tin nào. Không render khi
            người dùng đã tắt chuyển động: nó chỉ tồn tại để chạy hoạt cảnh,
            đứng yên thì là một hình thù lạ nằm sau tờ thư. */}
        {!reduceMotion && (
          <div className="letter-envelope" aria-hidden>
            <span className="letter-envelope-flap" />
            <span className="letter-envelope-body" />
          </div>
        )}

        <motion.article
          className="letter-sheet"
          initial={reduceMotion ? { opacity: 1 } : { opacity: 0, y: 64, rotateX: 14, scale: 0.94 }}
          animate={{ opacity: 1, y: 0, rotateX: 0, scale: 1 }}
          transition={{ delay: reduceMotion ? 0 : 0.72, duration: 0.9, ease: [0.16, 0.84, 0.28, 1] }}
          aria-labelledby="letter-title"
        >
          {/* Nếp gấp ngang - vết hằn của lá thư từng được gấp làm ba. */}
          <span className="letter-fold letter-fold--top" aria-hidden />
          <span className="letter-fold letter-fold--bottom" aria-hidden />

          <header className="letter-head">
            <img
              className="letter-wordmark"
              src="/images/vas-wordmark-stacked.png"
              alt="Vietnam America Schools"
            />
            <div className="letter-head-text">
              <p className="letter-kicker">Kỷ niệm 20 năm thành lập · 2006 – 2026</p>
              <h1 className="letter-title" id="letter-title">
                Thư Ngỏ
              </h1>
            </div>
          </header>

          {/* Vùng chữ - chỗ DUY NHẤT được phép cuộn trong trang. tabIndex=0 để
              bàn phím cuộn được khi màn hình thấp; không có nó thì người dùng
              bàn phím kẹt lại ở phần chữ bị cắt. */}
          <div className="letter-body" tabIndex={0} role="region" aria-label="Nội dung thư ngỏ">
            <p className="letter-salutation">Quý Phụ huynh và các em học sinh thân mến!</p>
            {PARAGRAPHS.map((text, i) => (
              <p className="letter-paragraph" key={i}>
                {text}
              </p>
            ))}
            <p className="letter-signoff">
              <span className="letter-signoff-role">Chủ tịch Hội đồng Quản trị</span>
              <span className="letter-signoff-org">VASchools</span>
            </p>
          </div>

          <footer className="letter-values" aria-label="5 giá trị cốt lõi">
            {CORE_VALUES.map((v) => (
              <span className="letter-value" key={v.name} style={{ "--value-color": v.color } as React.CSSProperties}>
                <span className="letter-value-dot" aria-hidden />
                {v.name}
              </span>
            ))}
          </footer>
        </motion.article>
      </div>

      {/* Mũi tên báo còn footer (địa chỉ, liên hệ) bên dưới. Lá thư chiếm trọn
          một khung nhìn nên nếu không có dấu hiệu này, đáy màn hình trông như
          hết trang và không ai cuộn xuống tìm phần liên hệ nữa. */}
      <span className="letter-scroll-cue" aria-hidden>
        <svg viewBox="0 0 24 24" width="18" height="18">
          <path
            d="M6 9l6 6 6-6"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        </svg>
      </span>
    </div>
  );
}
