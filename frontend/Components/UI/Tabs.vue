<template>
  <div class="flex flex-col">
    <div class="flex justify-start mt-3 border-b border-gray-600">
      <div v-for="(tab, index) in tabs" :key="tab.title" @click="selectTab(index)"
           class="cursor-pointer p-3"
           :class="{'active border-b border-b-4 font-bold': tab.isActive}"
      >{{ tab.title }}
      </div>
    </div>
    <div class="h-full mt-5">
      <slot></slot>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      selectedIndex: 0,
      tabs: []
    }
  },
  mounted () {
    this.selectTab(0)
  },
  methods: {
    selectTab (i) {
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
