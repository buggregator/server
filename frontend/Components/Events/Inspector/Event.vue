<template>
    <Event :event="event" class="event--inspector">
        <NavItem :event="event" class="event-inspector__nav"/>
    </Event>
</template>

<script>
import Event from "../Event"
import NavItem from "./NavItem"

export default {
    components: {
        Event, NavItem
    },
    props: {
        event: Object
    }
}
</script>
