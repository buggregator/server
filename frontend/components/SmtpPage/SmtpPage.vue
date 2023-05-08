<template>
  <div ref="main" class="smtp-page">
    <main class="smtp-page__main">
      <header class="smtp-page__header">
        <h2 class="smtp-page__header-title">{{ mail.subject }}</h2>
        <div class="smtp-page__header-meta">
          <span class="smtp-page__header-date">{{ date }}</span>
        </div>
      </header>

      <section class="smtp-page__sender">
        <template v-for="sender in senders">
          <div
            v-for="email in sender.address"
            :key="`${sender.title}-${email.email}`"
            class="smtp-page__sender-item"
            :class="`smtp-page__sender-${sender.title.toLowerCase()}`"
          >
            <div class="smtp-page__sender-title">{{ sender.title }}</div>
            <div class="smtp-page__sender-address">
              <template v-if="email.name">
                {{ email.name }} [{{ email.email }}]
              </template>
              <template v-else>
                {{ email.email }}
              </template>
            </div>
          </div>
        </template>
      </section>

      <section class="smtp-page__body">
        <Tabs :options="{ useUrlFragment: false }">
          <Tab name="Preview">
            <SmtpPagePreview device="tablet">
              <div v-html="htmlSource" />
            </SmtpPagePreview>
          </Tab>
          <Tab name="HTML">
            <CodeSnippet
              language="html"
              class="max-w-full"
              :code="event.payload.html"
            />
          </Tab>
          <Tab name="Raw">
            <CodeSnippet language="html" :code="event.payload.raw" />
          </Tab>
          <Tab name="Tech Info">
            <div>
              <h3 class="mb-3 font-bold">Email Headers</h3>
              <EventTable>
                <EventTableRow title="Id">
                  {{ event.payload.id }}
                </EventTableRow>
                <EventTableRow title="Subject">
                  {{ event.payload.subject }}
                </EventTableRow>
                <EventTableRow title="From">
                  <SmtpPageAddresses :addresses="event.payload.from" />
                </EventTableRow>
                <EventTableRow title="To">
                  <SmtpPageAddresses :addresses="event.payload.to" />
                </EventTableRow>
                <EventTableRow v-if="event.payload.cc.length" title="Cc">
                  <SmtpPageAddresses :addresses="event.payload.cc" />
                </EventTableRow>
                <EventTableRow v-if="event.payload.bcc.length" title="Bcc">
                  <SmtpPageAddresses :addresses="event.payload.bcc" />
                </EventTableRow>
                <EventTableRow
                  v-if="event.payload.reply_to.length"
                  title="Reply to"
                >
                  <SmtpPageAddresses :addresses="event.payload.reply_to" />
                </EventTableRow>
                <EventTableRow
                  v-if="event.payload.attachments.length"
                  title="Attachments"
                >
                  <SmtpPageAddresses :addresses="event.payload.attachments" />
                </EventTableRow>
              </EventTable>
            </div>
          </Tab>
        </Tabs>
      </section>
    </main>
  </div>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import moment from "moment";
import { Tabs, Tab } from "vue3-tabs-component";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";
import SmtpPagePreview from "~/components/SmtpPagePreview/SmtpPagePreview.vue";
import SmtpPageAddresses from "~/components/SmtpPageAddresses/SmtpPageAddresses.vue";
import EventTable from "~/components/EventTable/EventTable.vue";
import EventTableRow from "~/components/EventTableRow/EventTableRow.vue";

export default defineComponent({
  components: {
    EventTableRow,
    EventTable,
    SmtpPagePreview,
    SmtpPageAddresses,
    CodeSnippet,
    Tabs,
    Tab,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
    htmlSource: {
      type: String,
      required: true,
    },
  },
  data() {
    return {
      senders: [
        {
          title: "From",
          address: this.event.payload.from,
        },
        {
          title: "To",
          address: this.event.payload.to,
        },
        {
          title: "CC",
          address: this.event.payload.cc,
        },
        {
          title: "BCC",
          address: this.event.payload.bcc,
        },
        {
          title: "Reply to",
          address: this.event.payload.reply_to,
        },
      ],
    };
  },
  computed: {
    mail() {
      return this.event.payload;
    },
    date() {
      return moment(this.event.timestamp).format("DD.MM.YYYY HH:mm:ss");
    },
  },
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";
.smtp-page {
  @apply relative flex-1 flex flex-col;
}

.smtp-page__main {
  @apply flex-1 flex flex-col h-full flex-grow py-5 px-4 md:px-6 lg:px-8;
}

.smtp-page__header {
  @apply flex flex-col md:flex-row justify-between gap-y-2;
}

.smtp-page__header-meta {
  @apply flex flex-row items-center gap-x-5 mb-5;
}

.smtp-page__header-title {
  @apply text-lg lg:text-2xl;
}

.smtp-page__header-date {
  @include text-muted;
  @apply text-xs md:text-sm font-semibold;
}

.smtp-page__sender {
  @apply text-xs sm:text-sm font-semibold mt-3 flex flex-wrap items-center;
}

.smtp-page__sender-item {
  @apply flex border border-purple-300 rounded items-center mr-3 mb-2;
}

.smtp-page__sender-title {
  @apply px-2 md:px-3 py-1 border-r font-bold;
}

.smtp-page__sender-address {
  @apply px-2 md:px-3 bg-gray-800 py-1 text-white font-semibold rounded-r;
}

.smtp-page__sender-from .smtp-page__sender-address {
  @apply bg-blue-800;
}

.smtp-page__sender-to .smtp-page__sender-address {
  @apply bg-red-800;
}

.smtp-page__sender-cc .smtp-page__sender-address {
  @apply bg-purple-800;
}

.smtp-page__sender-reply .smtp-page__sender-address {
  @apply bg-green-800;
}

.smtp-page__body {
  @apply flex-1 flex flex-col;

  .tabs-component {
    @apply flex-1 flex flex-col;
  }

  .tabs-component-panel {
    @apply flex-1 flex flex-col;
  }
}
</style>
