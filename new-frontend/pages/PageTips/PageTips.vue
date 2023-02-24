<template>
  <div class="page-sidebar">
    <ul class="page-sidebar__list">
      <li class="page-sidebar__item">
        <icon-svg name="github" class="page-sidebar__icon" />

        <span>
          Bugregator&nbsp;
          <a
            href="https://github.com/buggregator/app"
            target="_blank"
            class="page-sidebar__link"
          >
            Github repository
          </a>
        </span>
      </li>

      <li class="page-sidebar__item">
        <icon-svg
          name="sentry"
          class="page-sidebar__icon page-sidebar__icon--sentry"
        />

        <span>
          Sentry DSN&nbsp;
          <a
            href="https://docs.sentry.io/product/sentry-basics/dsn-explainer/"
            target="_blank"
            class="page-sidebar__link"
          >
            {{ `http://sentry@${host}:${port}/1` }}
          </a>
        </span>
      </li>

      <li class="page-sidebar__item">
        <icon-svg
          name="inspector"
          class="page-sidebar__icon page-sidebar__icon--profilers"
        />

        <span>
          Inspector URL&nbsp;
          <a
            href="https://docs.inspector.dev/raw-php"
            target="_blank"
            class="page-sidebar__link"
          >
            {{ `http://${host}:${port}/inspector` }}
          </a>
        </span>
      </li>
      <li class="page-sidebar__item">
        <icon-svg
          name="docs"
          class="page-sidebar__icon page-sidebar__icon--profilers"
        />

        <span>
          VarDumper URL&nbsp;
          <a
            href="https://symfony.com/doc/current/components/var_dumper.html#the-dump-server"
            target="_blank"
            class="page-sidebar__link"
          >
            {{ `tcp://${host}:9912` }}
          </a>
        </span>
      </li>
      <li class="page-sidebar__item">
        <icon-svg
          name="docs"
          class="page-sidebar__icon page-sidebar__icon--profilers"
        />

        <span>
          Monolog URL&nbsp;
          <a
            href="https://github.com/Seldaek/monolog/blob/main/doc/sockets.md"
            target="_blank"
            class="page-sidebar__link"
          >
            {{ `tcp://${host}:9913` }}
          </a>
        </span>
      </li>
    </ul>
  </div>
</template>

<script lang="ts">
import IconSvg from "~/components/IconSvg/IconSvg.vue";
import { defineComponent } from "vue";

export default defineComponent({
  components: { IconSvg },
  setup() {
    const [host, port] = window.location.host.split(":");

    return {
      host,
      port: port || 80,
    };
  },
});
</script>

<style lang="scss" scoped>
.page-sidebar {
  @apply mt-6 p-6 bg-white dark:bg-gray-900 rounded-l text-gray-600 dark:text-gray-300 border;
}

.page-sidebar__title {
  @apply text-xl font-bold mb-3;
}

.page-sidebar__list {
  @apply flex flex-col space-y-4;
  @apply flex flex-col space-y-4;
}

.page-sidebar__item {
  @apply flex space-x-3 items-center;
}

.page-sidebar__link {
  @apply text-blue-600 dark:text-blue-200 underline break-all;
}

.page-sidebar__icon {
  @apply w-6 dark:text-white text-gray-800;
}

.page-sidebar__icon--sentry {
  @apply text-red-800;
}

.page-sidebar__icon--profilers {
  @apply text-blue-800;
}
</style>
