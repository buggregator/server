import { Meta, Story } from "@storybook/vue3";
import MeasurePayload from '~/components/RayDumpPreview/RayTypesPreview/MeasurePayload.vue';
import { normalizeRayDumpEvent } from "~/utils/normalize-event";
import eventMock from '~/mocks/ray-measure.json';
import eventStartMock from '~/mocks/ray-measure-start.json';

export default {
  title: "RayDump/Types/Measure",
  component: MeasurePayload
} as Meta<typeof MeasurePayload>;

const Template: Story = (args) => ({
  components: { MeasurePayload },
  setup() {
    return {
      args,
    };
  },
  template: `<MeasurePayload v-bind="args" />`,
});

export const Default = Template.bind({});
Default.args = {payload: normalizeRayDumpEvent(eventMock).payload.payload.payloads[0]};

export const Start = Template.bind({});
Start.args = {payload: normalizeRayDumpEvent(eventStartMock).payload.payload.payloads[0]};
