<template>
  <div class="graphviz"></div>
</template>

<style lang="scss">
.graphviz {
  @apply flex justify-items-stretch items-stretch bg-white rounded;
  height: 500px;

  .graph {
    > polygon {
      @apply fill-transparent;
    }
  }

  .node {
    @apply cursor-pointer;

    &.pmu {
      > text {
        @apply fill-white;
      }

      > polygon {
        @apply fill-red-600 stroke-red-800;
      }

      &:hover > polygon {
        @apply fill-red-800;
      }
    }

    &.default {
      > text {
        @apply fill-gray-800;
      }

      > polygon {
        @apply fill-gray-200 stroke-gray-400;
      }

      &:hover > polygon {
        @apply fill-gray-300;
      }
    }

    > text {
      @apply font-bold text-sm;
    }

    > path {
      stroke-width: 3px;
    }
  }
}
</style>

<script>
import {wasmFolder} from "@hpcc-js/wasm";
import {select, selectAll} from "d3-selection"
import {graphviz} from "d3-graphviz"
import {humanFileSize, formatDuration} from "@/Utils/converters"

import DigraphBuilder from "@/app/Profiler/DigraphBuilder"

wasmFolder("https://cdn.jsdelivr.net/npm/@hpcc-js/wasm/dist");

function attributer(datum, index, nodes) {
  if (datum.tag == "svg") {
    // datum.attributes.fill = ''
  }
}

export default {
  props: {
    event: Object
  },
  data() {
    return {
      metric: 'p_cpu',
      threshold: 1
    }
  },
  watch: {
    threshold() {
      this.renderGraph()
    },
    metric() {
      this.renderGraph()
    }
  },
  methods: {
    buildDigraph() {
      const builder = new DigraphBuilder(this.event.edges)

      return builder.build(this.metric, this.threshold)
    },
    findEdge(name) {
      return Object.entries(this.event.edges)
        .find(([k, v]) => v.callee === name)[1] || null
    },
    nodeHandler() {
      selectAll("g.node").on("mouseover", (e, tag) => {
        const el = e.target.parentNode
        const edge = this.findEdge(tag.key)
        this.$emit('hover', {
          name: edge.callee,
          cost: edge.cost,
          position: {
            x: e.pageX,
            y: e.pageY,
          }
        });
      }).on("mouseout", (e) => {
        const el = e.target.parentNode
        this.$emit('hide')
      })
    },
    renderGraph() {
      this.graph = select(this.$el)
        .graphviz()
        .width('100%')
        .height('100%')
        .attributer(attributer)
        .fit(true)
        .renderDot(this.buildDigraph(), this.nodeHandler)
    }
  },
  mounted() {
    this.renderGraph()
  },
  beforeDestroy() {
    this.graph.destroy()
  }
}
</script>
