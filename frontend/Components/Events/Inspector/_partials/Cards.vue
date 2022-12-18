<template>
  <section class="inspector-cards">
    <div class="item">
      <h4 class="item--name">Timestamp</h4>
      <strong class="item--value">{{ event.processDate }}</strong>
    </div>
    <div class="item">
      <h4 class="item--name">Duration</h4>
      <strong class="item--value">{{ event.process.duration }} ms</strong>
    </div>
    <div class="item">
      <h4 class="item--name">Result</h4>
      <span :class="`bg-${event.color}-700`" class="item--label">{{ event.processResult }}</span>
    </div>
  </section>
</template>

<script>
export default {
  props: ['event']
}
</script>

<style lang="scss">
.inspector-cards {
  @apply pt-5 pb-4 px-4 md:px-5 border-b flex flex-col sm:flex-row justify-between items-start divide-y sm:divide-y-0 sm:divide-x;

  > .item {
    @apply w-full sm:w-1/4 pb-5 sm:pb-0 sm:px-10 pt-5 sm:pt-0;

    &:first-child {
      @apply sm:px-0;
    }

    &--name {
      @apply text-gray-600 dark:text-gray-300 font-bold text-2xs mb-1 uppercase;
    }

    &--value {
      @apply text-xs sm:text-sm md:text-base;
    }

    &-label {
      @apply px-3 py-1 text-white rounded text-2xs font-bold;
    }
  }
}
</style>
