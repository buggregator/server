<template>
    <span class="label" :class="color">
        <slot></slot>
    </span>
</template>

<script>
export default {
    props: {
        color: {
            type: String,
            default: 'gray'
        }
    }
}
</script>
