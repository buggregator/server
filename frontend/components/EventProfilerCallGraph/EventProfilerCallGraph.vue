<template>
  <div class="graphviz--wrapper" :class="{ fullscreen: isFullscreen }">
    <div ref="graphviz" class="graphviz"></div>
    <div class="graphviz--toolbar">
      <button title="Full screen" @click="isFullscreen = !isFullscreen">
        <icon-svg name="fullscreen" class="w-4 h-4 fill-blue-500" />
      </button>
      <button
        class="text-xs uppercase text-gray-600"
        :class="{ 'font-bold': metric == 'cpu' }"
        @click="metric = 'cpu'"
      >
        CPU
      </button>
      <button
        class="text-xs uppercase text-gray-600"
        :class="{ 'font-bold': metric == 'pmu' }"
        @click="metric = 'pmu'"
      >
        Memory change
      </button>
      <button
        class="text-xs uppercase text-gray-600"
        :class="{ 'font-bold': metric == 'mu' }"
        @click="metric = 'mu'"
      >
        Memory usage
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { selectAll } from "d3-selection";
import { graphviz } from "d3-graphviz";
import { Graphviz } from "@hpcc-js/wasm/graphviz";

import IconSvg from "~/components/IconSvg/IconSvg.vue";

import { defineComponent, PropType } from "vue";
import { Profiler, ProfilerEdge } from "~/config/types";
import { addSlashes, DigraphBuilder } from "./DigraphBuilder";

export default defineComponent({
  components: { IconSvg },
  props: {
    event: {
      type: Object as PropType<Profiler>,
      required: true,
    },
    threshold: {
      type: Number,
      default: 1,
    },
  },
  emits: ["hover", "hide"],
  data() {
    return {
      isFullscreen: false,
      metric: "cpu",
    };
  },
  watch: {
    threshold(): void {
      this.renderGraph();
    },
    metric(): void {
      this.renderGraph();
    },
  },
  created(): void {
    this.renderGraph();
  },
  beforeUnmount() {
    this.graph.destroy();
  },
  methods: {
    buildDigraph(): string {
      const builder = new DigraphBuilder(this.event.edges);

      return builder.build(this.metric, this.threshold);
    },

    findEdge(name: string): ProfilerEdge | null {
      const found = Object.values(this.event.edges).filter(
        v => addSlashes(v.callee) === name
      );
      if (!found || found.length === 0) {
        return null;
      }

      return found[0] || null;
    },
    nodeHandler(): void {
      selectAll("g.node")
        .on("mouseover", (e, tag) => {
          const edge = this.findEdge(tag.key);
          if (!edge) {
            return;
          }

          this.$emit("hover", {
            callee: edge.callee,
            cost: edge.cost,
            position: {
              x: e.pageX,
              y: e.pageY,
            },
          });
        })
        .on("mouseout", () => {
          this.$emit("hide");
        });
    },
    renderGraph(): void {
      Graphviz.load().then(() => {
        this.graph = graphviz(this.$refs.graphviz, {})
          .width("100%")
          .height("100%")
          .fit(true)
          .renderDot(this.buildDigraph(), this.nodeHandler);
      });
    },
  },
});
</script>

<style lang="scss">
.graphviz {
  @apply flex-1 justify-items-stretch items-stretch bg-gray-700;

  .graph {
    > polygon {
      @apply fill-gray-700;
    }
  }

  &--wrapper {
    @apply relative flex rounded border border-gray-900 h-full;

    &.fullscreen {
      @apply rounded-none mt-0 top-0 left-0 fixed w-full h-full;
      z-index: 9998;
    }
  }

  &--toolbar {
    @apply absolute top-5 left-5 flex bg-gray-200 p-2 rounded gap-x-5;
    z-index: 9999;
  }

  .graph {
    > path {
      @apply fill-transparent;
    }
  }

  .edge {
    > path {
      stroke-width: 2px;
    }

    > text {
      @apply fill-white ml-2;
    }
  }

  .node {
    @apply cursor-pointer;

    > path {
      @apply rounded;
      stroke-width: 1;
    }

    &.pmu {
      > text {
        @apply fill-white;
      }

      > path {
        @apply fill-red-600 stroke-red-800;
      }

      &:hover > path {
        @apply fill-red-800;
      }
    }

    &.default {
      > text {
        @apply fill-gray-700;
      }

      > path {
        @apply fill-gray-200 stroke-gray-400;
      }

      &:hover > path {
        @apply fill-gray-300;
      }
    }

    > text {
      @apply font-bold text-sm;
    }
  }
}
</style>
