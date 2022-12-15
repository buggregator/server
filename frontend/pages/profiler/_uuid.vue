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
    <main class="flex h-full">
      <div class="w-1/3 border-r border-gray-600" ref="calls">
        <PerfectScrollbar :style="{height: menuHeight}">
          <CallsList :event="event" />
        </PerfectScrollbar>
      </div>
      <div class="w-2/3 overflow-x-hidden">
        <section>
          <Cards class="w-full dark:bg-gray-800 mb-5" :cost="event.peaks" />
        </section>

        <section class="p-5 bg-gray-800">
          <h1 class="text-lg font-bold mb-3">Flamechart</h1>
          <Flamegraph :event="event" :width="width" />
        </section>
      </div>
    </main>
  </div>
</template>

<script>
import {PerfectScrollbar} from 'vue2-perfect-scrollbar'
import ImageExport from "@/Components/UI/ImageExport"
import Cards from "@/Components/Events/Profiler/Show/Cards"
import JsonChip from "@/Components/UI/JsonChip"
import ProfilerEvent from "../../app/Event/Profiler"
import Flamegraph from "@/Components/Events/Profiler/Show/Flamegraph"
import CallsList from "@/Components/Events/Profiler/Show/CallsList"

export default {
  components: {
    CallsList,
    Flamegraph, Cards,
    JsonChip, ImageExport,
    PerfectScrollbar
  },
  data() {
    return {
      exportableEl: null,
      menuHeight: 0,
      width: 0,
    }
  },
  async asyncData({params, redirect, $api}) {
    const json = await $api.events.show(params.uuid)
    const event = new ProfilerEvent(json.payload, json.uuid, json.timestamp)
    if (!event) {
      redirect('/profiler')
    }

    return {event}
  },
  mounted() {
    this.exportableEl = this.$el
    this.calculateMenuHeight()
    this.width = document.documentElement.clientWidth - this.$refs.calls.offsetWidth - 63 - 20 - 120
    window.addEventListener('resize', (event) => {
      this.calculateMenuHeight()
      this.width = document.documentElement.clientWidth - this.$refs.calls.offsetWidth - 63 - 120 - 20
    });
  },
  methods: {
    async deleteEvent() {
      await this.$store.dispatch('events/delete', this.event)
    },
    calculateMenuHeight() {
      const headerHeight = this.$refs.header ? parseInt(this.$refs.header.offsetHeight) : 0
      this.menuHeight = (window.innerHeight - headerHeight - 2) + 'px'
    }
  },
  computed: {
    date() {
      return this.event.date.fromNow()
    }
  }
}
</script>
