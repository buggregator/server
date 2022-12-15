<template>
  <div>
    <NuxtLink :to="event.route.show" class="event-profiler__link">
      <Cards :event="event" class="event-profiler__cards"/>
    </NuxtLink>
  </div>
</template>

<script>
import Host from "@/Components/UI/Host"
import Cards from "./Show/Cards"

export default {
    components: {Host, Cards},
    props: {
        event: Object,
    }
}
</script>
