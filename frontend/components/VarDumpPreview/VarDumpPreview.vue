<template>
  <PreviewCard class="var-dump-preview" :event="event">
    <CodeSnippet
      v-if="isBooleanValue"
      language="php"
      :code="eventValue === 1"
    />

    <div
      v-if="!isBooleanValue"
      class="var-dump-preview__html"
      v-html="varDumpBody"
    />
  </PreviewCard>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent, VarDump } from "~/config/types";
import PreviewCard from "~/components/PreviewCard/PreviewCard.vue";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";
import { useNuxtApp } from "#app";

export default defineComponent({
  components: {
    PreviewCard,
    CodeSnippet,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  setup() {
    if (process.client) {
      const { $vendors } = useNuxtApp();

      return {
        Sfdump: $vendors.sfdump,
      };
    }

    // NOTE: storybook support
    return {
      Sfdump: window?.Sfdump as unknown,
    };
  },
  computed: {
    eventValue(): unknown {
      return (this.event.payload as VarDump).payload.value;
    },
    isBooleanValue(): boolean {
      return (
        (this.event.payload as VarDump).payload.type === "boolean" ||
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
    if (this.varDumpId && this.Sfdump && typeof this.Sfdump === "function") {
      this.Sfdump(this.varDumpId);
    }
  },
});
</script>

<style lang="scss" scoped>
.var-dump-preview {
  display: block;
}

.var-dump-preview__html {
  @apply border-gray-300 dark:border-gray-500 divide-gray-300 dark:divide-gray-500 font-mono py-2 px-2 md:px-3 lg:px-4 border bg-gray-800 text-blue-700 dark:text-white text-sm break-all text-2xs sm:text-xs md:text-sm lg:text-base;
}
</style>
