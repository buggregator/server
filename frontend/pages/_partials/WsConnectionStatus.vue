<template>
  <div class="ws-connection">
    <div class="ws-connection__text">
      <span v-if="connected">Hurry! Give me something. I can't wait to show it.</span>
      <span v-else>Houston we have a problem! Websocket is not connected. Try to reload page.</span>
    </div>
  </div>
</template>

<script>
export default {
  computed: {
    connected() {
      return this.$store.getters['ws/connected']
    }
  }
}
</script>

<style lang="scss">
.ws-connection {
  @apply flex flex-col text-center md:flex-row gap-4 items-center justify-center;
  &__icon {
    @apply h-8 w-8;
  }

  &__text {
    @apply text-2xs sm:text-xs md:text-sm lg:text-base text-gray-600 dark:text-gray-300 font-semibold;
  }
}
</style>
