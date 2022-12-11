<template>
  <div class="sentry-events__item">
    <NuxtLink tag="div" :to="event.route.show" class="sentry-events__item-link">
      <Exception :event="event" class="event-sentry__exception"/>

      <div class="event-sentry__wrap">
        <div class="event-sentry__wrap-labels">
          <span>{{ date }}</span>
          <span><strong>logger: </strong>{{ event.logger }}</span>
          <span><strong>env: </strong>{{ event.environment }}</span>
        </div>
        <Host :name="event.serverName"/>
      </div>
    </NuxtLink>
  </div>
</template>

<script>
import Host from "@/Components/UI/Host"
import JsonChip from "@/Components/UI/JsonChip"
import Exception from "../Payloads/Exception"

export default {
  components: {
    JsonChip, Host, Exception
  },
  props: {
    event: Object
  },
  computed: {
    date() {
      return this.event.date.fromNow()
    },
    location() {
      return this.event.location
    }
  }
}
</script>
