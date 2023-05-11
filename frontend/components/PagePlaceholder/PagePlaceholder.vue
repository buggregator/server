<template>
  <div class="page-placeholder">
    <ul class="page-placeholder__list">
      <li class="page-placeholder__item">
        <IconSvg name="github" class="page-placeholder__icon" />
        <span>
          Bugregator&nbsp;
          <a
            href="https://github.com/buggregator/app"
            target="_blank"
            class="page-placeholder__link"
          >
            Github repository
          </a>
        </span>
      </li>

      <li class="page-placeholder__item">
        <IconSvg
          name="sentry"
          class="page-placeholder__icon page-placeholder__icon--sentry"
        />

        <span>
          Sentry DSN&nbsp;
          <a
            href="https://docs.sentry.io/product/sentry-basics/dsn-explainer/"
            target="_blank"
            class="page-placeholder__link"
          >
            {{ sentryLinkText }}
          </a>
        </span>
      </li>

      <li class="page-placeholder__item">
        <IconSvg
          name="inspector"
          class="page-placeholder__icon page-placeholder__icon--profilers"
        />

        <span>
          Inspector URL&nbsp;
          <a
            href="https://docs.inspector.dev/raw-php"
            target="_blank"
            class="page-placeholder__link"
          >
            {{ inspectorLinkText }}
          </a>
        </span>
      </li>
      <li class="page-placeholder__item">
        <IconSvg
          name="docs"
          class="page-placeholder__icon page-placeholder__icon--profilers"
        />

        <span>
          VarDumper URL&nbsp;
          <a
            href="https://symfony.com/doc/current/components/var_dumper.html#the-dump-server"
            target="_blank"
            class="page-placeholder__link"
          >
            {{ varDumperLinkText }}
          </a>
        </span>
      </li>
      <li class="page-placeholder__item">
        <IconSvg
          name="docs"
          class="page-placeholder__icon page-placeholder__icon--profilers"
        />

        <span>
          Monolog URL&nbsp;
          <a
            href="https://github.com/Seldaek/monolog/blob/main/doc/sockets.md"
            target="_blank"
            class="page-placeholder__link"
          >
            {{ monologLinkText }}
          </a>
        </span>
      </li>
      <li class="page-placeholder__item">
        <IconSvg
          name="profiler"
          class="page-placeholder__icon page-placeholder__icon--profilers"
        />

        <span>
          XHProf URL
          <a
            href="https://github.com/spiral/profiler/tree/3.0"
            target="_blank"
            class="page-placeholder__link"
          >
            {{ XHProfLinkText }}
          </a>
        </span>
      </li>
      <li class="page-placeholder__item">
        <IconSvg
          name="smtp"
          class="page-placeholder__icon page-placeholder__icon--profilers"
        />

        <span>
          SMTP URL
          <a
            href="https://github.com/buggregator/spiral-app#3-fake-smtp-server-for-catching-mail"
            target="_blank"
            class="page-placeholder__link"
          >
            {{ smtpLinkText }}
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

  computed: {
    sentryLinkText() {
      return `http://sentry@${this.getHost()}/1`;
    },
    inspectorLinkText() {
      return `http://${this.getHost()}/api/inspector`;
    },
    varDumperLinkText() {
      return `tcp://${this.getHostName()}:9912`;
    },
    monologLinkText() {
      return `tcp://${this.getHostName()}:9913`;
    },
    XHProfLinkText() {
      return `http://${this.getHost()}/api/profiler/store`;
    },
    smtpLinkText() {
      return `smtp://${this.getHostName()}:1025`;
    },
  },
  methods: {
    getHost() {
      return window?.location?.host || "localhost:3000";
    },
    getHostName() {
      return window?.location?.hostname || "localhost";
    },
  },
});
</script>

<style lang="scss" scoped>
.page-placeholder {
  @apply p-6 bg-white dark:bg-gray-900 rounded-l text-gray-600 dark:text-gray-300 border;
}

.page-placeholder__title {
  @apply text-xl font-bold mb-3;
}

.page-placeholder__list {
  @apply flex flex-col space-y-4;
  @apply flex flex-col space-y-4;
}

.page-placeholder__item {
  @apply flex space-x-3 items-center;
}

.page-placeholder__link {
  @apply text-blue-600 dark:text-blue-200 underline break-all;
}

.page-placeholder__icon {
  @apply w-6 dark:text-white text-gray-800;
}

.page-placeholder__icon--sentry {
  @apply text-red-800;
}

.page-placeholder__icon--profilers {
  @apply text-blue-800;
}
</style>
