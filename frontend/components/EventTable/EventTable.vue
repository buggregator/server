<template>
  <div class="event-table">
    <slot></slot>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({});
</script>

<style lang="scss" scoped>
@import "assets/mixins";
.event-table {
  @include border-style;
  @apply bg-gray-100 dark:bg-gray-800 text-sm border divide-y;
}
</style>
