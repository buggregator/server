<template>
  <section class="event--profiler__cards">
    <div class="item">
      <h4 class="item--name">Calls</h4>
      <strong class="item--value">
        {{ cost.ct || 0 }}
      </strong>
    </div>
    <div class="item">
      <h4 class="item--name">CPU time</h4>
      <strong class="item--value">
        {{ cpu }}
        <div v-if="cost.p_cpu">{{ cost.p_cpu }}%</div>
<!--        <div v-if="cost.d_cpu">{{ d_cpu }}</div>-->
      </strong>
    </div>
    <div class="item">
      <h4 class="item--name">Wall time</h4>
      <strong class="item--value">
        {{ wt }}
        <div v-if="cost.p_wt">{{ cost.p_wt }}%</div>
      </strong>
    </div>
    <div class="item">
      <h4 class="item--name">Memory usage</h4>
      <strong class="item--value">
        {{ mu }}
        <div v-if="cost.p_mu">{{ cost.p_mu }}%</div>
<!--        <div v-if="cost.d_mu">{{ d_mu }}</div>-->
      </strong>
    </div>
    <div class="item">
      <h4 class="item--name">Change memory</h4>
      <strong class="item--value">
        {{ pmu }}
        <div v-if="cost.p_pmu">{{ cost.p_pmu }}%</div>
<!--        <div v-if="cost.d_pmu">{{ d_pmu }}</div>-->
      </strong>
    </div>
  </section>
</template>

<script>
import {humanFileSize, formatDuration} from "@/Utils/converters"

export default {
  props: {
    cost: Object
  },
  computed: {
    cpu() {
      return formatDuration(this.cost.cpu || 0);
    },
    d_cpu() {
      return formatDuration(this.cost.d_cpu || 0);
    },
    wt() {
      return formatDuration(this.cost.wt || 0);
    },
    mu() {
      return humanFileSize(this.cost.mu || 0);
    },
    d_mu() {
      return humanFileSize(this.cost.d_mu || 0);
    },
    pmu() {
      return humanFileSize(this.cost.pmu || 0);
    },
    d_pmu() {
      return humanFileSize(this.cost.d_pmu || 0);
    }
  }
}
</script>

<style lang="scss">
.event--profiler__cards {
  @apply bg-gray-200 dark:bg-gray-800 pt-5 pb-4 px-4 md:px-5 flex flex-col sm:flex-row justify-between items-start divide-y sm:divide-y-0 sm:divide-x divide-gray-300 dark:divide-gray-500;

  .item {
    @apply flex-auto pb-5 sm:pb-0 sm:px-10 pt-5 sm:pt-0;

    &:first-child {
      @apply sm:px-0;
    }

    &--name {
      @apply text-gray-600 dark:text-gray-300 font-bold text-2xs mb-1 uppercase truncate;
    }

    &--value {
      @apply text-2xs sm:text-xs md:text-base truncate;
    }
  }
}
</style>
