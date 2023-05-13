import { Meta, Story } from "@storybook/vue3";
import RayDumpPreview from '~/components/RayDumpPreview/RayDumpPreview.vue';
import { normalizeRayDumpEvent } from "~/utils/normalize-event";
import rayIntEventMock from '~/mocks/ray-int.json'
import rayCallerEventMock from '~/mocks/ray-caller.json'
import rayCarbonEventMock from '~/mocks/ray-carbon.json'
import rayColorEventMock from '~/mocks/ray-color.json'
import rayCounterEventMock from '~/mocks/ray-counter.json'
import rayDumpEventMock from '~/mocks/ray-dump.json'
import rayExceptionEventMock from '~/mocks/ray-exception.json'
import rayHideEventMock from '~/mocks/ray-hide.json'
import rayImageEventMock from '~/mocks/ray-image.json'
import rayJsonEventMock from '~/mocks/ray-json.json'
import rayLabelEventMock from '~/mocks/ray-label.json'
import rayMeasureEventMock from '~/mocks/ray-measure.json'
import rayNotifyEventMock from '~/mocks/ray-notify.json'
import raySizeEventMock from '~/mocks/ray-size.json'
import rayTableEventMock from '~/mocks/ray-table.json'
import rayTextEventMock from '~/mocks/ray-text.json'
import rayTraceEventMock from '~/mocks/ray-trace.json'

export default {
  title: "RayDump/RayDumpPreview/Common",
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

export const Text = Template.bind({});

Text.args = {event: normalizeRayDumpEvent(rayTextEventMock),};

export const Trace = Template.bind({});
Trace.args = {event: normalizeRayDumpEvent(rayTraceEventMock),};

export const Table = Template.bind({});
Table.args = {event: normalizeRayDumpEvent(rayTableEventMock),};

export const Size = Template.bind({});
Size.args = {event: normalizeRayDumpEvent(raySizeEventMock),};

export const Notify = Template.bind({});
Notify.args = {event: normalizeRayDumpEvent(rayNotifyEventMock),};

export const Measure = Template.bind({});
Measure.args = {event: normalizeRayDumpEvent(rayMeasureEventMock),};

export const Label = Template.bind({});
Label.args = {event: normalizeRayDumpEvent(rayLabelEventMock),};

export const Json = Template.bind({});
Json.args = {event: normalizeRayDumpEvent(rayJsonEventMock),};

export const Image = Template.bind({});
Image.args = {event: normalizeRayDumpEvent(rayImageEventMock),};

export const Hide = Template.bind({});
Hide.args = {event: normalizeRayDumpEvent(rayHideEventMock),};

export const Exception = Template.bind({});
Exception.args = {event: normalizeRayDumpEvent(rayExceptionEventMock),};

export const Dump = Template.bind({});
Dump.args = {event: normalizeRayDumpEvent(rayDumpEventMock),};

export const Counter = Template.bind({});
Counter.args = {event: normalizeRayDumpEvent(rayCounterEventMock),};

export const Color = Template.bind({});
Color.args = {event: normalizeRayDumpEvent(rayColorEventMock),};

export const Carbon = Template.bind({});
Carbon.args = {event: normalizeRayDumpEvent(rayCarbonEventMock),};

export const Int = Template.bind({});
Int.args = {event: normalizeRayDumpEvent(rayIntEventMock),};

export const Caller = Template.bind({});
Caller.args = {event: normalizeRayDumpEvent(rayCallerEventMock),};
