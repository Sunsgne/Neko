import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./lib/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        background: "hsl(222 47% 7%)",
        surface: "hsl(222 40% 11%)",
        elevated: "hsl(222 38% 14%)",
        border: "hsl(222 25% 20%)",
        foreground: "hsl(210 40% 96%)",
        muted: "hsl(215 20% 65%)",
        primary: "hsl(199 89% 52%)",
        success: "hsl(152 60% 45%)",
        warning: "hsl(38 92% 55%)",
        danger: "hsl(0 72% 55%)",
      },
      fontFamily: {
        sans: ["Inter", "system-ui", "sans-serif"],
        mono: ["JetBrains Mono", "ui-monospace", "monospace"],
      },
      borderRadius: {
        xl: "0.875rem",
      },
      boxShadow: {
        card: "0 1px 2px rgba(0,0,0,0.3), 0 8px 24px rgba(0,0,0,0.25)",
        glow: "0 0 0 1px hsl(199 89% 52% / 0.4), 0 0 24px hsl(199 89% 52% / 0.15)",
      },
    },
  },
  plugins: [],
};

export default config;
