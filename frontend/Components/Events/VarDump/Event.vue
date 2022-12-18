<template>
  <Event :event="event" class="event-vardump">
    <Dump
      :type="event.payloads.type"
      :value="event.payloads.value"
      class="event-vardump__dump"
    />
  </Event>
</template>

<script>
import Dump from "@/Components/UI/Dump"
import Event from "../Event"

export default {
  components: {Event, Dump},
  props: {
    event: Object,
  }
}
</script>
