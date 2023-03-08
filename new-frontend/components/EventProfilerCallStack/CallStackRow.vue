<template>
  <div
      class="profiler-callstack-item"
      @mouseover="onHover($event, edge)"
      @mouseout="$emit('hide')"
  >
    <div class="usage">
      <div class="usage--cpu" :style="{ width: `${p_cpu}%` }"/>
      <div class="usage--memory" :style="{ width: `${p_mu}%` }"/>
      <div class="usage--title">
        {{ edge.cost.p_cpu }}% / {{ edge.cost.p_mu }}%
      </div>
    </div>
    <div class="calls">{{ edge.cost.ct }}</div>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {ProfilerEdge} from "~/config/types";

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
        name: edge.callee,
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
.profiler-callstack-item {
  @apply flex items-stretch border-b border-gray-600 hover:bg-gray-900 cursor-pointer;

  > .callee {
    @apply text-xs flex-1 py-1 text-right pr-2 truncate border-r border-gray-600;
  }

  > .calls {
    @apply w-12 text-center text-xs py-1;
  }

  > .usage {
    @apply flex-1 text-center text-xs relative border-r border-gray-600;
  }

  .usage {
    &--cpu {
      @apply h-full bg-red-800 text-sm opacity-70;
    }

    &--memory {
      @apply h-full bg-purple-800 text-sm opacity-40 -mt-6;
    }

    &--title {
      @apply absolute inset-0 py-1;
    }
  }
}
</style>
