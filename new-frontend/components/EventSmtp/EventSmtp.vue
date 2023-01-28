<template>
  <event-card class="event-smtp" :event="event">
    <NuxtLink :to="eventLink" class="event-smtp__link">
      <h3 class="event-smtp__link-title">
        {{ event.payload.subject }}
      </h3>

      <div class="event-smtp__link-text">
        <span> <strong>To:</strong> {{ event.payload.to[0].email }} </span>

        <span>{{ fromNowDate }}</span>
      </div>
    </NuxtLink>
  </event-card>
</template>

<script lang="ts">
import { defineComponent, PropType } from "vue";
import { NormalizedEvent } from "~/config/types";
import EventCard from "~/components/EventCard/EventCard.vue";
import moment from "moment";

export default defineComponent({
  components: {
    EventCard,
  },
  props: {
    event: {
      type: Object as PropType<NormalizedEvent>,
      required: true,
    },
  },
  computed: {
    fromNowDate() {
      return moment(this.event.date).fromNow();
    },
    eventLink() {
      return `/smtp/${this.event.id}}`;
    },
  },
});
</script>

<style lang="scss" scoped>
@mixin text-muted {
  @apply text-gray-600 dark:text-gray-300;
}

.event-smtp {
}

.event-smtp__link {
  @apply block dark:bg-gray-800 text-sm hover:bg-white dark:hover:bg-gray-900 flex items-stretch dark:border-gray-600 flex flex-col p-5;
}

.event-smtp__nav-item {
  @apply border;
}

.event-smtp__link-title {
  @apply font-semibold mb-2;
}

.event-smtp__link-text {
  @include text-muted;
  @apply flex justify-between text-xs;
}
</style>
