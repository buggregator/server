<template>
  <div class="events-page">
    <page-header button-title="Clear events" @delete="clearEvents">
      <nuxt-link to="/" :disabled="!title">Home</nuxt-link>

      <template v-if="title">
        <span>&nbsp;/&nbsp;</span>
        <nuxt-link :disabled="true">{{ title }}</nuxt-link>
      </template>
    </page-header>

    <main v-if="events.length" class="events-page__events">
      <event-mapper
        v-for="event in events"
        :key="event.uuid"
        :event="event"
        class="events-page__event"
      />
    </main>

    <section v-if="!events.length" class="events-page__welcome">
      <page-tips class="events-page__tips" />
    </section>
  </div>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import EventMapper from "~/components/EventMapper/EventMapper.vue";
import PageTips from "~/pages/PageTips/PageTips.vue";
import PageHeader from "~/components/PageHeader/PageHeader.vue";
import { useNuxtApp } from "#app";

export default defineComponent({
  components: {
    PageTips,
    EventMapper,
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
  @apply flex flex-col divide-y;
}

.events-page__event {
  & + & {
    @apply border-b;
  }
}

.events-page__welcome {
  @apply flex-1 p-4 flex flex-col justify-center items-center bg-gray-50 dark:bg-gray-800 w-full h-full min-h-screen;
}
</style>
