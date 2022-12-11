const defaultTheme = require('tailwindcss/defaultTheme');
const colors = require('tailwindcss/colors')

/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    "./Components/**/*.vue",
    "./layouts/**/*.vue",
    "./pages/**/*.vue",
    "./nuxt.config.js",
  ],
  variants: {
    extend: {
      opacity: ['disabled'],
      borderWidth: ['hover', 'first'],
      ringWidth: ['hover'],
    }
  },
  theme: {
    extend: {
      fontFamily: {
        sans: ['Nunito', ...defaultTheme.fontFamily.sans],
      },
      transitionProperty: {
        'height': 'height'
      },
      boxShadow: {
        bottom: 'inset 0 -38px 38px -38px #ececec',
      },
      fontSize: {
        '2xs': ['0.6rem', {lineHeight: '1rem'}]
      },
    },
    colors: {
      transparent: 'transparent',
      current: 'currentColor',
      black: colors.black,
      white: colors.white,
      gray: colors.gray,
      purple: colors.indigo,
      green: colors.green,
      blue: colors.blue,
      red: colors.rose,
      pink: colors.pink,
      orange: colors.amber,
    }
  }
};
