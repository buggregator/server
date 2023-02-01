<template>
  <event-card class="event-var-dump" :event="event">
    <CodeSnippet
      v-if="isBooleanValue"
      language="php"
      :code="eventValue === 1"
    />

    <div
      v-if="!isBooleanValue"
      class="event-var-dump__html"
      v-html="varDumpBody"
    />
  </event-card>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import EventCard from "~/components/EventCard/EventCard.vue";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";
import Sfdump from "~/vendor/dumper";

export default defineComponent({
  components: {
    EventCard,
    CodeSnippet,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  computed: {
    eventValue(): unknown {
      return this.event.payload.payload.value;
    },
    isBooleanValue(): boolean {
      return (
        this.event.payload.payload.type === "boolean" ||
        this.eventValue === true ||
        this.eventValue === false
      );
    },
    varDumpId(): string | null {
      return (
        (this.eventValue as string)?.match(/(sf-dump-[0-9]+)/i)?.[0] || null
      );
    },
    varDumpBody(): string | unknown {
      if (this.varDumpId) {
        return (this.eventValue as string).replace(
          /<(style|script)\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/(style|script)>/gi,
          ""
        );
      }

      return this.eventValue;
    },
  },
  mounted() {
    if (this.varDumpId) {
      Sfdump(this.varDumpId);
    }
  },
});
</script>

<style lang="scss" scoped>
.event-var-dump {
  display: block;
}

.event-var-dump__html {
  @apply border-gray-300 dark:border-gray-500 divide-gray-300 dark:divide-gray-500 font-mono py-2 px-2 md:px-3 lg:px-4 border bg-gray-200 dark:bg-gray-800 text-blue-700 dark:text-white text-sm break-all text-2xs sm:text-xs md:text-sm lg:text-base;
}
</style>
