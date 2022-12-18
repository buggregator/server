<template>
    <div class="ws-connection">
        <div class="ws-connection__text">
            <span v-if="connected">Hurry! Give me something. I can't wait to show it.</span>
            <span v-else>Houston we have a problem! Websocket is not connected. Try to reload page.</span>
        </div>
    </div>
</template>

<script>
export default {
    computed: {
      connected() {
        return this.$store.getters['ws/connected']
      }
    }
}
</script>
