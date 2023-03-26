<template>
  <div ref="flamegraph" class="flamegraph-board">
    <canvas ref="flameChartCanvas" class="flamegraph-board__canvas"></canvas>
  </div>
</template>

<script lang="ts">
import { FlameChart, FlameChartNode } from "flame-chart-js";
import { defineComponent, PropType } from "vue";
import { ProfilerEdges } from "~/config/types";
import FlamegraphBuilder from "~/utils/flamegraph-builder";

export default defineComponent({
  props: {
    edges: {
      type: Object as PropType<ProfilerEdges>,
      required: true,
    },
  },
  emits: { hover: null, hide: null },
  unmounted() {
    // Destroy the flamechart instance
    // this.flameChart.destroy();
  },
  mounted() {
    const flameData = new FlamegraphBuilder(this.edges).build();

    this.$nextTick(() => {
      this.renderChart(
        this.$refs.flameChartCanvas as HTMLCanvasElement,
        flameData
      );
    });
  },
  methods: {
    renderChart(canvas: HTMLCanvasElement, flameData: FlameChartNode) {
      const { width = 1, height = 1 } = (
        this.$refs.flamegraph as HTMLElement
      ).getBoundingClientRect();

      // eslint-disable-next-line no-param-reassign
      canvas.width = width;
      // eslint-disable-next-line no-param-reassign
      canvas.height = height;

      const flameChart = new FlameChart({
        canvas,
        data: [flameData],
        settings: {
          styles: {
            main: {
              blockHeight: 20,
            },
          },
          options: {
            tooltip: (data, _, mouse) => {
              if (data === null) {
                this.$emit("hide");
              } else {
                this.$emit("hover", {
                  callee: data.data.source.name,
                  cost: data.data.source.cost,
                  position: {
                    x: mouse?.x || 0,
                    y: mouse?.y || 0,
                  },
                });
              }
            },
          },
        },
      });

      window.addEventListener("resize", () => {
        const { width: windowWidth, height: windowHeight } = (
          this.$refs.flamegraph as HTMLElement
        ).getBoundingClientRect();
        flameChart.resize(windowWidth, windowHeight);
      });
    },
  },
});
</script>

<style lang="scss" scoped>
.flamegraph-board {
  @apply w-full h-full;
}

.flamegraph-board__canvas {
  @apply bg-gray-300 w-full h-full;
}
</style>
