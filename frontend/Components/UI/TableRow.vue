<template>
    <div class="event-table__row">
        <div class="event-table__cell-name" v-if="title !== null"><span v-html="title"></span></div>
        <div class="event-table__cell-value">
            <slot></slot>
        </div>
    </div>
</template>

<script>
export default {
    props: {
        title: String
    }
}
</script>
