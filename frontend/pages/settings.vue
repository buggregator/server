<template>
  <main class="settings-page">
    <PageHeader>
      <NuxtLink to="/">Home</NuxtLink>&nbsp; /&nbsp;Settings
    </PageHeader>

    <section class="settings-page__content">
      <div class="settings-page__radio">
        <IconSvg
          name="sun"
          class="settings-page__radio-icon"
          :class="{ 'settings-page__radio-icon--dark': isDarkMode }"
        />

        <button
          class="settings-page__radio-button"
          :class="{ 'settings-page__radio-button--dark': isDarkMode }"
          @click="changeTheme()"
        >
          <span class="settings-page__radio-button-in" />
        </button>

        <IconSvg
          class="settings-page__radio-icon"
          name="moon"
          :class="{ 'settings-page__radio-icon--dark': !isDarkMode }"
        />
      </div>
    </section>
  </main>
</template>

<script lang="ts">
import { defineComponent } from "vue";
import { storeToRefs } from "pinia";
import IconSvg from "~/components/IconSvg/IconSvg.vue";
import PageHeader from "~/components/PageHeader/PageHeader.vue";
import { useThemeStore, THEME_MODES } from "~/stores/theme";

export default defineComponent({
  components: {
    IconSvg,
    PageHeader,
  },
  setup() {
    const themeStore = useThemeStore();
    const { themeChange } = themeStore;
    const { themeType } = storeToRefs(themeStore);

    return {
      themeType,
      themeChange,
    };
  },
  head: {
    title: "Settings | Buggregator",
  },
  computed: {
    isDarkMode() {
      return this.themeType === THEME_MODES.DARK;
    },
  },
  mounted() {
    if (this.isDarkMode) {
      document?.documentElement?.classList?.add(THEME_MODES.DARK);
    } else {
      document?.documentElement?.classList?.remove(THEME_MODES.DARK);
    }
  },
  methods: {
    changeTheme() {
      if (this.isDarkMode) {
        document?.documentElement?.classList?.remove(THEME_MODES.DARK);
      } else {
        document?.documentElement?.classList?.add(THEME_MODES.DARK);
      }

      return this.themeChange();
    },
  },
});
</script>

<style lang="scss" scoped>
.settings-page {
}

.settings-page__content {
  @apply p-3;
}

.settings__title {
  @apply text-2xl font-bold;
}

.settings-page__radio {
  @apply flex space-x-5 items-center my-5;
}

.settings-page__radio-icon {
  @apply opacity-100 w-8;
}

.settings-page__radio-icon--dark {
  @apply opacity-10;
}

.settings-page__radio-button {
  @apply relative inline-flex h-8 w-16 items-center rounded-full bg-gray-200;
}

.settings-page__radio-button-in {
  @apply inline-block h-6 w-6 transform rounded-full transition bg-blue-600 translate-x-2;

  .settings-page__radio-button--dark & {
    @apply translate-x-8;
  }
}
</style>
