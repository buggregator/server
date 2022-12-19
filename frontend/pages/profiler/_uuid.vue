<template>
  <div class="page-profiler">
    <main ref="main">
      <section class="call-stack__wrapper" ref="calls">
        <PerfectScrollbar :style="{height: callStackHeight}">
          <CallsList :event="event" @hover="showEdge" @hide="hideEdge"/>
        </PerfectScrollbar>
      </section>
      <div class="info__wrapper" ref="info">
        <section class="p-5 bg-gray-200  bg-gray-800">
          <Cards :cost="event.peaks"/>
        </section>

        <section class="p-5 bg-gray-200 bg-gray-800">
          <Tabs>
            <Tab title="Call graph">
              <div class="my-3">
                <div class="flex gap-x-5">
                  <button class="text-xs uppercase text-gray-600" @click="graphMetric = 'p_cpu'"
                          :class="{'text-gray-200': graphMetric == 'p_cpu'}">
                    CPU
                  </button>
                  <button class="text-xs uppercase text-gray-600" @click="graphMetric = 'p_pmu'"
                          :class="{'text-gray-200': graphMetric == 'p_pmu'}">
                    Memory change
                  </button>
                  <button class="text-xs uppercase text-gray-600" @click="graphMetric = 'p_mu'"
                          :class="{'text-gray-200': graphMetric == 'p_mu'}">
                    Memory usage
                  </button>
                </div>
              </div>
              <Graph :event="event" :metric="graphMetric" @hover="showEdge" @hide="hideEdge"/>
            </Tab>
            <Tab title="Flamechart">
              <FlameGraph :event="event" :width="width" @hover="showEdge" @hide="hideEdge"/>
            </Tab>
          </Tabs>
        </section>
      </div>
    </main>

    <CallInfo v-if="edge" :edge="edge"/>
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
import Tab from "@/Components/UI/Tab"
import Tabs from "@/Components/UI/Tabs"

export default {
  components: {
    CallInfo,
    Graph,
    CallsList,
    FlameGraph, Cards,
    JsonChip, ImageExport,
    PerfectScrollbar,
    Tab, Tabs
  },
  head() {
    return {
      title: `Profiler > ${this.event.uuid} | Buggregator`
    }
  },
  data() {
    return {
      callStackHeight: 0,
      width: 0,
      graphMetric: 'p_cpu',
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
    this.calculateCallStackHeight()
    this.calculateFlamechartWidth()
    window.addEventListener('resize', (event) => {
      this.calculateCallStackHeight()
      this.calculateFlamechartWidth()
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
    calculateFlamechartWidth() {
      this.width = document.documentElement.clientWidth - this.$refs.calls.offsetWidth - 63 - 120 - 20
    },
    calculateCallStackHeight() {
      this.callStackHeight = Math.max(window.innerHeight - 2) + 'px'
    }
  },
  computed: {
    date() {
      return this.event.date.fromNow()
    }
  }
}
</script>
