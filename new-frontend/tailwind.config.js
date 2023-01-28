/* eslint-disable @typescript-eslint/dot-notation */
import defaultColors from "tailwindcss/colors";
import { fontFamily, fontWeight } from "tailwindcss/defaultTheme";

module.exports = {
  darkMode: "class",
  content: [
    "./assets/**/*.{scss,css}",
    "./components/**/*.{js,vue,ts}",
    "./components/**/*.stories.{js,vue,ts}",
    "./layouts/**/*.vue",
    "./pages/**/*.vue",
    "./plugins/**/*.{js,ts}",
    "./nuxt.config.{js,ts}",
    "./app.vue",
  ],
  variants: {
    extend: {
      opacity: ["disabled"],
      borderWidth: ["hover", "first"],
      ringWidth: ["hover"],
    },
  },
  theme: {
    extend: {
      fontFamily: {
        sans: ["Nunito", fontFamily.sans],
      },
      transitionProperty: {
        height: "height",
      },
      boxShadow: {
        bottom: "inset 0 -38px 38px -38px #ececec",
      },
      fontSize: {
        "2xs": ["0.6rem", { lineHeight: "1rem" }],
      },
    },
    fontWeight: {
      ...fontWeight,
    },
    colors: {
      ...defaultColors,
      transparent: "transparent",
      current: "currentColor",
      purple: defaultColors.indigo,
      red: defaultColors.rose,
      orange: defaultColors.amber,
    },
  },
};
