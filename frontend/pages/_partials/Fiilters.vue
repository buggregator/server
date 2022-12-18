<template>
  <div class="events-page__filters" v-if="hasLabels">
    <span @click="toggleFilter(label)"
          v-for="label in labels"
          :key="label"
          class="label item"
          :class="{'active': selectedLabels.includes(label)}">
        {{ label }}
    </span>
  </div>
</template>

<style lang="scss">
.events-page__filters {
  @apply flex flex-row flex-wrap gap-2 items-center justify-center;

  > item {
    @apply cursor-pointer;
  }
}
</style>

<script>
import Label from "@/Components/UI/Label"

export default {
  components: {Label},
  methods: {
    toggleFilter(label) {
      console.log(label)
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
