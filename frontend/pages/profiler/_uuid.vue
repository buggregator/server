<template>
  <div class="page-profiler">
    <main ref="main">
      <section class="call-stack__wrapper" ref="calls">
        <PerfectScrollbar :style="{height: menuHeight}">
          <CallsList :event="event" @hover="showEdge" @hide="hideEdge"/>
        </PerfectScrollbar>
      </section>
      <div class="info__wrapper">
        <section class="p-5 bg-gray-200  bg-gray-800">
          <Cards :cost="event.peaks"/>
        </section>

        <section class="p-5 bg-gray-200  bg-gray-800">
          <h1 class="text-lg font-bold mb-3">Flamechart</h1>
          <FlameGraph :event="event" :width="width" @hover="showEdge" @hide="hideEdge"/>
        </section>

        <section class="p-5 bg-gray-200 bg-gray-800">
          <h1 class="text-lg font-bold mb-3">Call graph</h1>
          <Graph :event="event" @hover="showEdge" @hide="hideEdge"/>
        </section>
      </div>
    </main>

    <CallInfo v-if="edge" :edge="edge" />
  </div>
</template>

<style lang="scss">
.page-profiler {
  @apply relative;

  > main {
    @apply flex flex-col md:flex-row;
  }

  .call-stack__wrapper {
    @apply w-full md:w-1/6 border-r border-gray-300 dark:border-gray-500;
  }

  .info__wrapper {
    @apply w-full md:w-5/6 divide-y divide-gray-300 dark:divide-gray-500;
  }
}
</style>

<script>
import {PerfectScrollbar} from 'vue2-perfect-scrollbar'
import ImageExport from "@/Components/UI/ImageExport"
import JsonChip from "@/Components/UI/JsonChip"
import ProfilerEvent from "@/app/Event/Profiler"
import FlameGraph from "./_partials/Flamegraph"
import CallsList from "./_partials/CallsList"
import Graph from "./_partials/Graph"
import CallInfo from "./_partials/CallInfo"
import Cards from "@/Components/Events/Profiler/_partials/Cards"

export default {
  components: {
    CallInfo,
    Graph,
    CallsList,
    FlameGraph, Cards,
    JsonChip, ImageExport,
    PerfectScrollbar
  },
  head() {
    return {
      title: `Profiler > ${this.event.uuid} | Buggregator`
    }
  },
  data() {
    return {
      menuHeight: 0,
      width: 0,
      edge: null
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
    this.calculateMenuHeight()
    this.width = document.documentElement.clientWidth - this.$refs.calls.offsetWidth - 63 - 20 - 120
    window.addEventListener('resize', (event) => {
      this.calculateMenuHeight()
      this.width = document.documentElement.clientWidth - this.$refs.calls.offsetWidth - 63 - 120 - 20
    });
  },
  methods: {
    showEdge(edge) {
      this.edge = edge
    },
    hideEdge() {
      this.edge = null
    },
    async deleteEvent() {
      await this.$store.dispatch('events/delete', this.event)
    },
    calculateMenuHeight() {
      this.menuHeight = (this.$refs.main.offsetHeight - 2) + 'px'
    }
  },
  computed: {
    date() {
      return this.event.date.fromNow()
    }
  }
}
</script>
