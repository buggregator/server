<template>
    <div class="collapsed" :class="{'cursor-pointer': !isOpen}">
        <h3 class="collapsed__title-wrap" @click="isOpen = !isOpen">
            {{ title }}

            <div class="collapsed__icon">
                <svg viewBox="0 0 16 16"
                     fill="currentColor"
                     height="100%" width="100%"
                     :class="{'transform rotate-180': isOpen}"
                >
                    <path d="M14,11.75a.74.74,0,0,1-.53-.22L8,6.06,2.53,11.53a.75.75,0,0,1-1.06-1.06l6-6a.75.75,0,0,1,1.06,0l6,6a.75.75,0,0,1,0,1.06A.74.74,0,0,1,14,11.75Z"></path>
                </svg>
            </div>
        </h3>

        <div v-if="isOpen" class="collapsed__body">
            <slot></slot>
        </div>
    </div>
</template>

<script>
export default {
    props: {
        title: String,
        isOpen: {
            type: Boolean,
            default: false
        }
    }
}
</script>
