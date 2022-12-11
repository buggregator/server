<template>
    <a :href="href" target="_blank" class="json-chip">JSON</a>
</template>
<script>

export default {
    props: {
        href: String,
    },
}
</script>
