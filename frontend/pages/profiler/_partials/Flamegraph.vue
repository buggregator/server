<template>
  <div class="event--profiler__flamegraph" @mousemove="trackMousePosition">
    <div class="flamegraph" ref="flamegraph"></div>
  </div>
</template>

<script>
import {select} from "d3-selection"
import {flamegraph} from "d3-flame-graph";
import FlamegraphBuilder from "@/app/Profiler/FlamegraphBuilder"

export default {
  props: {
    event: Object,
    width: Number,
    cellHeight: {
      type: Number,
      default: 20
    }
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
      this.chart.width(width).update()
    }
  },
  methods: {
    trackMousePosition(e) {
      this.position.x = e.pageX;
      this.position.y = e.pageY;
    },
    onSpanHover(d) {
      this.$emit('hover', {
        name: d.data.name,
        cost: d.data.cost,
        position: this.position
      })
    },
    detailsHandler(d) {
      if (d === null) {
        this.$emit('hide')
      }
    },
    colorMapper(d, originalColor) {
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
    }
  },
  mounted() {
    const el = this.$refs.flamegraph
    const builder = new FlamegraphBuilder(this.event.edges)

    this.chart = flamegraph()
      .onHover(this.onSpanHover)
      .setDetailsHandler(this.detailsHandler)
      .sort(false)
      .inverted(true)
      .computeDelta(true)
      .setColorMapper(this.colorMapper)
      .cellHeight(this.cellHeight)
      .width(this.width)

    select(el)
      .datum(builder.build())
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
