<template>
  <div class="flex items-stretch border-b border-gray-600 hover:bg-gray-900 cursor-pointer"
       @mouseover="onHover(edge)"
       @mouseout="$emit('hide')"
  >
    <div class="text-sm flex-1 py-1 text-right pr-2 truncate border-r border-gray-600">
      {{ edge.callee }}
    </div>
    <div class="w-24 text-center text-xs relative border-r border-gray-600">
      <div class="h-full bg-red-800 text-sm opacity-70" :style="{width: `${p_cpu}%`}"/>
      <div class="h-full bg-purple-800 text-sm opacity-40 -mt-7" :style="{width: `${p_mu}%`}"/>
      <div class="absolute inset-0 py-1 z-1">
        {{ edge.cost.p_cpu }}%
      </div>
    </div>
    <div class="w-12 text-center text-xs py-1 z-1">
      {{ edge.cost.ct }}
    </div>
  </div>
</template>

<script>
import {humaCpuUsage, humanFileSize, formatDuration} from "@/Utils/converters"

export default {
  props: {
    edge: Object,
  },
  methods: {
    onHover(edge) {
      this.$emit('hover', {
        name: edge.callee,
        cost: edge.cost,
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
