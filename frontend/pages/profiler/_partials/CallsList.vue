<template>
  <div class="event--profiler__callitems">
    <header>
      <div class="callee">Function calls</div>
      <div class="cpu">CPU</div>
      <div class="calls">Calls</div>
    </header>

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

<style lang="scss">
.event--profiler__callitems {
  > header {
    @apply flex items-stretch bg-gray-600 border-b border-gray-600 text-xs text-center py-2;

    .callee {
      @apply flex-1 border-r border-gray-600;
    }

    .cpu {
      @apply w-24 border-r border-gray-600;
    }

    .calls {
      @apply w-12;
    }
  }
}
</style>
