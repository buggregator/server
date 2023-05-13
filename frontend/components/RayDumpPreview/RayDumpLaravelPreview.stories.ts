import { Meta, Story } from "@storybook/vue3";
import RayDumpPreview from '~/components/RayDumpPreview/RayDumpPreview.vue';
import { normalizeRayDumpEvent } from "~/utils/normalize-event";
import rayQueryEventMock from '~/mocks/ray-laravel-query.json'
import rayEloquentEventMock from '~/mocks/ray-laravel-eloquent.json'
import rayViewsEventMock from '~/mocks/ray-laravel-views.json'
import rayEventsEventMock from '~/mocks/ray-laravel-events.json'
import rayJobsEventMock from '~/mocks/ray-laravel-jobs.json'

export default {
  title: "RayDump/RayDumpPreview/Laravel",
  component: RayDumpPreview
} as Meta<typeof RayDumpPreview>;

const Template: Story = (args) => ({
  components: { RayDumpPreview },
  setup() {
    return {
      args,
    };
  },
  template: `<RayDumpPreview v-bind="args" />`,
});

export const Query = Template.bind({});
Query.args = {event: normalizeRayDumpEvent(rayQueryEventMock),};

export const Eloquent = Template.bind({});
Eloquent.args = {event: normalizeRayDumpEvent(rayEloquentEventMock),};

export const Views = Template.bind({});
Views.args = {event: normalizeRayDumpEvent(rayViewsEventMock),};

export const Events = Template.bind({});
Events.args = {event: normalizeRayDumpEvent(rayEventsEventMock),};

export const Jobs = Template.bind({});
Jobs.args = {event: normalizeRayDumpEvent(rayJobsEventMock),};
