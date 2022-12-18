<template>
  <div class="event" ref="event" :id="event.id" :class="{'collapsed': event.collapsed, 'open': !event.collapsed}">
    <div class="event__sidebar sidebar">
      <div class="event__labels">
        <JsonChip :href="event.route.json"/>
        <Label :color="event.color">
          {{ date }}
        </Label>
<!--        <Label :color="event.color">-->
<!--          {{ event.app }}-->
<!--        </Label>-->
        <Label v-if="hasLabels" v-for="label in labels" :key="label" :color="event.color">
          {{ label }}
        </Label>
      </div>
      <div class="sidebar__container" v-if="!isScreenshot">
        <ImageExport v-if="exportableEl" :name="`${event.app}-${event.id}`" :el="exportableEl"/>
        <button @click="toggle" class="button button__collapse" :class="color">
          <PlusIcon v-if="event.collapsed"/>
          <MinusIcon v-else/>
        </button>
        <button class="button button__delete" @click="deleteEvent">
          <TimesIcon/>
        </button>
      </div>
    </div>
    <div class="event__body" ref="event_body">
      <slot></slot>
    </div>
    <div class="event__origin" v-if="hasOrigin || hasServerName">
      <div class="event__origin-tags">
        <span v-if="hasOrigin && value" v-for="(value, tag) in event.origin"><strong>{{ tag }}: </strong>{{ value }}</span>
      </div>
      <Host v-if="hasServerName" :name="event.serverName" class="event__origin-host"/>
    </div>
  </div>
</template>

<script>
import Label from "@/Components/UI/Label"
import PlusIcon from "@/Components/UI/Icons/PlusIcon"
import MinusIcon from "@/Components/UI/Icons/MinusIcon"
import TimesIcon from "@/Components/UI/Icons/TimesIcon"
import JsonChip from "@/Components/UI/JsonChip"
import ImageExport from "@/Components/UI/ImageExport"
import Host from "@/Components/UI/Host"

export default {
  components: {
    MinusIcon, PlusIcon, TimesIcon, Label, JsonChip, ImageExport, Host
  },
  props: {
    event: Object
  },
  data() {
    return {
      open: true,
      exportableEl: null
    }
  },
  mounted() {
    this.exportableEl = this.$refs.event
  },
  methods: {
    toggle() {
      this.$store.commit('events/toggleCollapsedState', this.event)
    },
    deleteEvent() {
      this.$store.dispatch('events/delete', this.event)
    }
  },
  computed: {
    isScreenshot() {
      return this.$store.getters['theme/isScreenshot']
    },
    date() {
      return this.event.date.format('HH:mm:ss')
    },
    color() {
      const color = this.event.color

      switch (color) {
        case 'gray':
          return 'bg-gray-400 ring-gray-300';
      }

      return `bg-${color}-600 ring-${color}-300`
    },
    labels() {
      return this.event.labels
    },
    hasLabels() {
      return this.labels.length > 0
    },
    hasOrigin() {
      return this.event.origin && Object.entries(this.event.origin).length > 0
    },
    hasServerName() {
      return this.event.serverName !== null
    }
  }
}
</script>

