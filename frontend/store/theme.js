export const state = () => ({
  darkMode: true,
  screenshot: false
})

export const getters = {
  isDarkMode(state) {
    return state.darkMode
  },
  isScreenshot(state) {
    return state.screenshot
  }
}

export const actions = {
  detect(context) {
    context.commit(
      'toggle',
      (localStorage.theme === 'dark' || (!('theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches))
    )
  }
}

export const mutations = {
  takeScreenshot(state, value) {
    state.screenshot = value
  },
  toggle(state, value) {
    if (value === true) {
      localStorage.theme = 'dark'
      state.darkMode = true
      document.documentElement.classList.add('dark')
    } else {
      localStorage.theme = 'light'
      state.darkMode = false
      document.documentElement.classList.remove('dark')
    }
  }
}
