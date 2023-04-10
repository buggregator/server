import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryLaravelEventMock from '~/mocks/sentry-laravel.json'
import sentrySpiralEventMock from '~/mocks/sentry-spiral.json'
import SentryPageRequest from '~/components/SentryPageRequest/SentryPageRequest.vue';

export default {
  title: "Sentry/Page/SentryPageRequest",
  component: SentryPageRequest
} as Meta<typeof SentryPageRequest>;

const Template: Story = (args) => ({
  components: { SentryPageRequest },
  setup() {
    return {
      args,
    };
  },
  template: `<SentryPageRequest v-bind="args" />`,
});

export const Laravel = Template.bind({});

Laravel.args = {
  event: normalizeSentryEvent(sentryLaravelEventMock).payload,
};

export const Spiral = Template.bind({});

Spiral.args = {
  event: normalizeSentryEvent(sentrySpiralEventMock).payload,
};
