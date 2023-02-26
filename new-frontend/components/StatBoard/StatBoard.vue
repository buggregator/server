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

export default defineComponent({
  props: {
    cost: {
      type: Object as PropType<ProfilerCost>,
      required: true,
    },
  },
  computed: {
    statItems() {
      console.log("cost", this.cost);
      return [
        {
          title: "Calls",
          value: this.formatDuration(this.cost.ct || 0),
        },
        {
          title: "CPU time",
          value: this.formatDuration(this.cost.cpu || 0),
        },
        {
          title: "Wall time",
          value: this.formatDuration(this.cost.wt || 0),
        },
        {
          title: "Memory usage",
          value: this.humanFileSize(this.cost.mu || 0),
        },
        {
          title: "Change memory",
          value: this.humanFileSize(this.cost.pmu || 0),
        },
      ];
    },
  },
  methods: {
    formatDuration(inputMs: number) {
      let ms = inputMs;
      if (ms < 0) {
        ms = -ms;
      }

      ms /= 1_000;

      const time = {
        d: Math.floor(ms / 86_400_000),
        h: Math.floor(ms / 3_600_000) % 24,
        m: Math.floor(ms / 60_000) % 60,
        s: Math.floor(ms / 1_000) % 60,
        ms: ms % 1_000,
      };

      return Object.entries(time)
        .filter((val) => val[1] !== 0)
        .map((val) => `${val[1].toFixed(4)} ${val[1] !== 1 ? val[0] : val[0]}`)
        .join(", ");
    },
    humanFileSize(inputBytes: number) {
      let bytes = inputBytes;
      const thresh = 1024;

      if (Math.abs(bytes) < thresh) {
        return `${bytes} B`;
      }

      const units = ["KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"];
      let u = -1;
      const r = 10 ** 1;

      do {
        bytes /= thresh;
        u += 1;
      } while (
        Math.round(Math.abs(bytes) * r) / r >= thresh &&
        u < units.length - 1
      );

      return `${bytes.toFixed(1)} ${units[u]}`;
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.stat-board {
  @apply bg-gray-200 dark:bg-gray-800 pt-5 pb-4 px-4 md:px-5 flex flex-col sm:flex-row justify-between items-start divide-y sm:divide-y-0 sm:divide-x divide-gray-300 dark:divide-gray-500;
}

.stat-board__item {
  @apply sm:pb-0 sm:px-10 pt-5 sm:pt-0 flex-auto pb-5 sm:pb-0 sm:px-10 pt-5 sm:pt-0;

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
