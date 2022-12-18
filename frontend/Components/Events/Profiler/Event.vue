<template>
  <Event :event="event" class="event--profiler">
    <NuxtLink :to="event.route.show" class="event--profiler__link">
      <Cards :cost="event.peaks"/>
    </NuxtLink>
  </Event>
</template>

<script>
import Host from "@/Components/UI/Host"
import Cards from "./_partials/Cards"
import Event from "../Event"

export default {
  components: {Event, Host, Cards},
  props: {
    event: Object,
  }
}
</script>

<style lang="scss">
.event--profiler {
  &__link {

  }
}
</style>
