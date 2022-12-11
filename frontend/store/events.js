import Factory from "../app/Event/Factory";

export const state = () => ({
  selectedLabels: [],
  events: [],
})

export const actions = {
  clear({commit}, type) {
    this.$api.events.clear(type)
  },

  delete({commit}, event) {
    this.$api.events.delete(event.id)
  },

  async fetch({commit}) {
    const events = await this.$api.events.list()
    commit('set', events.map(e => Factory.create({json: e, store: this})))
  }
}

export const mutations = {
  selectLabel(state, label) {
    console.log(label)
    if (state.selectedLabels.includes(label)) {
      state.selectedLabels = state.selectedLabels.filter(l => label !== l)
    } else {
      state.selectedLabels.push(label)
    }
  },
  toggleCollapsedState(state, e) {
    const event = this.getters['events/eventByUuid'](e.uuid)

    if (event) {
      event.setCollapsed(!event.collapsed)
    }
  },
  clear(state, type) {
    if (type) {
      state.events = state.events.filter(e => e.app !== type)
      return
    }
    state.events = []
  },
  delete(state, uuid) {
    state.events = state.events.filter(e => e.uuid !== uuid)
  },
  push(state, event) {
    if (!event) {
      return;
    }

    const existsEvent = this.getters['events/eventByUuid'](event.uuid)

    if (existsEvent) {
      existsEvent.merge(event)
    } else {
      state.events.unshift(event)
    }
  },
  set(state, events) {
    state.events = events
  }
}

export const getters = {
  eventByUuid: (state) => (uuid) => {
    return state.events.find(event => event.uuid == uuid)
  },
  selectedLabels: state => {
    return state.selectedLabels
  },
  availableLabels: state => {
    let labels = [];

    state.events.forEach(event => {
      labels = [...labels, ...event.labels]
    })

    return labels.filter((item, index) => labels.indexOf(item) == index)
  },
  filteredByType: state => type => {
    return state.events.filter(e => e.app === type)
  },
  filtered: state => {
    return state.events
      // Filter by labels
      .filter(event => {
        if (state.selectedLabels.length > 0) {
          if (event.labels.length === 0) {
            return false
          }

          return state.selectedLabels.filter(
            value => event.labels.includes(value)
          ).length > 0
        }

        return true
      })
  }
}
