<template>
  <div class="flex flex-col">
    <div class="flamechart" ref="flamechart"></div>

    <div v-if="details" class="border border-gray-600 bg-gray-800">
      <h4 class="px-4 pt-4 pb-0 font-bold">{{ details.name }}</h4>
      <Cards v-if="details.cost" :cost="details.cost" />
    </div>
  </div>
</template>

<script>
import {select} from "d3-selection"
import {flamegraph} from "d3-flame-graph";
import Cards from "@/Components/Events/Profiler/Show/Cards"

export default {
  components: {Cards},
  props: {
    event: Object,
    width: Number,
  },
  data() {
    return {
      details: null,
    }
  },
  watch: {
    width(width) {
      this.chart
        .width(width)
        .update()
    }
  },
  mounted() {
    const el = this.$refs.flamechart
    this.chart = flamegraph()
      .onHover((d) => {
        this.details = {
          name: d.data.name,
          cost: d.data.cost,
        }
      })
      .setDetailsHandler((d) => {
        if (d === null) {
          this.details = d
        }
      })
      .sort(false)
      .inverted(true)
      .computeDelta(true)
      .setColorMapper(function (d, originalColor) {
        if (d.data.name === 'main()') {
          return '#333333'
        }

        if (d.data.cost.p_cpu >= 60) {
          return '#6C0000'
        }

        if (d.data.cost.p_cpu <= 10) {
          return '#009348'
        }

        return '#1f2937';
      })
      .cellHeight(25)
      .width(this.width)

    let datum = {
      0: {name: '_', value: 0, children: []}
    }

    const edges = Object.entries(this.event.edges)

    for (const [key, edge] of edges) {
      let parent = edge.caller || null
      let func = edge.callee || null

      let value = edge.cost.cpu || 0

      if (!datum.hasOwnProperty(func)) {
        datum[func] = {
          name: func,
          value: value,
          cost: edge.cost,
          children: []
        }
      }

      if (parent && !datum.hasOwnProperty(parent)) {
        datum[parent] = {
          name: parent,
          value: value,
          cost: edge.cost,
          children: []
        }
      }

      datum[parent || 0].children.push(datum[func])
    }

    select(el)
      .datum(datum['main()'])
      .call(this.chart)
  },
  destroyed() {
    this.chart.destroy()
  }
}
</script>
