<template>
  <section v-if="hasBreadcrumbs" class="sentry-breadcrumbs">
    <h3 class="sentry-section__title">breadcrumbs</h3>
    <div class="sentry-breadcrumbs__wrapper">
      <nav
          style="grid-template-columns: 1fr 100px 200px 17px"
          class="sentry-breadcrumbs__nav">
        <div class="sentry-breadcrumbs__nav-col-title">description</div>
        <div class="sentry-breadcrumbs__nav-col-title">level</div>
        <div class="sentry-breadcrumbs__nav-col-title">time</div>
      </nav>

      <div class="sentry-breadcrumbs__cols-wrapper">
        <div v-for="b in event.breadcrumbs.values" style="grid-template-columns: 1fr 100px 200px" class="sentry-breadcrumbs__cols">
          <div class="sentry-breadcrumbs__col">
            <p class="sentry-breadcrumbs__col-message">{{ b.message }}</p>

            <CodeSnippet v-if="b.data" class="mt-3" language="json" :code="b.data"/>

            <div class="sentry-breadcrumbs__col-details">
              <div class="sentry-breadcrumbs__col-detail">
                <div class="sentry-breadcrumbs__col-detail-title">type</div>
                <div class="sentry-breadcrumbs__col-detail-value">{{ b.type }}</div>
              </div>
              <div class="sentry-breadcrumbs__col-detail">
                <div class="sentry-breadcrumbs__col-detail-title">category</div>
                <div class="sentry-breadcrumbs__col-detail-value">{{ b.category }}</div>
              </div>
            </div>
          </div>
          <div class="sentry-breadcrumbs__col">
            {{ b.level }}
          </div>
          <div class="sentry-breadcrumbs__col">{{ date(b.timestamp) }}</div>
        </div>
      </div>
    </div>
  </section>
</template>

<script lang="ts">
import {defineComponent, PropType} from "vue";
import {Sentry} from "~/config/types";
import moment from "moment";
import CodeSnippet from "~/components/CodeSnippet/CodeSnippet.vue";

export default defineComponent({
  components: {
    CodeSnippet
  },
  props: {
    event: {
      type: Object as PropType<Sentry>,
      required: true,
    },
  },
  computed: {
    hasBreadcrumbs() {
      return this.event.breadcrumbs?.values.length > 0
    }
  },
  methods: {
    date(timestamp: number): string {
      return moment.unix(timestamp).fromNow()
    }
  }
});
</script>

<style lang="scss" scoped>
@import "assets/mixins";

.sentry-breadcrumbs {
  @apply py-5 px-4;
}

.sentry-section__title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}

.sentry-breadcrumbs__title {
  @include text-muted;
  @apply font-bold uppercase text-sm mb-5;
}

.sentry-breadcrumbs__wrapper {
  @apply flex flex-col border border-purple-300 dark:border-purple-700 rounded;
  max-height: 600px;
}

.sentry-breadcrumbs__nav {
  @apply border-b border-purple-300 dark:border-purple-700 grid bg-purple-50 dark:bg-purple-800 text-xs font-bold text-purple-600 dark:text-purple-100 rounded-t;
}

.sentry-breadcrumbs__nav-col-title {
  @apply p-3 uppercase;
}

.sentry-breadcrumbs__cols-wrapper {
  @apply bg-gray-100 dark:bg-gray-800 max-h-full flex-1 overflow-y-scroll divide-y divide-purple-300 dark:divide-purple-600;
}

.sentry-breadcrumbs__cols {
  @apply grid text-xs;
}

.sentry-breadcrumbs__col {
  @apply p-3;
}

.sentry-breadcrumbs__col-message {
  @apply font-bold;
}

.sentry-breadcrumbs__col-details {
  @apply flex flex-row flex-wrap items-center text-purple-600 dark:text-purple-100 text-2xs my-3 gap-3;
}

.sentry-breadcrumbs__col-detail {
  @apply flex border border-purple-300 dark:border-purple-700 rounded items-center;
}

.sentry-breadcrumbs__col-detail-title {
  @apply px-2 border-r dark:border-purple-700;
}

.sentry-breadcrumbs__col-detail-value {
  @apply px-2 bg-purple-100 dark:bg-purple-800 rounded-r font-bold;
}
</style>