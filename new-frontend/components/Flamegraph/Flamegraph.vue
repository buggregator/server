<template>
  <div ref="flamegraph" class="profiler-flamegraph">
    <canvas ref="flameChartCanvas"></canvas>
  </div>
</template>

<script lang="ts">
import {FlameChart} from 'flame-chart-js';
import {defineComponent, PropType} from "vue";
import {ProfilerEdge} from "~/config/types";
import FlamegraphBuilder from "./FlamegraphBuilder";

const delay = (time: number): Promise<void> => new Promise(resolve => setTimeout(resolve, time))

export default defineComponent({
  props: {
    edges: {
      type: Array as PropType<ProfilerEdge[]>,
      required: true,
    },
  },
  emits: {'hover': null, 'hide': null},
  unmounted() {
    // Destroy the flamechart instance
    // this.flameChart.destroy();
  },
  mounted() {
    const canvas = this.$refs.flameChartCanvas as HTMLCanvasElement;

    const data = [new FlamegraphBuilder(this.edges).build()];

    delay(10).then(() => {
      const {width = 0, height = 0} = this.$refs.flamegraph.getBoundingClientRect();
      canvas.width = width;
      canvas.height = height;

      this.flameChart = new FlameChart({
        canvas,
        data,
        settings: {
          styles: {
            "main": {
              "blockHeight": 20
            }
          },
          options: {
            tooltip: (data, renderEngine, mouse) => {
              if (data === null) {
                this.$emit('hide')
              } else {
                this.$emit("hover", {
                  callee: data.data.source.name,
                  cost: data.data.source.cost,
                  position: {
                    x: mouse.x,
                    y: mouse.y,
                  },
                });
              }
            },
          },
        },
      });

      window.addEventListener('resize', () => {
        const {width, height} = this.$refs.flamegraph.getBoundingClientRect();
        this.flameChart.resize(width, height);
      });
    });
  }
});
</script>

<style lang="scss" scoped>
.profiler-flamegraph {
  @apply h-full;
}

.profiler-flamegraph canvas {
  @apply bg-gray-300;
}
</style>
