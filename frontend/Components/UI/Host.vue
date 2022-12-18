<template>
    <div class="host">
        <span class="host__icon">
            <svg fill="currentColor" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 489.8 489.8"><circle cx="245" cy="453.9" r="26.8"/>
                <circle cx="422.8" cy="453.9" r="26.8"/>
                <circle cx="67" cy="453.9" r="26.8"/>
                <path d="M443 31c0-17-14-31-31-31H78C61 0 47 14 47 31v69c0 8 4 15 9 21-5 6-9 13-9 21v70c0 8 4 15 9 21-5 5-9 13-9 21v69c0 17 14 31 31 31h334c17 0 31-14 31-31v-69c0-8-3-16-9-21 6-6 9-13 9-22v-69c0-8-3-16-9-21 6-6 9-13 9-21V31zM65 31c0-7 6-13 12-13h335c7 0 13 6 13 13v69c0 6-6 12-13 12H78c-7 0-13-5-13-12V31zm360 111v70c0 6-6 12-13 12H78c-7 0-13-6-13-12v-70c0-6 6-12 13-12h334c7 0 13 6 13 12zm0 181c0 7-6 13-13 13H78c-7 0-13-6-13-13v-69c0-7 6-12 13-12h334c7 0 13 5 13 12v69z"/><path d="M241 74h113c5 0 9-4 9-9s-4-9-9-9H241c-5 0-9 4-9 9s4 9 9 9z"/><circle cx="148.8" cy="65.1" r="10.8"/><path d="M241 186h113c5 0 9-4 9-9s-4-9-9-9H241c-5 0-9 4-9 9s4 9 9 9zM354 280H241c-5 0-9 4-9 9s4 9 9 9h113c5 0 9-4 9-9s-4-9-9-9z"/><circle cx="148.8" cy="177" r="10.8"/><circle cx="148.8" cy="288.9" r="10.8"/><path d="M423 418c-17 0-31 11-35 27H280c-4-13-13-22-26-26v-28c0-5-4-9-9-9s-9 4-9 9v28c-13 3-22 13-26 26H102a36 36 0 100 18h108a36 36 0 0070 0h108a36 36 0 1035-45zM67 472a18 18 0 010-36 18 18 0 010 36zm178 0a18 18 0 010-36 18 18 0 010 36zm178 0a18 18 0 010-36 18 18 0 010 36z"/></svg>
        </span>
        <span class="host__name">{{ name }}</span>
    </div>
</template>

<script>
import Label from "@/Components/UI/Label";
export default {
    components: {Label},
    props: {
        name: String
    }
}
</script>

<style lang="scss">
.host {
  @apply inline-flex items-center justify-start space-x-2 py-1 text-gray-600 dark:text-gray-300;

  &__icon {
    @apply w-4 h-4;
  }

  &__name {
    @apply flex-initial font-bold text-xs;
  }
}
</style>
