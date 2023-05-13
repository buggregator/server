<template>
  <div class="dump-preview">
    <CodeSnippet v-if="isBooleanValue" language="php" :code="value === 1" />
    <div
      v-if="!isBooleanValue"
      class="var-dump-preview__html"
      v-html="dumpBody"
    />
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";
import { useNuxtApp } from "#app";

export default defineComponent({
  components: {
    CodeSnippet,
  },
  props: ["value", "type"],
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
    isBooleanValue(): boolean {
      return typeof this.value === "boolean" || this.type === "boolean";
    },
    dumpId(): string | null {
      if (!this.value) {
        return null;
      }
      return this.value.toString().match(/(sf-dump-[0-9]+)/i)?.[0] || null;
    },
    dumpBody(): string | unknown {
      if (this.dumpId) {
        return (this.value as string).replace(
          /<(style|script)\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/(style|script)>/gi,
          ""
        );
      }

      return this.value;
    },
  },
  mounted() {
    if (this.dumpId && this.Sfdump && typeof this.Sfdump === "function") {
      this.Sfdump(this.dumpId);
    }
  },
});
</script>

<style lang="scss" scoped>
.dump-preview {
  display: block;
}

.var-dump-preview__html {
  @apply border-gray-300 dark:border-gray-600 divide-gray-300 dark:divide-gray-600 font-mono py-2 px-2 md:px-3 lg:px-4 border bg-gray-900 text-white text-sm break-all text-2xs sm:text-xs md:text-sm lg:text-base;
}
</style>
