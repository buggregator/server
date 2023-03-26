<template>
  <section class="stat-board">
    <div v-for="item in statItems" :key="item.title" class="stat-board__item">
      <h4 class="stat-board__item-name">
        {{ item.title }}
      </h4>

      <strong class="stat-board__item-value">
        {{ item.value }}
      </strong>
    </div>
  </section>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { ProfilerCost } from "~/config/types";
import { humanFileSize, formatDuration } from "~/utils/formats";

export default defineComponent({
  props: {
    cost: {
      type: Object as PropType<ProfilerCost>,
      required: true,
    },
  },
  computed: {
    statItems() {
      return [
        {
          title: "Calls",
          value: formatDuration(this.cost.ct || 0),
        },
        {
          title: "CPU time",
          value: formatDuration(this.cost.cpu || 0),
        },
        {
          title: "Wall time",
          value: formatDuration(this.cost.wt || 0),
        },
        {
          title: "Memory usage",
          value: humanFileSize(this.cost.mu || 0),
        },
        {
          title: "Change memory",
          value: humanFileSize(this.cost.pmu || 0),
        },
      ];
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.stat-board {
  @apply flex flex-col sm:flex-row sm:justify-between items-stretch sm:items-start;
  @apply sm:pt-5 sm:pb-4 sm:px-4 md:px-5;
  @apply bg-gray-200 dark:bg-gray-800;
  @apply divide-y sm:divide-y-0 sm:divide-x divide-gray-300 dark:divide-gray-500;
}

.stat-board__item {
  @apply pl-3 sm:px-6 py-3;
  &:first-child {
    @apply sm:pl-0;
  }

  &:last-child {
    @apply sm:pr-0;
  }
}

.stat-board__item-name {
  @apply text-gray-600 dark:text-gray-300 font-bold text-2xs mb-1 uppercase truncate;
}

.stat-board__item-value {
  @apply text-2xs sm:text-xs md:text-base truncate;
}
</style>
