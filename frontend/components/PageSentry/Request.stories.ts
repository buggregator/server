import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryLaravelEventMock from '~/mocks/sentry-laravel.json'
import sentrySpiralEventMock from '~/mocks/sentry-spiral.json'
import RequestComponent from './Request.vue';

export default {
  title: "Pages/Sentry/Parts/Request",
  component: RequestComponent
} as Meta<typeof RequestComponent>;

const Template: Story = (args) => ({
  components: { RequestComponent },
  setup() {
    return {
      args,
    };
  },
  template: `<RequestComponent v-bind="args" />`,
});

export const Laravel = Template.bind({});

Laravel.args = {
  event: normalizeSentryEvent(sentryLaravelEventMock).payload,
};

export const Spiral = Template.bind({});

Spiral.args = {
  event: normalizeSentryEvent(sentrySpiralEventMock).payload,
};
