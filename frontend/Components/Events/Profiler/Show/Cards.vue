<template>
  <section class="event-profiler-cards">
    <div class="event-profiler-cards__item sm:px-0">
      <h4 class="event-profiler-cards__item-name">Number of calls</h4>
      <strong class="event-profiler-cards__item-value md:text-md">
        {{ cost.ct || 0 }}
      </strong>
    </div>
    <div class="event-profiler-cards__item pt-5 sm:pt-0">
      <h4 class="event-profiler-cards__item-name">CPU time</h4>
      <strong class="event-profiler-cards__item-value md:text-md">
        {{ cpu }}
        <span v-if="cost.p_cpu">- {{ cost.p_cpu }}%</span>
      </strong>
    </div>
    <div class="event-profiler-cards__item pt-5 sm:pt-0">
      <h4 class="event-profiler-cards__item-name">Wall time</h4>
      <strong class="event-profiler-cards__item-value md:text-md">
        {{ wt }}
        <span v-if="cost.p_wt">- {{ cost.p_wt }}%</span>
      </strong>
    </div>
    <div class="event-profiler-cards__item pt-5 sm:pt-0">
      <h4 class="event-profiler-cards__item-name">Change in PHP memory usage</h4>
      <strong class="event-profiler-cards__item-value md:text-md">
        {{ mu }}
        <span v-if="cost.p_mu">- {{ cost.p_mu }}%</span>
      </strong>
    </div>
    <div class="event-profiler-cards__item pt-5 sm:pt-0">
      <h4 class="event-profiler-cards__item-name">Peak PHP memory usage</h4>
      <strong class="event-profiler-cards__item-value md:text-md">
        {{ pmu }}
        <span v-if="cost.p_pmu">- {{ cost.p_pmu }}%</span>
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
    wt() {
      return formatDuration(this.cost.wt || 0);
    },
    mu() {
      return humanFileSize(this.cost.mu || 0);
    },
    pmu() {
      return humanFileSize(this.cost.pmu || 0);
    }
  }
}
</script>
