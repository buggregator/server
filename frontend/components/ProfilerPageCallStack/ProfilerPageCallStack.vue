<template>
  <div class="profiler-page-callstack">
    <header class="profiler-page-callstack__header">
      <div class="profiler-page-callstack__header-cpu">CPU / Memory</div>
      <div class="profiler-page-callstack__header-calls">Calls</div>
    </header>

    <div class="profiler-page-callstack__calls">
      <ProfilerPageCallStackRow
        v-for="(edge, key) in sortedEdges"
        :key="key"
        :edge="edge"
        @hover="$emit('hover', $event)"
        @hide="$emit('hide')"
      />
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { Profiler } from "~/config/types";
import ProfilerPageCallStackRow from "~/components/ProfilerPageCallStackRow/ProfilerPageCallStackRow.vue";

export default defineComponent({
  components: { ProfilerPageCallStackRow },
  props: {
    event: {
      type: Object as PropType<Profiler>,
      required: true,
    },
  },
  emits: ["hover", "hide"],
  computed: {
    sortedEdges() {
      return Object.entries(this.event.edges)
        .sort(([, a], [, b]) => b.cost.p_cpu - a.cost.p_cpu)
        .reduce((r, [k, v]) => ({ ...r, [k]: v }), {});
    },
  },
});
</script>

<style lang="scss" scoped>
.profiler-page-callstack {
}

.profiler-page-callstack__header {
  @apply flex items-stretch bg-gray-600 text-xs text-white text-center font-bold uppercase py-2;
}

.profiler-page-callstack__header-cpu {
  @apply flex-1 text-white;
}

.profiler-page-callstack__header-calls {
  @apply w-12;
}

.profiler-page-callstack__calls {
  @apply flex flex-col divide-y divide-gray-200 dark:divide-gray-600;
}
</style>
