<template>
  <div class="flex flex-col h-full flex-grow py-5 px-4 md:px-6 lg:px-8">
    <div class="flex flex-col flex-reverse md:flex-row justify-between items-center">
      <h2 class="text-sm sm:text-base md:text-lg lg:text-2xl">{{ event.subject }}</h2>

      <div class="flex items-center align-center space-x-3">
        <span class="text-sm font-semibold text-muted">{{ date }}</span>

        <JsonChip :href="event.route.json" />
        <button class="h-5 w-5" @click="deleteEvent">
          <svg class="fill-current" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
            <path
              d="m338 197-19 221c-1 10 14 11 15 1l19-221a8 8 0 0 0-15-1zM166 190c-4 0-7 4-7 8l19 221c1 10 16 9 15-1l-19-221c0-4-4-7-8-7zM249 197v222a7 7 0 1 0 15 0V197a7 7 0 1 0-15 0z"/>
            <path
              d="M445 58H327V32c0-18-14-32-31-32h-80c-17 0-31 14-31 32v26H67a35 35 0 0 0 0 69h8l28 333c2 29 27 52 57 52h192c30 0 55-23 57-52l4-46a8 8 0 0 0-15-2l-4 46c-2 22-20 39-42 39H160c-22 0-40-17-42-39L90 127h22a7 7 0 1 0 0-15H67a20 20 0 0 1 0-39h378a20 20 0 0 1 0 39H147a7 7 0 1 0 0 15h275l-21 250a8 8 0 0 0 15 2l21-252h8a35 35 0 0 0 0-69zm-133 0H200V32c0-10 7-17 16-17h80c9 0 16 7 16 17v26z"/>
          </svg>
        </button>
      </div>
    </div>

    <div class="text-xs font-semibold mt-3 flex flex-wrap items-center">
      <div class="flex border border-purple-300 rounded items-center mr-3 mb-2">
        <div class="px-3 py-1 border-r">From</div>
        <div class="px-3 py-1 bg-gray-800 text-white font-semibold rounded-r" v-for="email in event.event.from">
          {{ email.name }} [{{ email.email }}]
        </div>
      </div>

      <div class="flex border border-purple-300 rounded items-center mr-3 mb-2" v-for="email in event.event.to">
        <div class="px-3 py-1 border-r">To</div>
        <div class="px-3 py-1 bg-blue-800 text-white font-semibold rounded-r">{{ email.name }} [{{ email.email }}]</div>
      </div>

      <div class="flex border border-purple-300 rounded items-center mr-3 mb-2" v-for="email in event.event.cc">
        <div class="px-3 py-1 border-r">CC</div>
        <div class="px-3 py-1 bg-red-800 text-white font-semibold rounded-r">{{ email.name }} [{{ email.email }}]</div>
      </div>

      <div class="flex border border-purple-300 rounded items-center mr-3 mb-2" v-for="email in event.event.bcc">
        <div class="px-3 py-1 border-r">BCC</div>
        <div class="px-3 py-1 bg-purple-800 text-white font-semibold rounded-r">{{ email.name }} [{{ email.email }}]
        </div>
      </div>

      <div class="flex border border-purple-300 rounded items-center mr-3 mb-2" v-for="email in event.event.reply_to">
        <div class="px-3 py-1 border-r">Reply to</div>
        <div class="px-3 py-1 bg-green-800 text-white font-semibold rounded-r">{{ email.name }} [{{
            email.email
          }}]
        </div>
      </div>
    </div>

    <Tabs class="flex-1">
      <Tab title="Preview" class="h-full">
        <HtmlPreview>
          <iframe :src="htmlLink"/>
        </HtmlPreview>
      </Tab>
      <Tab title="HTML">
        <CodeSnippet language="html" class="max-w-full" :code="event.event.html"/>
      </Tab>
      <Tab title="Raw">
        <CodeSnippet language="html" :code="event.event.raw"/>
      </Tab>
      <Tab title="Tech Info">
        <div>
          <h3 class="mb-3 font-bold">Email Headers</h3>
          <Table>
            <TableRow title="Id">
              {{ event.event.id }}
            </TableRow>
            <TableRow title="Subject">
              {{ event.subject }}
            </TableRow>
            <TableRow title="From">
              <Addresses :addresses="event.event.from"/>
            </TableRow>
            <TableRow title="To">
              <Addresses :addresses="event.event.to"/>
            </TableRow>
            <TableRow v-if="event.event.cc.length" title="Cc">
              <Addresses :addresses="event.event.cc"/>
            </TableRow>
            <TableRow v-if="event.event.bcc.length" title="Bcc">
              <Addresses :addresses="event.event.bcc"/>
            </TableRow>
            <TableRow v-if="event.event.reply_to.length" title="Reply to">
              <Addresses :addresses="event.event.reply_to"/>
            </TableRow>
            <TableRow v-if="event.event.attachments.length" title="Attachments">
              <div class="flex flex-col space-y-2">
                <div v-for="(attachment, i) in event.event.attachments">
                  <span>{{ i + 1 }}.</span> {{ attachment.name }}
                </div>
              </div>
            </TableRow>
            <TableRow title="Content-Type">
              {{ event.event.content_type }}
            </TableRow>
          </Table>
        </div>
      </Tab>
    </Tabs>
  </div>
</template>

<script>
import SmtpEvent from "@/app/Event/Smtp"
import CodeSnippet from "@/Components/UI/CodeSnippet"
import Table from "@/Components/UI/Table"
import TableRow from "@/Components/UI/TableRow"
import Dump from "@/Components/UI/Dump"
import Collapsed from "@/Components/UI/Collapsed"
import HtmlPreview from "@/Components/UI/HtmlPreview"
import Addresses from "./_partials/Addresses"
import JsonChip from "@/Components/UI/JsonChip"
import Tab from "@/Components/UI/Tab"
import Tabs from "@/Components/UI/Tabs"

export default {
  layout: 'smtp',
  components: {
    CodeSnippet, Table, TableRow, Dump, Collapsed,
    HtmlPreview, Addresses, JsonChip, Tab, Tabs
  },
  async asyncData({params, redirect, $api}) {
    const json = await $api.events.show(params.uuid)
    const event = new SmtpEvent(json.payload, json.uuid, json.timestamp)
    if (!event) {
      redirect('/smtp')
    }

    return {event}
  },
  methods: {
    clearEvents() {
      this.$store.dispatch('events/clear', 'smtp')
    },
    async deleteEvent() {
      await this.$store.dispatch('events/delete', this.event)
      this.$router.push('/smtp')
    }
  },
  computed: {
    htmlLink() {
      return `/api/smtp/${this.event.uuid}/html`
    },
    date() {
      return this.event.date.format('DD.MM.YYYY HH:mm:ss')
    },
    events() {
      return this.$store.getters['events/filteredByType']('smtp')
    },
    hasEvents() {
      return this.events.length > 0
    }
  },
}
</script>
