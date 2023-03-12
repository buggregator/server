<template>
  <NuxtLayout>
    <div class="error">
      <h1 class="error__title">
        404
        <br />
        Page not found
      </h1>

      <button class="error__go-to-home" @click="handleError">Go to home</button>
    </div>
  </NuxtLayout>
</template>

<script lang="ts" setup>
import "./assets/tailwind.css";
import "highlight.js/lib/common";
import { clearError } from "#app";

const handleError = () => clearError({ redirect: "/" });
</script>

<style lang="scss" scoped>
.error {
  @apply p-6 rounded-l text-gray-600 dark:text-gray-300 text-center h-full w-full flex flex-col justify-between items-center m-auto;
}

.error__title {
  font-size: 100px;
}

.error__go-to-home {
}
</style>
