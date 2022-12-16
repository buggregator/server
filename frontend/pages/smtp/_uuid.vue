<template>
  <div class="flex flex-col flex-grow py-5 px-4 md:px-6 lg:px-8">
    <Info :event="event"/>
  </div>
</template>

<script>
import Info from "@/Components/Events/Smtp/Info"
import SmtpEvent from "@/app/Event/Smtp"

export default {
  layout: 'smtp',
  components: {Info},
  async asyncData({params, redirect, $api}) {
    const json = await $api.events.show(params.uuid)
    const event = new SmtpEvent(json.payload, json.uuid, json.timestamp)
    if (!event) {
      redirect('/smtp')
    }

    return {event}
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
