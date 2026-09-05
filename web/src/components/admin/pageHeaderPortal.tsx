import { createContext, useContext, useState, type ReactNode } from "react";

/**
 * Hạ tầng "teleport" cho AdminPageHeader - port pattern
 * usePageHeaderTarget.js của va-workspace sang React.
 *
 * AdminLayout render <PageHeaderSlotProvider>, AdminHeader đăng ký phần tử
 * DOM đích qua `useRegisterPageHeaderSlot`, mỗi trang con render
 * <AdminPageHeader> và nội dung được createPortal thẳng lên thanh header —
 * nhờ vậy title + nút hành động của trang nằm trên MỘT hàng header duy nhất
 * (kiểu 1Office) thay vì lặp lại một dải header riêng trong từng trang.
 *
 * Khi slot chưa sẵn sàng (null) thì AdminPageHeader render tại chỗ như
 * fallback - tránh mất tiêu đề nếu layout đổi.
 */
type PageHeaderSlotValue = {
  slot: HTMLElement | null;
  setSlot: (el: HTMLElement | null) => void;
};

const PageHeaderSlotContext = createContext<PageHeaderSlotValue | null>(null);

export function PageHeaderSlotProvider({ children }: { children: ReactNode }) {
  const [slot, setSlot] = useState<HTMLElement | null>(null);
  return (
    <PageHeaderSlotContext.Provider value={{ slot, setSlot }}>{children}</PageHeaderSlotContext.Provider>
  );
}

/** Dùng trong AdminHeader: ref callback gắn vào div đích. */
export function useRegisterPageHeaderSlot() {
  return useContext(PageHeaderSlotContext)?.setSlot ?? (() => {});
}

/** Dùng trong AdminPageHeader: phần tử đích hiện tại (null nếu chưa mount). */
export function usePageHeaderSlot() {
  return useContext(PageHeaderSlotContext)?.slot ?? null;
}
