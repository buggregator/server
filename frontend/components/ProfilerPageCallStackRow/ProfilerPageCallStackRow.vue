<template>
  <div
    class="profiler-page-call-stack-row"
    @mouseover="onHover($event, edge)"
    @mouseout="$emit('hide')"
  >
    <div class="profiler-page-call-stack-row__usage">
      <div
        class="profiler-page-call-stack-row__usage-cpu"
        :style="{ width: `${p_cpu}%` }"
      />
      <div
        class="profiler-page-call-stack-row__usage-memory"
        :style="{ width: `${p_mu}%` }"
      />
      <div class="profiler-page-call-stack-row__usage-title">
        {{ edge.cost.p_cpu }}% / {{ edge.cost.p_mu }}%
      </div>
    </div>
    <div class="profiler-page-call-stack-row__calls">{{ edge.cost.ct }}</div>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { ProfilerEdge } from "~/config/types";

export default defineComponent({
  props: {
    edge: {
      type: Object as PropType<ProfilerEdge>,
      required: true,
    },
  },
  emits: ["hover", "hide"],
  computed: {
    p_cpu(): number {
      return Math.min(100, this.edge.cost.p_cpu);
    },
    p_mu(): number {
      return Math.min(100, this.edge.cost.p_mu);
    },
  },
  methods: {
    onHover($event: MouseEvent, edge: ProfilerEdge) {
      this.$emit("hover", {
        callee: edge.callee,
        cost: edge.cost,
        position: {
          x: $event.pageX,
          y: $event.pageY,
        },
      });
    },
  },
});
</script>

<style lang="scss" scoped>
.profiler-page-call-stack-row {
  @apply flex items-stretch hover:bg-gray-200 dark:hover:bg-gray-600 cursor-pointer divide-x divide-gray-200 dark:divide-gray-600;
}

.profiler-page-call-stack-row__calls {
  @apply w-12 text-center text-xs py-1;
}

.profiler-page-call-stack-row__usage {
  @apply flex-1 text-center text-xs relative;
}

.profiler-page-call-stack-row__usage-cpu {
  @apply h-full bg-red-900 text-sm opacity-60;
}

.profiler-page-call-stack-row__usage-memory {
  @apply h-full bg-purple-800 text-sm opacity-40 -mt-6;
}

.profiler-page-call-stack-row__usage-title {
  @apply absolute inset-0 py-1 text-blue-900 dark:text-gray-200 font-bold;
}
</style>
