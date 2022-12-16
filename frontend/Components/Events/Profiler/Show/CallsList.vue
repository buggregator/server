<template>
  <div>
    <div class="flex items-stretch bg-gray-600 border-b border-gray-600 text-xs text-center py-2">
      <div class="flex-1 border-r border-gray-600">
        Function calls
      </div>
      <div class="w-24 border-r border-gray-600">
        CPU
      </div>
      <div class="w-12">
        Calls
      </div>
    </div>

    <CallsItem v-for="(edge, key) in sortedEdges"
               :key="key"
               :edge="edge"
               @hover="$emit('hover', $event)"
               @hide="$emit('hide')"
    />
  </div>
</template>

<script>
import CallsItem from "./CallsItem"

export default {
  components: {CallsItem},
  props: {
    event: Object,
  },
  computed: {
    sortedEdges() {
      return Object.entries(this.event.edges)
        .sort(([, a], [, b]) => b.cost.p_cpu - a.cost.p_cpu)
        .reduce((r, [k, v]) => ({...r, [k]: v}), {});
    }
  }
}
</script>
