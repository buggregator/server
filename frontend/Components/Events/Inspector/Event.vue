<template>
  <Event :event="event" class="event--inspector">
    <h3 class="event-0inspector__title">{{ event.process.name }} </h3>

    <NuxtLink :to="event.route.show" class="event-0inspector__link">
      <Cards :event="event" class="event--inspector__cards"/>
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
    @apply mb-3 font-semibold;
  }

  &__cards {
    @apply dark:bg-gray-800 mt-5;
  }

  &__link {
    @apply block;
  }
}
</style>
