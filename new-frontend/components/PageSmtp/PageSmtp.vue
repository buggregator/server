<template>
  <div ref="main" class="page-smtp">
    <main class="page-smtp__main">
      <header class="page-smtp__header">
        <h2 class="page-smtp__header-title">{{ mail.subject }}</h2>
        <div class="page-smtp__header-meta">
          <button class="page-smtp__delete-button" @click="$emit('delete', event)">
            <IconSvg name="trash-bin"/>
          </button>

          <span class="page-smtp__header-date">{{ date }}</span>
        </div>
      </header>

      <section class="page-smtp__sender">
        <template v-for="sender in senders">
          <div
              v-for="email in sender.address"
              :key="`${sender.title}-${email.email}`"
              class="page-smtp__sender-item"
              :class="`page-smtp__sender-${sender.title.toLowerCase()}`"
          >
            <div class="page-smtp__sender-title">{{ sender.title }}</div>
            <div class="page-smtp__sender-address">
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

      <section class="page-smtp__body">
        <Tabs :options="{ useUrlFragment: false }">
          <Tab name="Preview">
            <HtmlPreview device="tablet">
              <div v-html="HTML"/>
            </HtmlPreview>
          </Tab>
          <Tab name="HTML">
            <CodeSnippet language="html" class="max-w-full" :code="event.payload.html"/>
          </Tab>
          <Tab name="Raw">
            <CodeSnippet language="html" :code="event.payload.raw"/>
          </Tab>
          <Tab name="Tech Info">
            <div>
              <h3 class="mb-3 font-bold">Email Headers</h3>
              <TableC>
                <TableRow title="Id">
                  {{ event.payload.id }}
                </TableRow>
                <TableRow title="Subject">
                  {{ event.payload.subject }}
                </TableRow>
                <TableRow title="From">
                  <EmailAddresses :addresses="event.payload.from"/>
                </TableRow>
                <TableRow title="To">
                  <EmailAddresses :addresses="event.payload.to"/>
                </TableRow>
                <TableRow v-if="event.payload.cc.length" title="Cc">
                  <EmailAddresses :addresses="event.payload.cc"/>
                </TableRow>
                <TableRow v-if="event.payload.bcc.length" title="Bcc">
                  <EmailAddresses :addresses="event.payload.bcc"/>
                </TableRow>
                <TableRow v-if="event.payload.reply_to.length" title="Reply to">
                  <EmailAddresses :addresses="event.payload.reply_to"/>
                </TableRow>
                <TableRow v-if="event.payload.attachments.length" title="Attachments">
                  <EmailAttachments :attachments="event.payload.attachments"/>
                </TableRow>
              </TableC>
            </div>
          </Tab>
        </Tabs>
      </section>
    </main>
  </div>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import IconSvg from "~/components/IconSvg/IconSvg.vue";
import {NormalizedEvent} from "~/config/types";
import moment from "moment";
import {Tabs, Tab} from "vue3-tabs-component";
import TableC from "~/components/Table/Table.vue";
import TableRow from "~/components/Table/TableRow.vue";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";
import HtmlPreview from "./HtmlPreview.vue";
import EmailAddresses from "./EmailAddresses.vue";

export default defineComponent({
  components: {
    HtmlPreview,
    EmailAddresses,
    CodeSnippet,
    IconSvg,
    Tabs,
    Tab,
    TableC,
    TableRow
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
    HTML: {
      type: String,
      required: true,
    }
  },
  emits: ['delete'],
  data() {
    return {
      senders: [
        {
          title: 'From',
          address: this.event.payload.from
        },
        {
          title: 'To',
          address: this.event.payload.to
        },
        {
          title: 'CC',
          address: this.event.payload.cc
        },
        {
          title: 'BCC',
          address: this.event.payload.bcc
        },
        {
          title: 'Reply to',
          address: this.event.payload.reply_to
        }
      ]
    };
  },
  computed: {
    mail() {
      return this.event.payload;
    },
    date() {
      return moment(this.event.timestamp).format('DD.MM.YYYY HH:mm:ss')
    },
  }
});
</script>


<style lang="scss" scoped>
@import "assets/mixins";

.page-smtp {
  @apply relative;
}

.page-smtp__main {
  @apply flex flex-col h-full flex-grow py-5 px-4 md:px-6 lg:px-8;
}

.page-smtp__header {
  @apply flex flex-col md:flex-row justify-between items-center;
}

.page-smtp__header-meta {
  @apply flex flex-col md:flex-row items-center gap-x-5;
}

.page-smtp__header-title {
  @apply text-sm sm:text-base md:text-lg lg:text-2xl;
}

.page-smtp__header-date {
  @include text-muted;
  @apply text-sm font-semibold;
}

.page-smtp__sender {
  @apply text-xs font-semibold mt-3 flex flex-wrap items-center;
}

.page-smtp__sender-item {
  @apply flex border border-purple-300 rounded items-center mr-3 mb-2;
}

.page-smtp__sender-title {
  @apply px-3 py-1 border-r;
}

.page-smtp__sender-address {
  @apply px-3 bg-gray-800 py-1 text-white font-semibold rounded-r;
}

.page-smtp__sender-from .page-smtp__sender-address {
  @apply bg-blue-800;
}

.page-smtp__sender-to .page-smtp__sender-address {
  @apply bg-red-800;
}

.page-smtp__sender-cc .page-smtp__sender-address {
  @apply bg-purple-800;
}

.page-smtp__sender-reply .page-smtp__sender-address {
  @apply bg-green-800;
}

.page-smtp__body {

}

.page-smtp__delete-button {
  @apply h-5 w-5;
}

.page-smtp__delete-button svg {
  @apply fill-current;
}
</style>