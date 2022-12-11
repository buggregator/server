<template>
    <Event :event="event" class="event--vardump">
        <Dump
            :type="event.payloads.type"
            :value="event.payloads.value"
            class="event-var-dump__dump"
        />
        <Origin :origin="event.origin"/>
    </Event>
</template>

<script>
import Dump from "@/Components/UI/Dump";
import Event from "../Event";
import Origin from "./Origin";

export default {
    components: {Origin, Event, Dump},
    props: {
        event: Object,
    }
}
</script>
