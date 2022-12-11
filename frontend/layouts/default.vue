<template>
  <div class="main-layout">
    <div class="main-layout__sidebar-wrap">
      <LeftSidebar class="main-layout__sidebar">
        <slot name="left-sidebar"></slot>
      </LeftSidebar>
    </div>

    <div class="main-layout__content">
      <Nuxt/>
    </div>
  </div>
</template>

<script>
import LeftSidebar from './Sidebar/Left'

export default {
  components: {LeftSidebar}
}
</script>
