import { defineStore } from 'pinia';

export const THEME_MODES = {
  LIGHT: 'light',
  DARK: 'dark',
}

const checkDarkTheme = () => {
  if (process.client) {
    return window?.localStorage.getItem('theme') === THEME_MODES.DARK || window.matchMedia('(prefers-color-scheme: dark)').matches
  }

  return {
    themeType: false
  }
}

export const useThemeStore = defineStore('themeStore', {
  state: () => ({
    themeType: checkDarkTheme() ?  THEME_MODES.DARK : THEME_MODES.LIGHT
  }),
  actions: {
    themeChange() {
      const newType = this.themeType === THEME_MODES.DARK ? THEME_MODES.LIGHT : THEME_MODES.DARK;

      window?.localStorage.setItem('theme', newType);
      this.themeType = newType;
    },
  },
})
