/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts}',
  ],
  theme: {
    extend: {
      colors: {
        scout: {
          900: '#0f1117',
          800: '#1a1d2e',
          700: '#252940',
          600: '#363b58',
          accent: '#3b82f6',
        },
      },
    },
  },
  plugins: [],
}
