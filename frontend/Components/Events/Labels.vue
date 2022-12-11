<template>
  <div class="events-page__filters filters" v-if="hasLabels">
    <div class="filters__label-list">
      <Label @click="toggle(label)"
             v-for="label in labels"
             :key="label"
             class="filters__label-item"
             :class="{'active': selectedLabels.includes(label)}"
      >
        {{ label }}
      </Label>
    </div>
  </div>
</template>

<script>
import Label from "../UI/Label"

export default {
  components: {Label},
  methods: {
    toggle(label) {
      this.$store.commit('events/selectLabel', label)
    }
  },
  computed: {
    labels() {
      return this.$store.getters['events/availableLabels']
    },
    selectedLabels() {
      return this.$store.getters['events/selectedLabels']
    },
    hasLabels() {
      return this.labels.length > 0
    }
  }
}
</script>
