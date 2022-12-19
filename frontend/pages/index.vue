<template>
  <div class="events-page">
    <header class="events-page__header">
      <div>
        <Fiilters/>
      </div>

      <div class="events-page__filters" v-if="hasEvents">
        <button @click="clearEvents" class="events__btn-clear">
          Clear screen
        </button>
      </div>
    </header>

    <main v-if="hasEvents" class="events-page__events">
      <component
        :is="eventComponent(event)"
        :event="event"
        v-for="event in events"
        :key="event.uuid"
        class="events-page__event"
      />
    </main>

    <section v-else class="events-page__welcome-block">
      <WsConnectionStatus/>
      <Tips class="events-page__tips"/>
    </section>
  </div>
</template>

<script>
import WsConnectionStatus from "./_partials/WsConnectionStatus"
import Fiilters from "./_partials/Fiilters"
import Tips from "./_partials/Tips"

import SentryEvent from "@/app/Event/Sentry"
import SmtpEvent from "@/app/Event/Smtp"
import VarDumpEvent from "@/app/Event/VarDump"
import MonologEvent from "@/app/Event/Monolog"
import InspectorEvent from "@/app/Event/Inspector"
import ProfilerEvent from "@/app/Event/Profiler"

// import RayComponent from "@/Components/Events/Event"
import SentryComponent from "@/Components/Events/Sentry/Event"
import SmtpComponent from "@/Components/Events/Smtp/Event"
import VarDumpComponent from "@/Components/Events/VarDump/Event"
import MonologComponent from "@/Components/Events/Monolog/Event"
import InspectorComponent from "@/Components/Events/Inspector/Event"
import ProfilerComponent from "@/Components/Events/Profiler/Event"

export default {
  components: {
    Fiilters, WsConnectionStatus, Tips,
    SentryComponent, SmtpComponent, VarDumpComponent,
    MonologComponent, InspectorComponent, ProfilerComponent
  },
  head() {
    let title = `Buggregator - Waiting for events`
    if (this.hasEvents) {
      title = `Events [${this.events.length}] | Buggregator`
    }

    return {title}
  },
  mounted() {
    this.$store.dispatch('events/fetch')
  },
  computed: {
    hasEvents() {
      return this.events.length > 0
    },
    events() {
      return this.$store.getters['events/filtered']
    },
  },
  methods: {
    clearEvents() {
      this.$store.dispatch('events/clear')
    },
    eventComponent(event) {
      if (event instanceof SentryEvent) {
        return 'SentryComponent'
      } else if (event instanceof SmtpEvent) {
        return 'SmtpComponent'
      } else if (event instanceof VarDumpEvent) {
        return 'VarDumpComponent'
      } else if (event instanceof MonologEvent) {
        return 'MonologComponent'
      } else if (event instanceof InspectorEvent) {
        return 'InspectorComponent'
      } else if (event instanceof ProfilerEvent) {
        return 'ProfilerComponent'
      } else if (event instanceof RayEvent) {
        return 'RayComponent'
      }
    },
  },
}
</script>
