import { Globe, Mail, MapPin, Phone } from "lucide-react";

const HEADQUARTERS = [
  { label: "Hội sở 1", address: "252 Lạc Long Quân, Phường Bình Thới, TP.HCM" },
  { label: "Hội sở 2", address: "806 Âu Cơ, Phường Tân Bình, TP.HCM" },
] as const;
const HOTLINE_DISPLAY = "0828 252 806";
const HOTLINE_TEL = "+84828252806";
const LANDLINE_DISPLAY = "(028) 7307 3806";
const LANDLINE_TEL = "+842873073806";
const EMAIL = "tuyensinh@vaschools.edu.vn";
const WEBSITE_LABEL = "www.vaschools.edu.vn";
const WEBSITE_HREF = "https://www.vaschools.edu.vn";
const ZALO_HREF = "https://zalo.me/0828252806";
const YOUTUBE_HREF = "https://www.youtube.com/channel/UC4isMJ7ZKVNxJ8LIb_aRn2w";
const FACEBOOK_HREF = "https://www.facebook.com/vaschools.edu.vn";

function ZaloMark() {
  return (
    <svg className="public-footer-zalo" viewBox="0 0 48 48" aria-hidden>
      <rect width="48" height="48" rx="12" fill="#0068FF" />
      <path
        fill="#fff"
        d="M9.5 15.2c0-2.7 2.2-4.9 4.9-4.9h19.2c2.7 0 4.9 2.2 4.9 4.9v12.4c0 2.7-2.2 4.9-4.9 4.9H23.1L16.2 39v-6.5h-1.8c-2.7 0-4.9-2.2-4.9-4.9V15.2z"
      />
      <text
        x="24"
        y="25.6"
        textAnchor="middle"
        fill="#0068FF"
        fontFamily="Arial, Helvetica, sans-serif"
        fontSize="9.4"
        fontWeight="700"
      >
        Zalo
      </text>
    </svg>
  );
}

function YouTubeMark() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden>
      <path
        fill="currentColor"
        d="M23.5 6.2a3.05 3.05 0 0 0-2.15-2.16C19.5 3.6 12 3.6 12 3.6s-7.5 0-9.35.44A3.05 3.05 0 0 0 .5 6.2 31.8 31.8 0 0 0 0 12a31.8 31.8 0 0 0 .5 5.8 3.05 3.05 0 0 0 2.15 2.16C4.5 20.4 12 20.4 12 20.4s7.5 0 9.35-.44A3.05 3.05 0 0 0 23.5 17.8 31.8 31.8 0 0 0 24 12a31.8 31.8 0 0 0-.5-5.8ZM9.75 15.57V8.43L15.84 12l-6.09 3.57Z"
      />
    </svg>
  );
}

function FacebookMark() {
  return (
    <svg viewBox="0 0 24 24" width="18" height="18" aria-hidden>
      <path
        fill="currentColor"
        d="M14.5 8.5V6.8c0-.7.5-1.3 1.2-1.3H17V3h-2.2C12.3 3 11 4.4 11 6.6v1.9H9v2.6h2V21h3.5v-9.9h2.4l.4-2.6h-2.8Z"
      />
    </svg>
  );
}

/**
 * Footer dùng chung mọi trang public: khép khu vườn bằng dải sóng + đất,
 * tấm đồng kỷ niệm 20 năm (cùng chất liệu nameplate khung tranh), rồi
 * cụm địa chỉ / liên hệ của Hệ thống Trường Việt Mỹ.
 */
export function PublicFooter() {
  return (
    <footer className="public-footer">
      <div className="public-footer-wave" aria-hidden>
        <svg viewBox="0 0 1440 64" preserveAspectRatio="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M0,28 C160,56 320,8 480,24 C640,40 800,4 960,22 C1120,40 1280,12 1440,30 L1440,64 L0,64 Z" />
        </svg>
      </div>

      <div className="public-footer-body">
        <div className="public-footer-inner">
          <div className="public-footer-brand">
            <img className="public-footer-wordmark" src="/images/vas-white.png" alt="Vietnam America Schools" />
            <p className="public-footer-system">Hệ thống Trường Việt Mỹ</p>
            <p className="public-footer-title">Phòng Triển Lãm Tranh</p>
            <div className="public-footer-plaque">
              <span className="public-footer-screw" aria-hidden />
              <div className="public-footer-copy">
                <strong>Kỷ niệm 20 năm thành lập Trường Việt Mỹ</strong>
                <span>2006 – 2026</span>
              </div>
              <span className="public-footer-screw" aria-hidden />
            </div>
          </div>

          <div className="public-footer-col">
            <h2 className="public-footer-heading">Hội sở</h2>
            <ul className="public-footer-contacts">
              {HEADQUARTERS.map((hq) => (
                <li key={hq.label}>
                  <p className="public-footer-address">
                    <span className="public-footer-icon" aria-hidden>
                      <MapPin size={16} strokeWidth={1.75} />
                    </span>
                    <span>
                      <em>{hq.label}</em>
                      {hq.address}
                    </span>
                  </p>
                </li>
              ))}
            </ul>
          </div>

          <div className="public-footer-col">
            <h2 className="public-footer-heading">Liên hệ</h2>
            <ul className="public-footer-contacts">
              <li>
                <p className="public-footer-address">
                  <span className="public-footer-icon" aria-hidden>
                    <Phone size={16} strokeWidth={1.75} />
                  </span>
                  <span>
                    <em>Hotline</em>
                    <span className="public-footer-phones">
                      <a className="public-footer-phone" href={`tel:${HOTLINE_TEL}`}>
                        {HOTLINE_DISPLAY}
                      </a>
                      <span className="public-footer-phones-sep" aria-hidden>
                        –
                      </span>
                      <a className="public-footer-phone" href={`tel:${LANDLINE_TEL}`}>
                        {LANDLINE_DISPLAY}
                      </a>
                    </span>
                  </span>
                </p>
              </li>
              <li>
                <a className="public-footer-link" href={`mailto:${EMAIL}`}>
                  <span className="public-footer-icon" aria-hidden>
                    <Mail size={16} strokeWidth={1.75} />
                  </span>
                  <span>
                    <em>Email</em>
                    {EMAIL}
                  </span>
                </a>
              </li>
              <li>
                <a className="public-footer-link" href={WEBSITE_HREF} target="_blank" rel="noopener noreferrer">
                  <span className="public-footer-icon" aria-hidden>
                    <Globe size={16} strokeWidth={1.75} />
                  </span>
                  <span>
                    <em>Website</em>
                    {WEBSITE_LABEL}
                  </span>
                </a>
              </li>
            </ul>
          </div>

          <div className="public-footer-col">
            <h2 className="public-footer-heading">Kết nối</h2>
            <ul className="public-footer-contacts">
              <li>
                <a
                  className="public-footer-link"
                  href={ZALO_HREF}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <span className="public-footer-social-link public-footer-social-link--zalo" aria-hidden>
                    <ZaloMark />
                  </span>
                  <span>
                    <em>Zalo</em>
                    0828 252 806
                  </span>
                </a>
              </li>
              <li>
                <a
                  className="public-footer-link"
                  href={YOUTUBE_HREF}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <span className="public-footer-social-link public-footer-social-link--youtube" aria-hidden>
                    <YouTubeMark />
                  </span>
                  <span>
                    <em>YouTube</em>
                    VA Schools
                  </span>
                </a>
              </li>
              <li>
                <a
                  className="public-footer-link"
                  href={FACEBOOK_HREF}
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  <span className="public-footer-social-link public-footer-social-link--facebook" aria-hidden>
                    <FacebookMark />
                  </span>
                  <span>
                    <em>Facebook</em>
                    vaschools.edu.vn
                  </span>
                </a>
              </li>
            </ul>
          </div>
        </div>

        <div className="public-footer-bar">
          <span className="public-footer-leaf" aria-hidden>
            🌿
          </span>
          <p>© {new Date().getFullYear()} Hệ thống Trường Việt Mỹ</p>
          <span className="public-footer-leaf public-footer-leaf--right" aria-hidden>
            🌿
          </span>
        </div>
      </div>
    </footer>
  );
}
