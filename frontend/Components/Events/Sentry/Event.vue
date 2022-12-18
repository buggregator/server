<template>
  <Event :event="event" class="event--sentry">
    <Exception :event="event" class="event-sentry__exception"/>
  </Event>
</template>

<script>
import Host from "@/Components/UI/Host"
import Event from "../Event"
import Exception from "./_partials/Exception"

export default {
  components: {Event, Exception, Host},
  props: {
    event: Object
  }
}
</script>
