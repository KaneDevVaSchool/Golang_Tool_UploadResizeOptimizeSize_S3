// Âm thanh toast/reaction tổng hợp bằng Web Audio API - không dùng file
// .mp3/.wav (zero asset, không tăng bundle size, không cần quản lý license).
// Mỗi lần gọi tạo 1 AudioContext ngắn hạn rồi tự đóng - tránh giữ context
// sống toàn app (một số trình duyệt giới hạn số AudioContext đồng thời).

type OscType = OscillatorType;

function playBeep(ctx: AudioContext, freq: number, start: number, duration: number, peakGain: number, type: OscType = "sine") {
  const osc = ctx.createOscillator();
  const gain = ctx.createGain();
  osc.type = type;
  osc.frequency.value = freq;

  // Envelope mềm: tăng nhanh rồi giảm dần (exponential) để tránh tiếng click
  // khi bật/tắt oscillator đột ngột.
  gain.gain.setValueAtTime(0.0001, start);
  gain.gain.exponentialRampToValueAtTime(peakGain, start + 0.02);
  gain.gain.exponentialRampToValueAtTime(0.0001, start + duration);

  osc.connect(gain);
  gain.connect(ctx.destination);
  osc.start(start);
  osc.stop(start + duration + 0.02);
}

function getAudioContext(): AudioContext | null {
  const Ctor = window.AudioContext || (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!Ctor) return null;
  try {
    return new Ctor();
  } catch {
    // Một số trình duyệt chặn tạo AudioContext trước tương tác người dùng đầu
    // tiên - im lặng bỏ qua, không làm gián đoạn luồng chính (toast vẫn hiện).
    return null;
  }
}

// closeSoon đóng context sau khi tiếng đã phát xong, tránh giữ tài nguyên.
function closeSoon(ctx: AudioContext, delayMs: number) {
  window.setTimeout(() => {
    void ctx.close().catch(() => {});
  }, delayMs);
}

export function playSuccessSound() {
  const ctx = getAudioContext();
  if (!ctx) return;
  const t = ctx.currentTime;
  playBeep(ctx, 880, t, 0.1, 0.12, "sine");
  playBeep(ctx, 1174.7, t + 0.1, 0.14, 0.1, "sine");
  closeSoon(ctx, 400);
}

export function playErrorSound() {
  const ctx = getAudioContext();
  if (!ctx) return;
  const t = ctx.currentTime;
  playBeep(ctx, 320, t, 0.16, 0.14, "triangle");
  playBeep(ctx, 220, t + 0.14, 0.2, 0.12, "triangle");
  closeSoon(ctx, 500);
}

export function playInfoSound() {
  const ctx = getAudioContext();
  if (!ctx) return;
  const t = ctx.currentTime;
  playBeep(ctx, 660, t, 0.09, 0.08, "sine");
  closeSoon(ctx, 250);
}

// playReactionSound: 1 tần số riêng mỗi loại cảm xúc (tương tự toast sound
// nhưng thêm noise "pop" ngắn để cảm giác vui/nhanh hơn, phù hợp thao tác
// click liên tục ở ReactionPicker). freqBase do caller truyền theo loại
// reaction (xem web/src/components/public/ReactionPicker.tsx).
export function playReactionSound(freqBase: number) {
  const ctx = getAudioContext();
  if (!ctx) return;
  const t = ctx.currentTime;
  playBeep(ctx, freqBase, t, 0.08, 0.1, "sine");
  playBeep(ctx, freqBase * 1.5, t + 0.05, 0.1, 0.07, "sine");
  closeSoon(ctx, 300);
}
