<template>
  <div class="graphviz--wrapper" :class="{fullscreen: isFullscreen}">
    <div class="graphviz" ref="graphviz"></div>
    <div class="graphviz--toolbar">
      <button @click="isFullscreen = !isFullscreen" title="Full screen">
        <FullscreenIcon class="w-4 h-4 fill-blue-500"/>
      </button>
    </div>
  </div>
</template>

<script>
import {wasmFolder} from "@hpcc-js/wasm";
import {select, selectAll} from "d3-selection"
import {graphviz} from "d3-graphviz"

import {addSlashes, DigraphBuilder} from "@/app/Profiler/DigraphBuilder"
import FullscreenIcon from "@/Components/UI/Icons/FullscreenIcon"

wasmFolder("https://cdn.jsdelivr.net/npm/@hpcc-js/wasm/dist");

export default {
  components: {FullscreenIcon},
  props: {
    event: Object,
    metric: {
      type: String,
      default: 'p_cpu'
    },
    threshold: {
      type: Number,
      default: 1
    }
  },
  data() {
    return {
      isFullscreen: false,
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
      const found = Object.entries(this.event.edges)
        .find(([k, v]) => addSlashes(v.callee) === name)

      if (!found || found.length === 0) {
        return null
      }

      return found[1] || null
    },
    nodeHandler() {
      selectAll("g.node").on("mouseover", (e, tag) => {
        const edge = this.findEdge(tag.key)
        if (!edge) {
          return
        }

        this.$emit('hover', {
          name: edge.callee,
          cost: edge.cost,
          position: {
            x: e.pageX,
            y: e.pageY,
          }
        });
      }).on("mouseout", (e) => {
        this.$emit('hide')
      })
    },
    renderGraph() {
      this.graph = select(this.$refs.graphviz)
        .graphviz()
        .width('100%')
        .height('100%')
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

<style lang="scss">
.graphviz {
  @apply flex-1 justify-items-stretch items-stretch bg-white rounded;

  &--wrapper {
    @apply relative flex;
    height: 600px;

    &.fullscreen {
      z-index: 9998;
      width: 100%;
      height: 100%;
      position: fixed;
      top: 0;
      left: 0;
    }
  }

  &--toolbar {
    @apply absolute top-5 right-5 flex flex-col bg-gray-200 p-2 rounded;
    z-index: 9999;
  }

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
