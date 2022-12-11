<template>
    <section class="inspector-cards">
        <div class="inspector-cards__item sm:px-0">
            <h4 class="inspector-cards__item-name">Timestamp</h4>
            <strong class="inspector-cards__item-value md:text-md">{{ event.processDate }}</strong>
        </div>
        <div class="inspector-cards__item pt-5 sm:pt-0">
            <h4 class="inspector-cards__item-name">Duration</h4>
            <strong class="inspector-cards__item-value md:text-md">{{ event.process.duration }} ms</strong>
        </div>
        <div class="inspector-cards__item pt-5 sm:pt-0">
            <h4 class="inspector-cards__item-name">Result</h4>
            <span :class="`bg-${event.color}-700`" class="inspector-cards__item-label">{{ event.processResult }}</span>
        </div>
    </section>
</template>

<script>
export default {
    props: ['event']
}
</script>
