<template>
  <section>
    <h2>SMTP Settings</h2>
    <p class="smtp-page_description">Use these settings to send messages directly from your email client or mail transfer agent.</p>

    <div v-if="hasEvents" class="my-3">
      <button @click="clearEvents" class="events__btn-clear">Clear events</button>
    </div>
  </section>
</template>

<script>

export default {
  layout: 'smtp',
  head() {
    return {
      title: `SMTP [${this.events.length}] | Buggregator`
    }
  },
  methods: {
    clearEvents() {
      this.$store.dispatch('events/clear', 'smtp')
    },
  },
  computed: {
    events() {
      return this.$store.getters['events/filteredByType']('smtp')
    },
    hasEvents() {
      return this.events.length > 0
    }
  },
}
</script>
