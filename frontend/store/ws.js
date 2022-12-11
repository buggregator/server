export const state = () => ({
  connected: false,
})

export const getters = {
  connected(state) {
    return state.connected
  },
}

export const mutations = {
  disconnect(state) {
    state.connected = false
  },
  connect(state) {
    state.connected = true
  }
}
