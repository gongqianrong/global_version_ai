/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./app/**/*.{js,jsx,ts,tsx}",
    "./components/**/*.{js,jsx,ts,tsx}",
  ],
  presets: [require("nativewind/preset")],
  important: "html",
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: "#0EA5E9",
          50: "#F0F9FF",
          100: "#E0F2FE",
          500: "#0EA5E9",
          600: "#0284C7",
          700: "#0369A1",
        },
        auction: {
          DEFAULT: "#F97316",
          500: "#F97316",
          600: "#EA580C",
        },
      },
    },
  },
  plugins: [],
};
