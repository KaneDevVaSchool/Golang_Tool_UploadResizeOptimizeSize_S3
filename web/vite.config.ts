import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      // /auth/google/* là browser-redirect flow (OAuth), nằm ngoài /api -
      // cần proxy riêng để nút đăng nhập hoạt động đúng ở dev mode (:5173).
      "/auth": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
    rollupOptions: {
      output: {
        // Tách thư viện lớn khỏi code ứng dụng. Không tách thì mỗi lần sửa
        // một dòng trong src/ là toàn bộ react + framer-motion + recharts
        // đổi hash và khách phải tải lại từ đầu; tách ra thì các chunk này
        // nằm yên trong cache trình duyệt qua nhiều lần deploy.
        //
        // Chỉ tách theo thư viện, không tách theo trang: việc chia theo route
        // đã do React.lazy trong App.tsx lo, Rollup tự sinh chunk tương ứng.
        manualChunks(id) {
          if (!id.includes("node_modules")) return;

          if (id.includes("framer-motion") || id.includes("motion-dom") || id.includes("motion-utils")) {
            return "vendor-motion";
          }
          if (id.includes("react-router")) {
            return "vendor-router";
          }
          // Đặt sau cùng trong nhóm react: "react-dom" và "react-router" đều
          // chứa chuỗi "react", nên các nhánh riêng phải được kiểm tra trước.
          if (id.includes("/react/") || id.includes("/react-dom/") || id.includes("/scheduler/")) {
            return "vendor-react";
          }
        },
      },
    },
  },
});
