import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentrySpiralEventMock from '~/mocks/sentry-spiral.json'
import sentryLaravelEventMock from '~/mocks/sentry-laravel.json'
import SentryPreview from '~/components/SentryPreview/SentryPreview.vue';

export default {
  title: "Sentry/Components/SentryPreview",
  component: SentryPreview
} as Meta<typeof SentryPreview>;

const Template: Story = (args) => ({
  components: { SentryPreview },
  setup() {
    return {
      args,
    };
  },
  template: `<SentryPreview v-bind="args" />`,
});

export const Spiral = Template.bind({});

Spiral.args = {
  event: normalizeSentryEvent(sentrySpiralEventMock),
};

export const Laravel = Template.bind({});

Laravel.args = {
  event: normalizeSentryEvent(sentryLaravelEventMock),
};
