<template>
  <div>
    <div class="flex items-stretch bg-gray-600 border-b border-gray-600 text-xs text-center py-2">
      <div class="flex-1 border-r border-gray-600">
        Function calls
      </div>
      <div class="w-24 border-r border-gray-600">
        % CPU
      </div>
      <div class="w-24 border-r border-gray-600">
        CPU
      </div>
      <div class="w-12">
        Calls
      </div>
    </div>

    <div class="flex items-stretch border-b border-gray-600" v-for="edge in sortedEdges">
      <div class="text-sm flex-1 py-1 text-right pr-2 truncate border-r border-gray-600">
        {{ edge.callee }}
      </div>
      <div class="w-24 text-center text-xs relative border-r border-gray-600">
        <div class="absolute inset-0 py-1">
          {{ edge.cost.p_cpu }}%
        </div>
        <div class="h-full bg-gray-800 text-sm" :style="{width: `${edge.cost.p_cpu}%`}">
        </div>
      </div>
      <div class="w-12 text-center text-xs py-1">
        {{ edge.cost.cpu }}
      </div>
      <div class="w-12 text-center text-xs py-1">
        {{ edge.cost.ct }}
      </div>
    </div>
  </div>
</template>

<script>
export default {
    props: {
        event: Object,
    },
  computed: {
      sortedEdges() {
          return Object.entries(this.event.edges)
            .sort(([,a],[,b]) => b.cost.p_cpu - a.cost.p_cpu)
            .reduce((r, [k, v]) => ({ ...r, [k]: v }), {});
      }
  }
}
</script>
