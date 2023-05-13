<template>
  <div class="event-table-row">
    <div v-if="hasTitle" class="event-table-row__cell-name">
      <span v-html="title"></span>
    </div>
    <div class="event-table-row__cell-value">
      <slot></slot>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
  props: {
    title: {
      type: String,
      required: true,
    },
  },
  computed: {
    hasTitle(): boolean {
      return this.title !== null;
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";
.event-table-row {
  @include text-responsive;
  @apply md:flex bg-gray-50 dark:bg-gray-800 md:space-y-1 py-1 md:py-2 px-2 md:px-4 items-center sm:divide-x divide-gray-300 dark:divide-gray-700;
}

.event-table-row__cell-name {
  @apply font-bold mr-5 md:w-1/4 sm:pr-4;
}

.event-table-row__cell-value {
  @apply break-all md:w-3/4 sm:pl-4;

  .dump {
    @apply border-0;
  }
}
</style>
