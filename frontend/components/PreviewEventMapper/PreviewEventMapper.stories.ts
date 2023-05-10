import { Meta, Story } from "@storybook/vue3";
import PreviewEventMapper from '~/components/PreviewEventMapper/PreviewEventMapper.vue';
import monologEventMock from '~/mocks/monolog.json'
import sentryEventMock from '~/mocks/sentry-spiral.json'
import smtpEventMock from '~/mocks/smtp-welcome.json'
import varDumpEventMock from '~/mocks/var-dump-object.json'
import profilerEventMock from '~/mocks/profiler.json'
import inspectorEventMock from '~/mocks/inspector.json'
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
  title: "Preview/PreviewEventMapper",
  component: PreviewEventMapper
} as Meta<typeof PreviewEventMapper>;

const Template: Story = (args) => ({
  components: { PreviewEventMapper },
  setup() {
    return {
      args,
    };
  },
  template: `<PreviewEventMapper v-bind="args" />`,
});

export const Default = Template.bind({});

Default.args = {
  event: { ...smtpEventMock, type: 'unknown' },
};

export const Monolog = Template.bind({});

Monolog.args = {
  event: monologEventMock,
};

export const Sentry = Template.bind({});

Sentry.args = {
  event: sentryEventMock,
};

export const Smtp = Template.bind({});

Smtp.args = {
  event: smtpEventMock,
};

export const VarDump = Template.bind({});

VarDump.args = {
  event: varDumpEventMock,
};

export const Profiler = Template.bind({});

Profiler.args = {
  event: profilerEventMock,
};

export const Inspector = Template.bind({});

Inspector.args = {
  event: inspectorEventMock,
};

export const RayTrace = Template.bind({});
RayTrace.args = {event: rayTraceEventMock,};

export const RayText = Template.bind({});
RayText.args = {event: rayTextEventMock,};

export const RayTable = Template.bind({});
RayTable.args = {event: rayTableEventMock,};

export const RaySize = Template.bind({});
RaySize.args = {event: raySizeEventMock,};

export const RayNotify = Template.bind({});
RayNotify.args = {event: rayNotifyEventMock,};

export const RayMeasure = Template.bind({});
RayMeasure.args = {event: rayMeasureEventMock,};

export const RayLabel = Template.bind({});
RayLabel.args = {event: rayLabelEventMock,};

export const RayJson = Template.bind({});
RayJson.args = {event: rayJsonEventMock,};

export const RayImage = Template.bind({});
RayImage.args = {event: rayImageEventMock,};

export const RayHide = Template.bind({});
RayHide.args = {event: rayHideEventMock,};

export const RayException = Template.bind({});
RayException.args = {event: rayExceptionEventMock,};

export const RayDump = Template.bind({});
RayDump.args = {event: rayDumpEventMock,};

export const RayCounter = Template.bind({});
RayCounter.args = {event: rayCounterEventMock,};

export const RayColor = Template.bind({});
RayColor.args = {event: rayColorEventMock,};

export const RayCarbon = Template.bind({});
RayCarbon.args = {event: rayCarbonEventMock,};

export const RayInt = Template.bind({});
RayInt.args = {event: rayIntEventMock,};

export const RayCaller = Template.bind({});
RayCaller.args = {event: rayCallerEventMock,};

const TemplateList: Story = (args) => ({
  components: { PreviewEventMapper },
  setup() {

    return {
      args,
      eventsList: [monologEventMock,sentryEventMock,smtpEventMock,varDumpEventMock,profilerEventMock,inspectorEventMock, { ...smtpEventMock, type: 'unknown' }]
    };
  },
  template: `<PreviewEventMapper class="border-b" v-for="event in eventsList" :event="event" :key="event.uuid"/>`,
});


export const EventsList = TemplateList.bind({});

EventsList.args = {
  event: inspectorEventMock,
};
