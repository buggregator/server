<template>
    <div>
        <h1 class="event-inspector__title">
            <span>{{ event.process.name }}</span>
        </h1>

        <NuxtLink :to="event.route.show" class="event-inspector__link">
            <Cards :event="event" class="event-inspector__cards"/>
        </NuxtLink>

        <div class="event-inspector__footer">
            <span>{{ date }}</span>
            <Host :name="event.serverName" class="event-inspector__host"/>
        </div>
    </div>
</template>

<script>
import Host from "@/Components/UI/Host"
import Cards from "./Show/Cards"

export default {
    components: {Cards, Host},
    props: {
        event: Object,
    },
    computed: {
        date() {
            return this.event.date.fromNow()
        },
    }
}
</script>
<style lang="css">
.nav-item--sidebar .nav-item__list {
    width: 100%;
}

.nav-item--sidebar .nav-item__item {
    width: auto !important;
    min-width: 50%;
    max-width: 100%;
}
</style>
