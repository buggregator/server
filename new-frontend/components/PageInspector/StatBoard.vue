<template>
  <section class="inspector-statboard">
    <div class="item">
      <h4 class="item--name">Timestamp</h4>
      <strong class="item--value">{{ processDate }}</strong>
    </div>
    <div class="item">
      <h4 class="item--name">Duration</h4>
      <strong class="item--value">{{ transaction.duration }} ms</strong>
    </div>
    <div class="item">
      <h4 class="item--name">Result</h4>
      <span class="item--label">{{ processResult }}</span>
    </div>
  </section>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {InspectorTransaction} from "~/config/types";
import moment from "moment";

export default defineComponent({
  props: {
    transaction: {
      type: Object as PropType<InspectorTransaction>,
      required: true,
    },
  },
  computed: {
    processDate(): string {
      return moment(this.transaction.timestamp).toLocaleString();
    },
    processResult(): string {
      return (this.transaction.result || 'success').toUpperCase();
    }
  }
});
</script>

<style lang="scss" scoped>
.inspector-statboard {
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