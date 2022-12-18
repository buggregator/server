<template>
  <div class="profiler-page page">
    <main>
      <header>
        <h1 class="page-title">{{ title }}</h1>
        <div v-if="hasEvents" class="page-buttons">
          <button @click="clearEvents" class="events__btn-clear">Clear events</button>
        </div>
      </header>
      <section class="page-events" v-if="hasEvents">
        <Event v-for="event in events" :key="event.uuid" :event="event"/>
      </section>
      <div v-else class="page-events__empty">
        <svg class="page-events__empty-icon" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 41 40">
          <path
            d="m35.6 14.6 5.2-5.2.2-.6c0-.3-.2-.5-.4-.6L26.6.1a.7.7 0 0 0-.9.1l-5.2 5.2L15.3.2a.7.7 0 0 0-.9-.1L.4 8.2c-.2.1-.4.3-.4.6 0 .2 0 .4.2.6l5.2 5.2-5.2 5.2-.2.6c0 .2.2.4.4.6l5.3 3v6.8c0 .3.2.5.4.6l14 8.2h.8l14-8.2c.2 0 .4-.3.4-.6v-6.7l5.3-3.1c.2-.2.4-.4.4-.6 0-.2 0-.4-.2-.6l-5.2-5.2Zm-2.6-.1-12.5 7.2L8 14.5 17.4 9l3.1-1.9 11.4 6.6 1.1.7Zm-6.7-13L39 9l-4.6 4.7-.8-.5-11.9-6.9 4.6-4.6Zm-11.6 0 4.6 4.7-12.7 7.4-4.7-4.7 12.8-7.3Zm-8.1 14 12.7 7.3-4.7 4.6-7.8-4.5L2 20l4.7-4.7Zm.6 9.3 7.2 4.2h.4l.5-.1 4.5-4.5v13.1L7.2 30.3v-5.5Zm26.6 5.5-12.6 7.2V24.4l4.5 4.5.5.2.4-.1 7.2-4.2v5.5Zm-7.5-2.9-4.6-4.6 12.7-7.4 4.6 4.7-12.7 7.3Z"/>
        </svg>
        <h3>Events not found</h3>
      </div>
    </main>
  </div>
</template>

<script>
export default {
  components: {Event},
  mounted() {
    this.$store.dispatch('events/fetch')
  },
  computed: {
    events() {
      return this.$store.getters['events/filteredByType'](this.type)
    },
    hasEvents() {
      return this.events.length > 0
    }
  },
  methods: {
    clearEvents() {
      this.$store.dispatch('events/clear', this.type)
    }
  }
}
</script>
