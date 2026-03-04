module.exports = {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ui: {
          bg: "rgb(var(--ui-bg) / <alpha-value>)",
          surface: "rgb(var(--ui-surface) / <alpha-value>)",
          panel: "rgb(var(--ui-panel) / <alpha-value>)",
          line: "rgb(var(--ui-line) / <alpha-value>)",
          text: "rgb(var(--ui-text) / <alpha-value>)",
          muted: "rgb(var(--ui-muted) / <alpha-value>)",
          accent: "rgb(var(--ui-accent) / <alpha-value>)"
        }
      },
      spacing: {
        1: "4px",
        2: "8px",
        3: "12px",
        4: "16px",
        6: "24px",
        8: "32px"
      },
      maxWidth: {
        feed: "760px"
      }
    }
  },
  plugins: []
};
