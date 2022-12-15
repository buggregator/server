<template>
  <div>
    <nav ref="header" class="breadcrumbs">
      <NuxtLink class="text-muted" :to="event.route.index">Profiler</NuxtLink>
      <div class="h-1 w-1">
        <svg class="fill-current" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 330 330">
          <path d="M251 154 101 4a15 15 0 1 0-22 22l140 139L79 304a15 15 0 0 0 22 22l150-150a15 15 0 0 0 0-22z"/>
        </svg>
      </div>
      <span>Event - {{ event.id }}</span>
    </nav>
    <main class="flex flex-col flex-grow items-center p-5">
      <h1>Flamechart</h1>
      <Flamegraph :event="event" />
    </main>
  </div>
</template>

<script>
import JsonChip from "@/Components/UI/JsonChip"
import ProfilerEvent from "../../app/Event/Profiler"
import Flamegraph from "@/Components/Events/Profiler/Show/Flamechart"

export default {
  components: {
    Flamegraph,
    JsonChip,
  },
  async asyncData({params, redirect, $api}) {
    const json = await $api.events.show(params.uuid)
    const event = new ProfilerEvent(json.payload, json.uuid, json.timestamp)
    if (!event) {
      redirect('/profiler')
    }

    return {event}
  },
  methods: {
    async deleteEvent() {
      await this.$store.dispatch('events/delete', this.event)
    }
  },
  computed: {
    date() {
      return this.event.date.fromNow()
    }
  }
}
</script>
