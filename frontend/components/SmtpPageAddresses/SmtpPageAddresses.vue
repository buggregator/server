<template>
  <div class="smtp-page-addresses">
    <div
      v-for="user in addresses"
      :key="user.email"
      class="smtp-page-addresses__item"
    >
      <span v-if="user.name" class="smtp-page-addresses__item-name">{{
        user.name
      }}</span>
      <span>{{ user.email }}</span>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";

export default defineComponent({
  props: {
    addresses: {
      type: Array,
      required: true,
    },
  },
});
</script>

<style lang="scss" scoped>
.smtp-page-addresses {
  @apply flex flex-col divide-y text-sm;
}

.smtp-page-addresses__item {
  @apply flex items-center space-x-2 py-2;
}

.smtp-page-addresses__item-name {
  @apply text-white bg-gray-800 px-2;
}
</style>
