/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#171717",
        paper: "#f7f4ea",
        accent: "#d9652b",
        teal: "#176b63",
        moss: "#56724b",
      },
    },
  },
  plugins: [],
};
