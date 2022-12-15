<template>
    <section class="py-5 px-4 md:px-6 lg:px-8 border-b dark:border-gray-400">
        <h3 class="text-muted font-bold uppercase text-sm mb-5">exceptions</h3>
        <Exception :exception="exception" v-for="exception in exceptions" />
    </section>
</template>

<script>
import File from "../UI/File";
import Exception from "./Exception";

export default {
    components: {File, Exception},
    props: {
        exceptions: Array
    },
}
</script>
