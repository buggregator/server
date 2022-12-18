<template>
  <div class="tips">
    <ul class="tips__list">
      <li class="tips__item">
        <GithubIcon class="tips__icon github"/>
        <a href="https://github.com/buggregator/app" target="_blank" class="tips__link">Github repository</a>
      </li>
      <li class="tips__item">
        <SentryIcon class="tips__icon sentry"/>
        <span>Sentry DSN <a href="https://docs.sentry.io/product/sentry-basics/dsn-explainer/" target="_blank"
                            class="tips__link">{{ sentryDsn }}</a></span>
      </li>
      <li class="tips__item">
        <InspectorIcon class="tips__icon inspector"/>
        <span>Inspector URL <a href="https://docs.inspector.dev/raw-php" target="_blank"
                               class="tips__link">{{ inspectorUrl }}</a></span>
      </li>
      <li class="tips__item">
        <DocsIcon class="tips__icon inspector"/>
        <span>VarDumper URL <a href="https://symfony.com/doc/current/components/var_dumper.html#the-dump-server"
                               target="_blank" class="tips__link">{{ varDumperUrl }}</a></span>
      </li>
      <li class="tips__item">
        <DocsIcon class="tips__icon inspector"/>
        <span>Monolog URL <a href="https://github.com/Seldaek/monolog/blob/main/doc/sockets.md" target="_blank"
                             class="tips__link">{{ monologUrl }}</a></span>
      </li>
    </ul>
  </div>
</template>

<script>
import {computed} from "vue";
import GithubIcon from "@/Components/UI/Icons/GithubIcon";
import SentryIcon from "@/Components/UI/Icons/SentryIcon";
import InspectorIcon from "@/Components/UI/Icons/InspectorIcon";
import DocsIcon from "@/Components/UI/Icons/DocsIcon";

export default {
  components: {DocsIcon, InspectorIcon, SentryIcon, GithubIcon},
  setup() {
    const [host, port] = window.location.host.split(':')

    const sentryDsn = computed(() => `http://sentry@${host}:${port}/1`)
    const inspectorUrl = computed(() => `http://${host}:${port}/inspector`)
    const varDumperUrl = computed(() => `tcp://${host}:9912`)
    const monologUrl = computed(() => `tcp://${host}:9913`)

    return {sentryDsn, inspectorUrl, varDumperUrl, monologUrl}
  }
}
</script>

<style lang="scss">
.tips {
  @apply mt-6 p-6 bg-white dark:bg-gray-900 rounded-l text-gray-600 dark:text-gray-300 border;

  &__title {
    @apply text-xl font-bold mb-3;
  }

  &__list {
    @apply flex flex-col space-y-4;
    @apply flex flex-col space-y-4;
  }

  &__item {
    @apply flex space-x-3 items-center;
  }

  &__link {
    @apply text-blue-600 dark:text-blue-200 underline break-all;
  }

  &__icon {
    @apply w-6 dark:text-white;

    &.github {
      @apply text-gray-800;
    }

    &.sentry {
      @apply text-red-800;
    }

    &.inspector {
      @apply text-blue-800;
    }

    &.ray {
      @apply text-blue-800;
    }
  }
}
</style>
