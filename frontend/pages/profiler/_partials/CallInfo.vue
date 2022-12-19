<template>
  <div class="call-information__wrapper" :style="position">
    <h4>{{ edge.name }}</h4>
    <Cards v-if="edge.cost" :cost="edge.cost"/>
  </div>
</template>

<style lang="scss">
.call-information__wrapper {
  @apply bg-gray-800 absolute border border-gray-300 dark:border-gray-600;
  z-index: 9999;

  > h4 {
    @apply px-4 pt-4 pb-0 font-bold truncate;
  }
}
</style>

<script>
import Cards from "@/Components/Events/Profiler/_partials/Cards"

export default {
  components: {Cards},
  props: {
    edge: Object,
    width: {
      type: Number,
      default: 750
    },
    height: {
      type: Number,
      default: 170
    }
  },
  computed: {
    position() {
      if ((this.width + this.edge.position.x) > window.innerWidth - 40) {
        const deltaX = (this.width + this.edge.position.x) - window.innerWidth + 100
        this.edge.position.x -= deltaX;
      }

      if (this.height + this.edge.position.y > window.innerHeight) {
        this.edge.position.y = this.edge.position.y - this.height;
      }

      return {
        top: (this.edge.position.y + 10) + 'px',
        left: this.edge.position.x + 'px',
        width: this.width + 'px'
      }
    }
  }
}
</script>
