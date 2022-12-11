<template>
  <Event :event="event" class="event--sentry">
    <Exception :event="event" class="event-sentry__exception"/>

    <div class="event-sentry__wrap">
      <div class="event-sentry__wrap-labels">
        <span>{{ date }}</span>
        <span><strong>logger: </strong>{{ event.logger }}</span>
        <span><strong>env: </strong>{{ event.environment }}</span>
      </div>
      <Host :name="event.serverName" class="event-sentry__host"/>
    </div>
  </Event>
</template>

<script>
import Host from "@/Components/UI/Host"
import Event from "../Event"
import Exception from "./Payloads/Exception"

export default {
  components: {Event, Exception, Host},
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
