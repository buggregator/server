import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryLaravelEventMock from '~/mocks/sentry-laravel.json'
import sentrySpiralEventMock from '~/mocks/sentry-spiral.json'
import TagsComponent from './Tags.vue';

export default {
  title: "Pages/Sentry/Parts/Tags",
  component: TagsComponent
} as Meta<typeof TagsComponent>;

const Template: Story = (args) => ({
  components: { TagsComponent },
  setup() {
    return {
      args,
    };
  },
  template: `<TagsComponent v-bind="args" />`,
});

export const Laravel = Template.bind({});

Laravel.args = {
  event: normalizeSentryEvent(sentryLaravelEventMock).payload,
};

export const Spiral = Template.bind({});

Spiral.args = {
  event: normalizeSentryEvent(sentrySpiralEventMock).payload,
};
