import { Meta, Story } from "@storybook/vue3";
import { normalizeSentryEvent } from "~/utils/normalize-event";
import sentryLaravelEventMock from '~/mocks/sentry-laravel.json'
import sentrySpiralEventMock from '~/mocks/sentry-spiral.json'
import BreadcrumbsComponent from './Breadcrumbs.vue';

export default {
  title: "Pages/Sentry/Parts/Breadcrumbs",
  component: BreadcrumbsComponent
} as Meta<typeof BreadcrumbsComponent>;

const Template: Story = (args) => ({
  components: { BreadcrumbsComponent },
  setup() {
    return {
      args,
    };
  },
  template: `<BreadcrumbsComponent v-bind="args" />`,
});

export const Laravel = Template.bind({});

Laravel.args = {
  event: normalizeSentryEvent(sentryLaravelEventMock).payload,
};

export const Spiral = Template.bind({});

Spiral.args = {
  event: normalizeSentryEvent(sentrySpiralEventMock).payload,
};
