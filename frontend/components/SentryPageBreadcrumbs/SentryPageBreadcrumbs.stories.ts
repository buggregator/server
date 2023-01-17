import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryLaravelEventMock from '~/mocks/sentry-laravel.json'
import sentrySpiralEventMock from '~/mocks/sentry-spiral.json'
import SentryPageBreadcrumbs from '~/components/SentryPageBreadcrumbs/SentryPageBreadcrumbs.vue';

export default {
  title: "Sentry/Page/SentryPageBreadcrumbs",
  component: SentryPageBreadcrumbs
} as Meta<typeof SentryPageBreadcrumbs>;

const Template: Story = (args) => ({
  components: { SentryPageBreadcrumbs },
  setup() {
    return {
      args,
    };
  },
  template: `<SentryPageBreadcrumbs v-bind="args" />`,
});

export const Laravel = Template.bind({});

Laravel.args = {
  event: normalizeSentryEvent(sentryLaravelEventMock).payload,
};

export const Spiral = Template.bind({});

Spiral.args = {
  event: normalizeSentryEvent(sentrySpiralEventMock).payload,
};
