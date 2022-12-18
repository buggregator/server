<template>
    <a :href="href" target="_blank" class="json-chip">JSON</a>
</template>

<style lang="scss">
.json-chip {
  @apply font-bold px-2 rounded-full text-2xs inline-flex items-center transition-colors text-blue-700 bg-blue-100 border hover:bg-blue-700 hover:text-white;
}
</style>

<script>
export default {
    props: {
        href: String,
    },
}
</script>
