<template>
  <section class="event-profiler-cards">
    <div class="event-profiler-cards__item sm:px-0">
      <h4 class="event-profiler-cards__item-name">Calls</h4>
      <strong class="event-profiler-cards__item-value md:text-md">{{ event.peaks.ct }}</strong>
    </div>
    <div class="event-profiler-cards__item pt-5 sm:pt-0">
      <h4 class="event-profiler-cards__item-name">CPU</h4>
      <strong class="event-profiler-cards__item-value md:text-md">{{ event.peaks.cpu }}</strong>
    </div>
    <div class="event-profiler-cards__item pt-5 sm:pt-0">
      <h4 class="event-profiler-cards__item-name">Instant memory</h4>
      <strong class="event-profiler-cards__item-value md:text-md">{{ event.peaks.mu }}</strong>
    </div>
    <div class="event-profiler-cards__item pt-5 sm:pt-0">
      <h4 class="event-profiler-cards__item-name">Memory</h4>
      <strong class="event-profiler-cards__item-value md:text-md">{{ event.peaks.pmu }}</strong>
    </div>
  </section>
</template>

<script>
export default {
  props: ['event']
}
</script>
