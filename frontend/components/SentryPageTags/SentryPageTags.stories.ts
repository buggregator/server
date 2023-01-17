import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryLaravelEventMock from '~/mocks/sentry-laravel.json'
import sentrySpiralEventMock from '~/mocks/sentry-spiral.json'
import SentryPageTags from '~/components/SentryPageTags/SentryPageTags.vue';

export default {
  title: "Sentry/Page/SentryPageTags",
  component: SentryPageTags
} as Meta<typeof SentryPageTags>;

const Template: Story = (args) => ({
  components: { SentryPageTags },
  setup() {
    return {
      args,
    };
  },
  template: `<SentryPageTags v-bind="args" />`,
});

export const Laravel = Template.bind({});

Laravel.args = {
  event: normalizeSentryEvent(sentryLaravelEventMock).payload,
};

export const Spiral = Template.bind({});

Spiral.args = {
  event: normalizeSentryEvent(sentrySpiralEventMock).payload,
};
