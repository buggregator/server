<template>
  <Event :event="event" class="event-profiler">
    <NavItem :event="event" />
    <Origin :origin="event.origin"/>
  </Event>
</template>

<script>
import Event from "../Event"
import Origin from "./Origin"
import NavItem from "./NavItem"

export default {
  components: {Event, NavItem, Origin},
  props: {
    event: Object,
  }
}
</script>
