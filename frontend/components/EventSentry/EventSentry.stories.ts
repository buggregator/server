import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import EventSentry from '~/components/EventSentry/EventSentry.vue';
import sentrySpiralEventMock from '~/mocks/sentry-spiral.json'
import sentryLaravelEventMock from '~/mocks/sentry-laravel.json'

export default {
  title: "Sentry",
  component: EventSentry
} as Meta<typeof EventSentry>;

const Template: Story = (args) => ({
  components: { EventSentry },
  setup() {
    return {
      args,
    };
  },
  template: `<event-sentry v-bind="args" />`,
});

export const Spiral = Template.bind({});

Spiral.args = {
  event: normalizeSentryEvent(sentrySpiralEventMock),
};

export const Laravel = Template.bind({});

Laravel.args = {
  event: normalizeSentryEvent(sentryLaravelEventMock),
};
