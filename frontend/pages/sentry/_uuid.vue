<template>
  <div>
    <main class="flex flex-col flex-grow">
      <header class="bg-gray-50 dark:bg-gray-900 py-5 px-4 md:px-6 lg:px-8 border-b">
        <div class="flex justify-between items-center">
          <h1 class="font-bold text-sm sm:text-base md:text-lg lg:text-2xl  break-all sm:break-normal">
            {{ event.payload.type }}
          </h1>
          <JsonChip :href="event.route.json" class="mr-auto ml-1.5 mb-2"/>
          <button class="h-5 w-5" @click="deleteEvent">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
              <path
                d="m338 197-19 221c-1 10 14 11 15 1l19-221a8 8 0 0 0-15-1zM166 190c-4 0-7 4-7 8l19 221c1 10 16 9 15-1l-19-221c0-4-4-7-8-7zM249 197v222a7 7 0 1 0 15 0V197a7 7 0 1 0-15 0z"/>
              <path
                d="M445 58H327V32c0-18-14-32-31-32h-80c-17 0-31 14-31 32v26H67a35 35 0 0 0 0 69h8l28 333c2 29 27 52 57 52h192c30 0 55-23 57-52l4-46a8 8 0 0 0-15-2l-4 46c-2 22-20 39-42 39H160c-22 0-40-17-42-39L90 127h22a7 7 0 1 0 0-15H67a20 20 0 0 1 0-39h378a20 20 0 0 1 0 39H147a7 7 0 1 0 0 15h275l-21 250a8 8 0 0 0 15 2l21-252h8a35 35 0 0 0 0-69zm-133 0H200V32c0-10 7-17 16-17h80c9 0 16 7 16 17v26z"/>
            </svg>
          </button>
        </div>
        <pre class="text-muted text-sm" v-html="event.payload.value"/>
        <p class="text-muted text-sm mt-3">{{ date }}</p>
      </header>

      <Tags :event="event"/>
      <Exceptions :exceptions="event.exceptions"/>
      <Breadcrumbs :event="event"/>
      <User :event="event"/>
      <Request :event="event"/>
      <App :event="event"/>
      <Device :event="event"/>
      <OS :event="event"/>
    </main>
  </div>
</template>

<script>
import Tags from "@/Components/Events/Sentry/Show/Tags"
import Breadcrumbs from "@/Components/Events/Sentry/Show/Breadcrumbs"
import User from "@/Components/Events/Sentry/Show/User"
import Request from "@/Components/Events/Sentry/Show/Request"
import App from "@/Components/Events/Sentry/Show/App"
import Device from "@/Components/Events/Sentry/Show/Device"
import OS from "@/Components/Events/Sentry/Show/OS"
import Exceptions from "@/Components/Events/Sentry/Show/Exceptions"
import JsonChip from "@/Components/UI/JsonChip"
import SentryEvent from "../../app/Event/Sentry"

export default {
  components: {
    JsonChip,
    Tags, Breadcrumbs, Request,
    App, Device, OS,
    User, Exceptions
  },
  head() {
    return {
      title: `Sentry > ${this.event.uuid} | Buggregator`
    }
  },
  async asyncData({params, redirect, $api}) {
    const json = await $api.events.show(params.uuid)
    const event = new SentryEvent(json.payload, json.uuid, json.timestamp)
    if (!event) {
      redirect('/sentry')
    }

    return {event}
  },
  methods: {
    async deleteEvent() {
      await this.$store.dispatch('events/delete', this.event)
      this.$router.push('/sentry')
    }
  },
  computed: {
    date() {
      return this.event.date.fromNow()
    },
    stacktrace() {
      return this.event.stacktrace
    }
  }
}
</script>
