<template>
  <div class="events-page">
    <PageHeader button-title="Clear events" @delete="clearEvents">
      <NuxtLink to="/" :disabled="!title">Home</NuxtLink>

      <template v-if="title">
        <span>&nbsp;/&nbsp;</span>
        <NuxtLink :disabled="true">{{ title }}</NuxtLink>
      </template>
    </PageHeader>

    <main v-if="events.length" class="events-page__events">
      <PreviewEventMapper
        v-for="event in events"
        :key="event.uuid"
        :event="event"
        class="events-page__event"
      />
    </main>

    <section v-if="!events.length" class="events-page__welcome">
      <PagePlaceholder class="events-page__tips" />
    </section>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import PagePlaceholder from "~/components/PagePlaceholder/PagePlaceholder.vue";
import PageHeader from "~/components/PageHeader/PageHeader.vue";
import { useNuxtApp } from "#app";
import PreviewEventMapper from "~/components/PreviewEventMapper/PreviewEventMapper.vue";

export default defineComponent({
  components: {
    PagePlaceholder,
    PreviewEventMapper,
    PageHeader,
  },
  setup() {
    if (process.client) {
      const { $events } = useNuxtApp();

      return {
        events: $events.items,
        clearEvents: $events.removeAll,
        title: "",
      };
    }
    return {
      events: [],
      title: "",
      clearEvents: () => {},
    };
  },
});
</script>

<style lang="scss">
@import "assets/mixins";

.events-page {
  @apply h-full w-full;
}

.events-page__events {
  @include border-style;
  @apply flex flex-col divide-y divide-y-2 divide-gray-200 dark:divide-gray-600;
}

.events-page__event {
  & + & {
    @apply border-b border-gray-200 dark:border-gray-700;
  }
}

.events-page__welcome {
  @apply flex-1 p-4 flex flex-col justify-center items-center bg-gray-50 dark:bg-gray-800 w-full h-full min-h-screen;
}
</style>
