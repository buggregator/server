<template>
  <NavItem :event="event"/>
</template>

<script>
import NavItem from "../NavItem"

export default {
  components: {NavItem},
  props: {
    event: Object
  }
}
</script>
