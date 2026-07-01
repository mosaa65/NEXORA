/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,jsx}"],
  theme: {
    extend: {
      colors: {
        charcoal: "#080A12",
        royal: "#5A32F4",
        electric: "#19B7FF",
        glass: "rgba(255, 255, 255, 0.08)"
      }
    }
  },
  plugins: []
};
