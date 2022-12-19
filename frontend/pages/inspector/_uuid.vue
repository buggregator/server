<template>
  <div>
    <main class="flex flex-col flex-grow">
      <header class="bg-gray-50 dark:bg-gray-700 py-5 px-4 md:px-6 lg:px-8 border-b">

        <div class="flex flex-col md:flex-row justify-between items-center">
          <h1 class="text-sm sm:text-base md:text-lg lg:text-2xl font-bold break-all sm:break-normal">
            {{ event.process.name }}
          </h1>
          <JsonChip :href="event.route.json" class="mb-2 ml-1.5 mr-auto"/>

          <div class="mt-5 sm:ml-5 sm:mt-0 flex justify-between sm:flex-none">
            <button class="fill-current text-blue-500 h-5 w-5" @click="deleteEvent">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
                <path
                  d="m338 197-19 221c-1 10 14 11 15 1l19-221a8 8 0 0 0-15-1zM166 190c-4 0-7 4-7 8l19 221c1 10 16 9 15-1l-19-221c0-4-4-7-8-7zM249 197v222a7 7 0 1 0 15 0V197a7 7 0 1 0-15 0z"/>
                <path
                  d="M445 58H327V32c0-18-14-32-31-32h-80c-17 0-31 14-31 32v26H67a35 35 0 0 0 0 69h8l28 333c2 29 27 52 57 52h192c30 0 55-23 57-52l4-46a8 8 0 0 0-15-2l-4 46c-2 22-20 39-42 39H160c-22 0-40-17-42-39L90 127h22a7 7 0 1 0 0-15H67a20 20 0 0 1 0-39h378a20 20 0 0 1 0 39H147a7 7 0 1 0 0 15h275l-21 250a8 8 0 0 0 15 2l21-252h8a35 35 0 0 0 0-69zm-133 0H200V32c0-10 7-17 16-17h80c9 0 16 7 16 17v26z"/>
              </svg>
            </button>
          </div>
        </div>
      </header>

      <Cards :event="event" class="px-4 md:px-6 lg:px-8"/>
      <TimelineChart :event="event" v-if="event && event.event.length > 0"/>
      <Url :event="event"/>
      <Request :event="event"/>
    </main>
  </div>
</template>

<script>
import JsonChip from "@/Components/UI/JsonChip"
import InspectorEvent from "@/app/Event/Inspector"
import TimelineChart from "@/Components/Events/Inspector/_partials/Timeline"
import Cards from "@/Components/Events/Inspector/_partials/Cards"
import Request from "@/Components/Events/Inspector/_partials/Request"
import Url from "@/Components/Events/Inspector/_partials/Url"

export default {
  components: {
    JsonChip, Request, Url, TimelineChart, Cards
  },
  head() {
    return {
      title: `Inspector > ${this.event.uuid} | Buggregator`
    }
  },
  async asyncData({params, redirect, $api}) {
    const json = await $api.events.show(params.uuid)
    const event = new InspectorEvent(json.payload, json.uuid, json.timestamp)
    if (!event) {
      redirect('/inspector')
    }

    return {event}
  },
  methods: {
    async deleteEvent() {
      await this.$store.dispatch('events/delete', this.event)
      this.$router.push('/inspector')
    }
  }
}
</script>
