<template>
  <Event :event="event" class="event--inspector">
    <h1 class="event-inspector__title">
      <span>{{ event.process.name }}</span>
    </h1>

    <NuxtLink :to="event.route.show" class="event-inspector__link">
      <Cards :event="event" class="event-inspector__cards"/>
    </NuxtLink>
  </Event>
</template>

<script>
import Cards from "./_partials/Cards"
import Event from "../Event"

export default {
  components: {
    Event, Cards
  },
  props: {
    event: Object
  },
  computed: {
    date() {
      return this.event.date.fromNow()
    },
  }
}
</script>

<style lang="scss">
.event--inspector {
  &__title {
    @apply font-semibold text-black dark:text-gray-100 mb-2 flex justify-between items-center;
  }

  &__cards {
    @apply border dark:bg-gray-800;
  }

  &__link {
    @apply block;
  }
}
</style>
