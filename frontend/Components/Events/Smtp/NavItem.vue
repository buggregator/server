<template>
  <NuxtLink :to="event.route.show"
            class="event-smtp__link"
            :class="{'active': isActive }"
  >
    <div class="event-sentry__left" :class="{'active': isActive }"></div>
    <div class="event-smtp__link-body">
      <h3 class="event-smtp__link-title" :class="{ 'font-bold': isActive }">{{ event.event.subject }}</h3>
      <div class="event-smtp__link-text">
        <span>
            <strong>To:</strong> {{ event.event.to[0].email }}
        </span>
        <span>{{ date }}</span>
      </div>
    </div>

  </NuxtLink>
</template>

<script>
export default {
  props: {
    event: Object
  },
  computed: {
    url() {
      return this.event.route.show
    },
    date() {
      return this.event.date.fromNow()
    },
    isActive() {
      if (!this.$route.params.uuid) {
        return false
      }

      return this.$route.params.uuid === this.event.uuid
    }
  }
}
</script>
