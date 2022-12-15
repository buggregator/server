<template>
  <div class="w-full flamechart"></div>
</template>

<script>
import {select} from "d3-selection"
import {flamegraph} from "d3-flame-graph";

export default {
  props: {
    event: Object,
  },
  mounted() {
    const el = this.$el
    this.chart = flamegraph()
      .sort(false)
      .inverted(true)
      .computeDelta(true)
      .setColorMapper(function (d, originalColor) {
        if (d.data.name === 'main()') {
          return '#333333'
        }

        if (d.data.percent > 80) {
          return '#6C0000'
        }

        if (d.data.percent > 10) {
          return '#1f2937'
        }

        return '#009348';
      })
      .cellHeight(20)
      .width(el.offsetWidth)

    let datum = {
      0: {name: '_', value: 0, percent: 0, children: []}
    }

    const edges = Object.entries(this.event.edges)
    const maxCpu = this.event.peaks.cpu


    for (const [key, edge] of edges) {
      let parent = edge.caller || null
      let func = edge.callee || null

      if (!datum.hasOwnProperty(func)) {
        datum[func] = {
          name: func,
          value: edge.cost.cpu || 0,
          percent: Math.floor(edge.cost.cpu / maxCpu * 100),
          children: []
        }
      }

      if (parent && !datum.hasOwnProperty(parent)) {
        datum[parent] = {
          name: parent,
          value: edge.cost.cpu || 0,
          percent: Math.floor(edge.cost.cpu / maxCpu * 100),
          children: []
        }
      }

      datum[parent || 0].children.push(datum[func])
    }

    select(this.$el)
      .datum(datum['main()'])
      .call(this.chart)

    window.onresize = (e) => {
      chart.width(el.offsetWidth)
      chart.update()
    }
  },
  destroyed() {
    this.chart.destroy()
  }
}
</script>
