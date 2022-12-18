<template>
  <div class="event--profiler__flamegraph" @mousemove="trackMousePosition">
    <div class="flamegraph" ref="flamegraph"></div>
  </div>
</template>

<script>
import {select} from "d3-selection"
import {flamegraph} from "d3-flame-graph";

export default {
  props: {
    event: Object,
    width: Number,
  },
  data() {
    return {
      position: {
        x: 0,
        y: 0,
      },
    }
  },
  watch: {
    width(width) {
      this.chart
        .width(width)
        .update()
    }
  },
  methods: {
    trackMousePosition(e) {
      this.position.x = e.clientX;
      this.position.y = e.clientY;
    }
  },
  mounted() {
    const el = this.$refs.flamegraph
    this.chart = flamegraph()
      .onHover((d) => {
        this.$emit('hover', {
          name: d.data.name,
          cost: d.data.cost,
          position: this.position
        })
      })
      .setDetailsHandler((d) => {
        if (d === null) {
          this.$emit('hide')
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

<style lang="scss">
.event--profiler__flamegraph {
  @apply flex flex-col;

  .flamegraph {
    .frame {
      @apply cursor-pointer flex content-center;

      &:hover > rect {
        @apply fill-blue-900;
      }

      stroke: rgba(255, 158, 44, .2);
    }

    .d3-flame-graph-label {
      font-size: 80%;
      @apply px-2;
    }
  }
}
</style>
