<template>
    <Tab v-slot="{ selected }" as="template" class="tab">
        <div class="tab__wrap">
            <button class="tab__btn" :class="{'text-muted': !selected}">
                <slot></slot>
            </button>
            <div class="tab__thumb" :class="{'selected': selected}"></div>
        </div>
    </Tab>
</template>

<script>
import {Tab} from '@headlessui/vue'

export default {
    components: {Tab}
}
</script>
