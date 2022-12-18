<template>
    <Event :event="event" class="event--smtp">
        <NavItem :event="event" class="event-smtp__nav-item"/>
    </Event>
</template>

<script>
import Event from "../Event";
import NavItem from "@/pages/smtp/_partials/NavItem"

export default {
    components: {
        Event, NavItem
    },
    props: {
        event: Object
    }
}
</script>
