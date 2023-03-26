<template>
  <div class="profiler-callstack">
    <header>
      <div class="cpu">CPU / Memory</div>
      <div class="calls">Calls</div>
    </header>

    <div class="profiler-callstack__calls">
      <CallStackRow
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
import CallStackRow from "./CallStackRow.vue";

export default defineComponent({
  components: { CallStackRow },
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
.profiler-callstack {
  > header {
    @apply flex items-stretch bg-gray-600 text-xs text-white text-center font-bold uppercase py-2;

    .callee {
      @apply flex-1 text-white;
    }

    .cpu {
      @apply flex-1 text-white;
    }

    .calls {
      @apply w-12;
    }
  }
}

.profiler-callstack__calls {
  @apply flex flex-col divide-y divide-gray-200 dark:divide-gray-600;
}
</style>
