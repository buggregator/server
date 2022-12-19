<template>
  <div class="event--profiler__callitem" @mouseover="onHover($event, edge)" @mouseout="$emit('hide')">
<!--    <div class="callee">{{ edge.callee }}</div>-->
    <div class="usage">
      <div class="usage--cpu" :style="{width: `${p_cpu}%`}"/>
      <div class="usage--memory" :style="{width: `${p_mu}%`}"/>
      <div class="usage--title">{{ edge.cost.p_cpu }}% / {{ edge.cost.p_mu }}%</div>
    </div>
    <div class="calls">{{ edge.cost.ct }}</div>
  </div>
</template>

<script>
import {humanFileSize, formatDuration} from "@/Utils/converters"

export default {
  props: {
    edge: Object,
  },
  methods: {
    onHover($event, edge) {
      this.$emit('hover', {
        name: edge.callee,
        cost: edge.cost,
        position: {
          x: $event.pageX,
          y: $event.pageY,
        }
      });
    }
  },
  computed: {
    p_cpu() {
      return Math.min(100, this.edge.cost.p_cpu)
    },
    p_mu() {
      return Math.min(100, this.edge.cost.p_mu)
    },
    cpu() {
      return formatDuration(this.edge.cost.cpu)
    },
    mu() {
      return humanFileSize(this.edge.cost.mu)
    }
  }
}
</script>

<style lang="scss">
.event--profiler__callitem {
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
