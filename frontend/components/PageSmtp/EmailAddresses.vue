<template>
  <div class="email-address__list">
    <div v-for="user in addresses" class="email-address__list-item">
      <span v-if="user.name" class="email-address__username">{{ user.name }}</span>
      <span class="email-address__email">{{ user.email }}</span>
    </div>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";

export default defineComponent({
  props: {
    addresses: {
      type: Array,
      required: true
    }
  },
})
</script>

<style lang="scss" scoped>
.email-address__list {
  @apply flex flex-col divide-y text-sm;
}

.email-address__list-item {
  @apply flex items-center space-x-2 py-2;
}

.email-address__username {
  @apply text-white bg-gray-800 px-2;
}

.email-address__email {

}
</style>
