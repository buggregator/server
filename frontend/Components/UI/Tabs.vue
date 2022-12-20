<template>
  <div class="tabs--container">
    <div class="tabs--wrapper">
      <div class="tab"
           v-for="(tab, index) in tabs" :key="tab.title"
           @click="selectTab(index)"
           :class="{'active': tab.isActive}"
      >{{ tab.title }}
      </div>
    </div>
    <div class="tab--content">
      <slot></slot>
    </div>
  </div>
</template>

<style lang="scss">
.tabs {
  &--wrapper {
    @apply flex justify-start mt-3 border-b border-gray-600;

    .tab {
      @apply cursor-pointer p-3;

      &.active {
        @apply border-b border-b-4 font-bold;
      }

      &--content {
        @apply mt-5;
      }
    }
  }

  &--container {
    @apply flex flex-col;
  }
}
</style>

<script>
export default {
  data() {
    return {
      selectedIndex: 0,
      tabs: []
    }
  },
  mounted() {
    this.selectTab(0)
  },
  methods: {
    selectTab(i) {
      this.selectedIndex = i
      // loop over all the tabs
      this.tabs.forEach((tab, index) => {
        tab.isActive = (index === i)
      })
    }
  },
  created() {
    this.tabs = this.$children
  }
}
</script>
